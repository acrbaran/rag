CREATE TABLE announcement_types (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(60) NOT NULL,
    normalized_name VARCHAR(240) NOT NULL UNIQUE,
    created_by VARCHAR(36) NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE announcements (
    id VARCHAR(36) PRIMARY KEY,
    status VARCHAR(16) NOT NULL DEFAULT 'draft',
    version INTEGER NOT NULL DEFAULT 1,
    type_id VARCHAR(36) NOT NULL DEFAULT 'maintenance',
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    starts_at INTEGER,
    ends_at INTEGER,
    show_banner BOOLEAN NOT NULL DEFAULT TRUE,
    banner_color VARCHAR(16) NOT NULL DEFAULT 'neutral',
    audience_mode VARCHAR(16) NOT NULL DEFAULT 'all',
    audience_ids TEXT NOT NULL DEFAULT '[]',
    created_by VARCHAR(36) NOT NULL,
    published_by VARCHAR(36),
    published_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    CHECK (status IN ('draft', 'maintenance', 'resolved', 'withdrawn')),
    CHECK (audience_mode IN ('all', 'workspaces', 'roles', 'users')),
    CHECK (banner_color IN ('neutral', 'blue', 'green', 'amber', 'red', 'violet')),
    CHECK (starts_at IS NULL OR ends_at IS NULL OR ends_at > starts_at)
);
CREATE INDEX idx_announcements_status_published ON announcements (status, published_at DESC);
CREATE INDEX idx_announcements_created ON announcements (created_at DESC);

CREATE TABLE announcement_notification_states (
    id VARCHAR(36) PRIMARY KEY,
    announcement_id VARCHAR(36) NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
    user_id VARCHAR(36) NOT NULL,
    phase VARCHAR(16) NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    dismissed_version INTEGER,
    first_seen_at INTEGER,
    CHECK (phase IN ('published', 'resolved')),
    UNIQUE (announcement_id, user_id, phase)
);
CREATE INDEX idx_announcement_notification_states_user ON announcement_notification_states (user_id);

CREATE TABLE announcement_idempotency (
    id VARCHAR(36) PRIMARY KEY,
    actor_id VARCHAR(36) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    action VARCHAR(16) NOT NULL,
    request_hash VARCHAR(64) NOT NULL,
    resource_id VARCHAR(36) NOT NULL,
    created_at INTEGER NOT NULL,
    UNIQUE (actor_id, idempotency_key)
);
