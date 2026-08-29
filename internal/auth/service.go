package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/sury-dev/portfolio-server/internal/config"
	"github.com/sury-dev/portfolio-server/internal/utils"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

const (
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"
)

type Service struct {
	repo Repository
	cfg  config.AuthConfig
}

func NewService(repo Repository, cfg config.AuthConfig) *Service {
	return &Service{repo: repo, cfg: cfg}
}

type loginResult struct {
	AccessToken  string
	RefreshToken string
	AccessExp    time.Time
	RefreshExp   time.Time
}

type tokenClaims struct {
	Typ string `json:"typ"`
	jwt.RegisteredClaims
}

func (s *Service) Login(ctx context.Context, password string) (*loginResult, error) {
	passwordHash, err := s.repo.GetPasswordHash(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := utils.CompareEncryptedString(passwordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, accessExp, err := mintToken(s.cfg.AccessSecretKey, tokenTypeAccess, s.cfg.AccessTokenDuration)
	if err != nil {
		return nil, err
	}
	refreshToken, refreshExp, err := mintToken(s.cfg.RefreshSecretKey, tokenTypeRefresh, s.cfg.RefreshTokenDuration)
	if err != nil {
		return nil, err
	}

	if err := s.repo.ReplaceSession(
		ctx,
		utils.HashSHA256(accessToken),
		utils.HashSHA256(refreshToken),
		accessExp,
		refreshExp,
	); err != nil {
		return nil, err
	}

	return &loginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AccessExp:    accessExp,
		RefreshExp:   refreshExp,
	}, nil
}

// ValidateAccessToken checks JWT signature/exp/typ=access and that SHA256(token)
// matches the current admin.access_token_hash (immediate invalidation on overwrite).
func (s *Service) ValidateAccessToken(ctx context.Context, rawToken string) error {
	return s.validateSessionToken(ctx, rawToken, tokenTypeAccess, s.cfg.AccessSecretKey, true)
}

// ValidateRefreshToken checks JWT signature/exp/typ=refresh and that SHA256(token)
// matches the current admin.refresh_token_hash.
func (s *Service) ValidateRefreshToken(ctx context.Context, rawToken string) error {
	return s.validateSessionToken(ctx, rawToken, tokenTypeRefresh, s.cfg.RefreshSecretKey, false)
}

func (s *Service) validateSessionToken(
	ctx context.Context,
	rawToken string,
	expectedTyp string,
	secret string,
	access bool,
) error {
	if rawToken == "" {
		return ErrInvalidToken
	}

	claims, err := parseToken(rawToken, secret)
	if err != nil {
		return ErrInvalidToken
	}
	if claims.Typ != expectedTyp {
		return ErrInvalidToken
	}

	session, err := s.repo.GetSession(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidToken
		}
		return err
	}

	tokenHash := utils.HashSHA256(rawToken)
	now := time.Now().UTC()

	if access {
		if session.AccessTokenHash == nil || *session.AccessTokenHash != tokenHash {
			return ErrInvalidToken
		}
		if session.AccessTokenExpiresAt == nil || now.After(*session.AccessTokenExpiresAt) {
			return ErrInvalidToken
		}
		return nil
	}

	if session.RefreshTokenHash == nil || *session.RefreshTokenHash != tokenHash {
		return ErrInvalidToken
	}
	if session.RefreshTokenExpiresAt == nil || now.After(*session.RefreshTokenExpiresAt) {
		return ErrInvalidToken
	}
	return nil
}

func mintToken(secret, typ string, ttl time.Duration) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	claims := tokenClaims{
		Typ: typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "admin",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign %s token: %w", typ, err)
	}
	return token, expiresAt, nil
}

func parseToken(rawToken, secret string) (*tokenClaims, error) {
	token, err := jwt.ParseWithClaims(rawToken, &tokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*tokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
