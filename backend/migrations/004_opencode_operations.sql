CREATE TABLE IF NOT EXISTS opencode_sessions (
    id text PRIMARY KEY CHECK (id ~ '^[a-f0-9]{32}$'),
    project_id text NOT NULL CHECK (project_id ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    external_id text,
    title text NOT NULL DEFAULT 'Nova sessão',
    state text NOT NULL DEFAULT 'creating' CHECK (state IN ('creating', 'ready', 'uncertain', 'failed')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(project_id, external_id)
);

CREATE TABLE IF NOT EXISTS opencode_operations (
    id text PRIMARY KEY CHECK (id ~ '^[a-f0-9]{32}$'),
    project_id text NOT NULL CHECK (project_id ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    session_id text NOT NULL REFERENCES opencode_sessions(id),
    kind text NOT NULL CHECK (kind IN ('create_session', 'prompt', 'permission')),
    input text NOT NULL DEFAULT '',
    state text NOT NULL CHECK (state IN ('submitting', 'accepted', 'running', 'completed', 'failed', 'uncertain')),
    external_id text,
    error_code text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS opencode_sessions_project_updated_idx
    ON opencode_sessions(project_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS opencode_operations_session_created_idx
    ON opencode_operations(session_id, created_at);
