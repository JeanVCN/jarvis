-- Generation completion and playback are separate: audio may be interrupted later.
ALTER TABLE conversation_turns ADD COLUMN IF NOT EXISTS delivery_status text NOT NULL DEFAULT 'pending';
