DROP INDEX IF EXISTS idx_document_embeddings_vector;
DROP INDEX IF EXISTS idx_sync_jobs_status_scheduled;
DROP INDEX IF EXISTS idx_document_permissions_doc_group;
DROP INDEX IF EXISTS idx_document_permissions_doc_user;
DROP INDEX IF EXISTS idx_documents_connector;
DROP INDEX IF EXISTS idx_documents_source;

DROP TABLE IF EXISTS ai_config;
DROP TABLE IF EXISTS sync_jobs;
DROP TABLE IF EXISTS document_permissions;
DROP TABLE IF EXISTS document_embeddings;
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS connector_configs;
DROP TABLE IF EXISTS user_groups;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS users;

DROP EXTENSION IF EXISTS vector;
