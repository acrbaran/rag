CREATE TABLE conversation_feedback (
    id VARCHAR(36) PRIMARY KEY,
    session_id VARCHAR(36) NOT NULL UNIQUE REFERENCES sessions(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(512) NOT NULL,
    helpful BOOLEAN,
    category VARCHAR(16),
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT conversation_feedback_category_check CHECK (category IS NULL OR category IN ('suggestion', 'complaint')),
    CONSTRAINT conversation_feedback_comment_length CHECK (char_length(comment) <= 2000)
);
CREATE INDEX idx_conversation_feedback_tenant_updated ON conversation_feedback (tenant_id, updated_at DESC);
