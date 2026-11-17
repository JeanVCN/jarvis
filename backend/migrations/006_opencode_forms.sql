ALTER TABLE opencode_operations
    DROP CONSTRAINT IF EXISTS opencode_operations_kind_check;

ALTER TABLE opencode_operations
    ADD CONSTRAINT opencode_operations_kind_check
    CHECK (kind IN ('create_session', 'prompt', 'permission', 'interrupt', 'form'));
