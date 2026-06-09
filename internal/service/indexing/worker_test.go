package indexing

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/connector"
	"github.com/michaelahli/nexd/internal/service/embedding"
)

type fakeDocumentRepository struct {
	documentID uuid.UUID
	docs       []Document
	chunks     []EmbeddingChunk
}

func (r *fakeDocumentRepository) UpsertDocument(ctx context.Context, doc Document) (uuid.UUID, error) {
	r.docs = append(r.docs, doc)
	return r.documentID, nil
}

func (r *fakeDocumentRepository) ReplaceEmbeddings(ctx context.Context, documentID uuid.UUID, chunks []EmbeddingChunk) error {
	r.chunks = append([]EmbeddingChunk(nil), chunks...)
	return nil
}

type fakePermissionSyncer struct {
	documentID uuid.UUID
	targets    []PermissionTarget
}

func (s *fakePermissionSyncer) SyncDocumentPermissions(ctx context.Context, documentID uuid.UUID, targets []PermissionTarget) error {
	s.documentID = documentID
	s.targets = append([]PermissionTarget(nil), targets...)
	return nil
}

type fakeDocumentSource struct {
	docs []connector.Document
	err  error
}

func (s *fakeDocumentSource) Documents(ctx context.Context, job Job) (<-chan connector.Document, <-chan error) {
	docs := make(chan connector.Document, len(s.docs))
	for _, doc := range s.docs {
		docs <- doc
	}
	close(docs)

	errs := make(chan error, 1)
	if s.err != nil {
		errs <- s.err
	}
	close(errs)
	return docs, errs
}

type fakeEmbedder struct {
	calls int
}

func (e *fakeEmbedder) EmbedText(ctx context.Context, text string) ([]embedding.ChunkEmbedding, error) {
	e.calls++
	return []embedding.ChunkEmbedding{{Index: 0, Text: text, Vector: embedding.Vector{1, 2, 3}}}, nil
}

func TestProcessorProcessesDocuments(t *testing.T) {
	documentID := uuid.New()
	userID := uuid.New()
	documents := &fakeDocumentRepository{documentID: documentID}
	permissions := &fakePermissionSyncer{}
	embedder := &fakeEmbedder{}
	processor := NewProcessor(ProcessorOptions{
		Documents:   documents,
		Permissions: permissions,
		Source: &fakeDocumentSource{docs: []connector.Document{{
			SourceType:  "smb",
			SourceID:    "file-1",
			Title:       "File 1",
			Content:     "hello world",
			Permissions: []connector.PermissionTarget{{UserID: &userID, PermissionType: "read"}},
		}}},
		Embedder: embedder,
	})

	processed, err := processor.Process(context.Background(), Job{ConnectorID: uuid.New()})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected one processed document, got %d", processed)
	}
	if len(documents.docs) != 1 || documents.docs[0].SourceID != "file-1" {
		t.Fatalf("unexpected documents: %#v", documents.docs)
	}
	if len(documents.chunks) != 1 || documents.chunks[0].Text != "hello world" {
		t.Fatalf("unexpected chunks: %#v", documents.chunks)
	}
	if permissions.documentID != documentID || len(permissions.targets) != 1 || permissions.targets[0].UserID == nil || *permissions.targets[0].UserID != userID {
		t.Fatalf("unexpected permission sync: id=%s targets=%#v", permissions.documentID, permissions.targets)
	}
	if embedder.calls != 1 {
		t.Fatalf("expected one embed call, got %d", embedder.calls)
	}
}

func TestProcessorReturnsSourceErrors(t *testing.T) {
	processor := NewProcessor(ProcessorOptions{
		Documents: &fakeDocumentRepository{documentID: uuid.New()},
		Source:    &fakeDocumentSource{err: errors.New("source down")},
		Embedder:  &fakeEmbedder{},
	})

	_, err := processor.Process(context.Background(), Job{ConnectorID: uuid.New()})
	if err == nil {
		t.Fatal("expected source error")
	}
}

type fakeQueue struct {
	mu         sync.Mutex
	jobs       []Job
	completed  []uuid.UUID
	failed     []uuid.UUID
	failNext   bool
	onComplete chan struct{}
}

func (q *fakeQueue) Next(ctx context.Context) (Job, bool, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.failNext {
		return Job{}, false, errors.New("queue failed")
	}
	if len(q.jobs) == 0 {
		return Job{}, false, nil
	}
	job := q.jobs[0]
	q.jobs = q.jobs[1:]
	return job, true, nil
}

func (q *fakeQueue) Complete(ctx context.Context, jobID uuid.UUID, documentsProcessed int) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.completed = append(q.completed, jobID)
	if q.onComplete != nil {
		close(q.onComplete)
		q.onComplete = nil
	}
	return nil
}

func (q *fakeQueue) Fail(ctx context.Context, jobID uuid.UUID, jobErr error) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.failed = append(q.failed, jobID)
	return nil
}

func TestServiceCompletesJobs(t *testing.T) {
	jobID := uuid.New()
	completeCh := make(chan struct{})
	queue := &fakeQueue{jobs: []Job{{ID: jobID, ConnectorID: uuid.New()}}, onComplete: completeCh}
	processor := NewProcessor(ProcessorOptions{
		Documents: &fakeDocumentRepository{documentID: uuid.New()},
		Source:    &fakeDocumentSource{docs: []connector.Document{{SourceID: "1", Content: "hello"}}},
		Embedder:  &fakeEmbedder{},
	})
	service := NewService(queue, processor, Config{Workers: 1, PollInterval: time.Millisecond})
	service.sleep = func(ctx context.Context, d time.Duration) error { return ctx.Err() }

	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	select {
	case <-completeCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for job completion")
	}
	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if len(queue.completed) != 1 || queue.completed[0] != jobID {
		t.Fatalf("expected completed job %s, got %#v", jobID, queue.completed)
	}
}
