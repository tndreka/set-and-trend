package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
	"set-and-trend/backend/internal/db"
)

type AuthService struct {
	queries *db.Queries
}

func NewAuthService(queries *db.Queries) *AuthService {
	return &AuthService{queries: queries}
}

// HashPassword hashes a plain text password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a plain text password with a hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// SignUp creates a new user account
func (s *AuthService) SignUp(ctx context.Context, username, email, password, name, surname string) (*db.CreateUserRow, error) {
	// Check if username already exists
	_, err := s.queries.GetUserByUsername(ctx, username)
	if err == nil {
		return nil, fmt.Errorf("username already taken")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to check username: %w", err)
	}

	// Check if email already exists
	_, err = s.queries.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, fmt.Errorf("email already registered")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}

	// Hash password
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		ID:           uuid.New(),
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		Name:         pgtype.Text{String: name, Valid: name != ""},
		Surname:      pgtype.Text{String: surname, Valid: surname != ""},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

type LoginResult struct {
	Token    string
	UserID   uuid.UUID
	Username string
	Email    string
}

func (s *AuthService) Login(ctx context.Context, usernameOrEmail, password string) (*LoginResult, error) {
	var user db.GetUserByUsernameRow
	var err error

	user, err = s.queries.GetUserByUsername(ctx, usernameOrEmail)
	if err != nil {
		userByEmail, emailErr := s.queries.GetUserByEmail(ctx, usernameOrEmail)
		if emailErr != nil {
			return nil, fmt.Errorf("invalid credentials")
		}
		user.ID = userByEmail.ID
		user.Username = userByEmail.Username
		user.Email = userByEmail.Email
		user.PasswordHash = userByEmail.PasswordHash
	}

	if !CheckPassword(password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid credentials")
	}

	err = s.queries.UpdateUserLastLogin(ctx, user.ID)
	if err != nil {
		fmt.Printf("Warning: failed to update last login for user %s: %v\n", user.Username, err)
	}

	token, err := GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &LoginResult{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
	}, nil
}

// Helper function to convert string to pgtype.Text
func TextFromString(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}