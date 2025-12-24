package repositories

import (
	"context"
	
	"github.com/google/uuid"
	"time"
	"set-and-trend/backend/internal/db"
)

type UserRepository struct {
	q *db.Queries
}

func NewUserRepository(q *db.Queries) *UserRepository {
	return &UserRepository{q: q}
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func (r *UserRepository) CreateUser(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := r.q.CreateUser(ctx, id)
	if err != nil {
		return nil, err
	}
	
	return &User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt.Time,
	}, nil
}
