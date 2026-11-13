CREATE TABLE IF NOT EXISTS conversations (
 id text PRIMARY KEY, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS conversation_turns (
 id text PRIMARY KEY, conversation_id text NOT NULL REFERENCES conversations(id),
 input text NOT NULL, provider text NOT NULL, model text NOT NULL DEFAULT '',
 status text NOT NULL DEFAULT 'running', generated text NOT NULL DEFAULT '',
 played_segment integer NOT NULL DEFAULT -1, played_ms integer NOT NULL DEFAULT 0,
 created_at timestamptz NOT NULL DEFAULT now(), finished_at timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS one_running_turn ON conversation_turns(conversation_id) WHERE status='running';
CREATE TABLE IF NOT EXISTS conversation_events (
 seq bigserial PRIMARY KEY, turn_id text NOT NULL REFERENCES conversation_turns(id),
 type text NOT NULL, data jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS conversation_events_turn ON conversation_events(turn_id,seq);
CREATE TABLE IF NOT EXISTS conversation_tools (
 turn_id text NOT NULL REFERENCES conversation_turns(id), call_id text NOT NULL,
 name text NOT NULL, input text NOT NULL, status text NOT NULL, result text NOT NULL DEFAULT '',
 PRIMARY KEY(turn_id,call_id)
);
