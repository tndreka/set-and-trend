package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"set-and-trend/backend/internal/db"
)

type FeedbackRepository struct {
	q *db.Queries
}

func NewFeedbackRepository(q *db.Queries) *FeedbackRepository {
	return &FeedbackRepository{q: q}
}

// TradeFeedback represents the API response format
type TradeFeedback struct {
	ID             uuid.UUID `json:"id"`
	TradeID        uuid.UUID `json:"trade_id"`
	FollowedPlan   bool      `json:"followed_plan"`
	EmotionBefore  string    `json:"emotion_before"`
	EmotionDuring  string    `json:"emotion_during"`
	EmotionAfter   string    `json:"emotion_after"`
	BiggestMistake *string   `json:"biggest_mistake,omitempty"`
	ScreenshotURL  *string   `json:"screenshot_url,omitempty"`
	FeedbackAt     time.Time `json:"feedback_at"`
}

// FeedbackCreateParams contains parameters for creating feedback
type FeedbackCreateParams struct {
	TradeID        uuid.UUID
	FollowedPlan   bool
	EmotionBefore  string
	EmotionDuring  string
	EmotionAfter   string
	BiggestMistake *string
	ScreenshotURL  *string
}

// CreateFeedback creates a new trade feedback entry
func (r *FeedbackRepository) CreateFeedback(ctx context.Context, params FeedbackCreateParams) (*TradeFeedback, error) {
	// Convert string pointers to pgtype.Text
	var biggestMistake, screenshotURL pgtype.Text
	if params.BiggestMistake != nil && *params.BiggestMistake != "" {
		biggestMistake = pgtype.Text{String: *params.BiggestMistake, Valid: true}
	}
	if params.ScreenshotURL != nil && *params.ScreenshotURL != "" {
		screenshotURL = pgtype.Text{String: *params.ScreenshotURL, Valid: true}
	}

	dbFeedback, err := r.q.CreateTradeFeedback(ctx, db.CreateTradeFeedbackParams{
		TradeID:        params.TradeID,
		FollowedPlan:   params.FollowedPlan,
		EmotionBefore:  db.EmotionType(params.EmotionBefore),
		EmotionDuring:  db.EmotionType(params.EmotionDuring),
		EmotionAfter:   db.EmotionType(params.EmotionAfter),
		BiggestMistake: biggestMistake,
		ScreenshotUrl:  screenshotURL,
	})
	if err != nil {
		return nil, err
	}

	return mapDBFeedbackToFeedback(dbFeedback), nil
}

// GetFeedbackByTradeID retrieves feedback for a specific trade
func (r *FeedbackRepository) GetFeedbackByTradeID(ctx context.Context, tradeID uuid.UUID) (*TradeFeedback, error) {
	dbFeedback, err := r.q.GetTradeFeedbackByTradeID(ctx, tradeID)
	if err != nil {
		return nil, err
	}

	return mapDBFeedbackToFeedback(dbFeedback), nil
}

// UpdateFeedback updates an existing feedback entry
func (r *FeedbackRepository) UpdateFeedback(ctx context.Context, params FeedbackCreateParams) (*TradeFeedback, error) {
	var biggestMistake, screenshotURL pgtype.Text
	if params.BiggestMistake != nil && *params.BiggestMistake != "" {
		biggestMistake = pgtype.Text{String: *params.BiggestMistake, Valid: true}
	}
	if params.ScreenshotURL != nil && *params.ScreenshotURL != "" {
		screenshotURL = pgtype.Text{String: *params.ScreenshotURL, Valid: true}
	}

	dbFeedback, err := r.q.UpdateTradeFeedback(ctx, db.UpdateTradeFeedbackParams{
		TradeID:        params.TradeID,
		FollowedPlan:   params.FollowedPlan,
		EmotionBefore:  db.EmotionType(params.EmotionBefore),
		EmotionDuring:  db.EmotionType(params.EmotionDuring),
		EmotionAfter:   db.EmotionType(params.EmotionAfter),
		BiggestMistake: biggestMistake,
		ScreenshotUrl:  screenshotURL,
	})
	if err != nil {
		return nil, err
	}

	return mapDBFeedbackToFeedback(dbFeedback), nil
}

// DeleteFeedback deletes feedback for a trade
func (r *FeedbackRepository) DeleteFeedback(ctx context.Context, tradeID uuid.UUID) error {
	return r.q.DeleteTradeFeedback(ctx, tradeID)
}

// mapDBFeedbackToFeedback converts DB model to API response
func mapDBFeedbackToFeedback(dbFeedback db.TradeFeedback) *TradeFeedback {
	var biggestMistake, screenshotURL *string
	if dbFeedback.BiggestMistake.Valid {
		biggestMistake = &dbFeedback.BiggestMistake.String
	}
	if dbFeedback.ScreenshotUrl.Valid {
		screenshotURL = &dbFeedback.ScreenshotUrl.String
	}

	return &TradeFeedback{
		ID:             dbFeedback.ID,
		TradeID:        dbFeedback.TradeID,
		FollowedPlan:   dbFeedback.FollowedPlan,
		EmotionBefore:  string(dbFeedback.EmotionBefore),
		EmotionDuring:  string(dbFeedback.EmotionDuring),
		EmotionAfter:   string(dbFeedback.EmotionAfter),
		BiggestMistake: biggestMistake,
		ScreenshotURL:  screenshotURL,
		FeedbackAt:     dbFeedback.FeedbackAt.Time,
	}
}
