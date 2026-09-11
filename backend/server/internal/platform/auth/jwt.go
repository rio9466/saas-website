package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Product-fixed token lifetimes for administrator auth.
const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 30 * 24 * time.Hour
)

// TokenType distinguishes access and refresh JWTs.
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

var (
	ErrTokenInvalid      = errors.New("token invalid")
	ErrTokenTypeMismatch = errors.New("token type mismatch")
	ErrJWTConfigInvalid  = errors.New("jwt config invalid")
)

// JWTConfig holds HS256 signing material and registered claim identity.
type JWTConfig struct {
	Secret   string
	Issuer   string
	Audience string
}

// Claims are the JWT payload for administrator access and refresh tokens.
// Roles and permissions must never appear here.
type Claims struct {
	jwt.RegisteredClaims
	SessionID string    `json:"sid"`
	TokenType TokenType `json:"typ"`
}

// TokenService issues and validates HS256 JWTs.
type TokenService struct {
	secret   []byte
	issuer   string
	audience string
	now      func() time.Time
}

// NewTokenService validates config and returns a TokenService.
func NewTokenService(cfg JWTConfig) (*TokenService, error) {
	if strings.TrimSpace(cfg.Secret) == "" {
		return nil, fmt.Errorf("%w: secret is required", ErrJWTConfigInvalid)
	}
	if strings.TrimSpace(cfg.Issuer) == "" {
		return nil, fmt.Errorf("%w: issuer is required", ErrJWTConfigInvalid)
	}
	if strings.TrimSpace(cfg.Audience) == "" {
		return nil, fmt.Errorf("%w: audience is required", ErrJWTConfigInvalid)
	}
	return &TokenService{
		secret:   []byte(cfg.Secret),
		issuer:   cfg.Issuer,
		audience: cfg.Audience,
		now:      time.Now,
	}, nil
}

// IssueAccess creates a 15-minute access token for subject and session.
func (s *TokenService) IssueAccess(subject, sessionID string) (string, *Claims, error) {
	return s.issue(subject, sessionID, TokenTypeAccess, AccessTokenTTL)
}

// IssueRefresh creates a 30-day refresh token for subject and session.
func (s *TokenService) IssueRefresh(subject, sessionID string) (string, *Claims, error) {
	return s.issue(subject, sessionID, TokenTypeRefresh, RefreshTokenTTL)
}

// IssueRefreshExpiring creates a refresh token for subject and session whose
// exp is exactly expiresAt. Refresh rotation uses this to keep every rotated
// JWT aligned with the session's absolute expiry, never extending it.
func (s *TokenService) IssueRefreshExpiring(subject, sessionID string, expiresAt time.Time) (string, *Claims, error) {
	return s.issueExpiring(subject, sessionID, TokenTypeRefresh, expiresAt)
}

func (s *TokenService) issue(subject, sessionID string, typ TokenType, ttl time.Duration) (string, *Claims, error) {
	if s == nil {
		return "", nil, errors.New("token service is not initialized")
	}
	if strings.TrimSpace(subject) == "" {
		return "", nil, fmt.Errorf("%w: subject is required", ErrTokenInvalid)
	}
	if strings.TrimSpace(sessionID) == "" {
		return "", nil, fmt.Errorf("%w: session id is required", ErrTokenInvalid)
	}
	if ttl <= 0 {
		return "", nil, fmt.Errorf("%w: non-positive ttl", ErrTokenInvalid)
	}
	return s.sign(subject, sessionID, typ, s.now().UTC().Add(ttl))
}

func (s *TokenService) issueExpiring(subject, sessionID string, typ TokenType, expiresAt time.Time) (string, *Claims, error) {
	if s == nil {
		return "", nil, errors.New("token service is not initialized")
	}
	if strings.TrimSpace(subject) == "" {
		return "", nil, fmt.Errorf("%w: subject is required", ErrTokenInvalid)
	}
	if strings.TrimSpace(sessionID) == "" {
		return "", nil, fmt.Errorf("%w: session id is required", ErrTokenInvalid)
	}
	expiresAt = expiresAt.UTC()
	now := s.now().UTC()
	if !expiresAt.After(now) {
		return "", nil, fmt.Errorf("%w: expiresAt is not in the future", ErrTokenInvalid)
	}
	// A rotated refresh token must never outlive the product refresh window.
	if expiresAt.After(now.Add(RefreshTokenTTL)) {
		return "", nil, fmt.Errorf("%w: expiresAt exceeds refresh window", ErrTokenInvalid)
	}
	return s.sign(subject, sessionID, typ, expiresAt)
}

func (s *TokenService) sign(subject, sessionID string, typ TokenType, expiresAt time.Time) (string, *Claims, error) {
	now := s.now().UTC()
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   subject,
			Audience:  []string{s.audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
		SessionID: sessionID,
		TokenType: typ,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", nil, fmt.Errorf("sign token: %w", err)
	}
	return signed, claims, nil
}

// ParseAndValidate parses a JWT, validates registered claims, and requires expectedType.
func (s *TokenService) ParseAndValidate(tokenString string, expectedType TokenType) (*Claims, error) {
	if s == nil {
		return nil, errors.New("token service is not initialized")
	}
	if strings.TrimSpace(tokenString) == "" {
		return nil, ErrTokenInvalid
	}
	if expectedType != TokenTypeAccess && expectedType != TokenTypeRefresh {
		return nil, fmt.Errorf("%w: %q", ErrTokenTypeMismatch, expectedType)
	}

	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("%w: unexpected signing method", ErrTokenInvalid)
		}
		return s.secret, nil
	},
		jwt.WithIssuer(s.issuer),
		jwt.WithAudience(s.audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
	)
	if err != nil || !parsed.Valid {
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	if claims.Subject == "" || claims.ID == "" || claims.SessionID == "" {
		return nil, ErrTokenInvalid
	}
	if claims.ExpiresAt == nil || claims.IssuedAt == nil || claims.NotBefore == nil {
		return nil, ErrTokenInvalid
	}
	if claims.TokenType != expectedType {
		return nil, ErrTokenTypeMismatch
	}
	return claims, nil
}
