package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/michaelahli/nexd/internal/repository"
)

type fakeUsersRepo struct{}

func (f *fakeUsersRepo) List(ctx context.Context) ([]repository.UserRecord, error) {
	return []repository.UserRecord{{ID: uuid.New(), Email: "user@example.com", Name: "User", IsActive: true}}, nil
}
func (f *fakeUsersRepo) Update(ctx context.Context, user repository.UserRecord) error { return nil }
func (f *fakeUsersRepo) Delete(ctx context.Context, userID uuid.UUID) error           { return nil }

func TestUsersList(t *testing.T) {
	h := NewUsers(&fakeUsersRepo{})
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	w := httptest.NewRecorder()

	h.List(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestUsersUpdate(t *testing.T) {
	h := NewUsers(&fakeUsersRepo{})
	body := `{"id":"` + uuid.NewString() + `","email":"user@example.com","name":"User","is_active":true}`
	req := httptest.NewRequest(http.MethodPut, "/admin/users", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.Update(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
