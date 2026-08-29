package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sury-dev/portfolio-server/internal/store"
)

// Repository isolates auth persistence from the service layer.
type Repository interface {
	GetPasswordHash(ctx context.Context) (string, error)
	GetSession(ctx context.Context) (*Session, error)
	ReplaceSession(ctx context.Context, accessHash, refreshHash string, accessExp, refreshExp time.Time) error
}

// Session is the current singleton admin session state.
type Session struct {
	AccessTokenHash       *string
	RefreshTokenHash      *string
	AccessTokenExpiresAt  *time.Time
	RefreshTokenExpiresAt *time.Time
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) GetPasswordHash(ctx context.Context) (string, error) {
	var passwordHash string
	err := r.db.QueryRow(ctx, store.GetAdminPasswordHash).Scan(&passwordHash)
	if err != nil {
		return "", err
	}
	return passwordHash, nil
}

func (r *postgresRepository) GetSession(ctx context.Context) (*Session, error) {
	var session Session
	err := r.db.QueryRow(ctx, store.GetAdminSession).Scan(
		&session.AccessTokenHash,
		&session.RefreshTokenHash,
		&session.AccessTokenExpiresAt,
		&session.RefreshTokenExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *postgresRepository) ReplaceSession(
	ctx context.Context,
	accessHash, refreshHash string,
	accessExp, refreshExp time.Time,
) error {
	tag, err := r.db.Exec(
		ctx,
		store.UpdateAdminSession,
		accessHash,
		refreshHash,
		accessExp,
		refreshExp,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("admin session update: expected 1 row, got %d", tag.RowsAffected())
	}
	return nil
}
