CREATE TABLE conversation_feedback (
    id VARCHAR(36) PRIMARY KEY,
    session_id VARCHAR(36) NOT NULL UNIQUE REFERENCES sessions(id) ON DELETE CASCADE,
    tenant_id INTEGER NOT NULL,
    user_id VARCHAR(512) NOT NULL,
    helpful BOOLEAN,
    category VARCHAR(16),
    comment TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (category IS NULL OR category IN ('suggestion', 'complaint')),
    CHECK (length(comment) <= 2000)
);
CREATE INDEX idx_conversation_feedback_tenant_updated ON conversation_feedback (tenant_id, updated_at DESC);
