-- Add emotion_type enum and trade_feedback table

CREATE TYPE emotion_type AS ENUM ('calm', 'anxious', 'fomo', 'revenge', 'other');

CREATE TABLE trade_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trade_id UUID NOT NULL UNIQUE REFERENCES trades(id) ON DELETE CASCADE,
    followed_plan BOOLEAN NOT NULL,
    emotion_before emotion_type NOT NULL,
    emotion_during emotion_type NOT NULL,
    emotion_after emotion_type NOT NULL,
    biggest_mistake TEXT,
    screenshot_url TEXT,
    feedback_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trade_feedback_trade_id ON trade_feedback(trade_id);
