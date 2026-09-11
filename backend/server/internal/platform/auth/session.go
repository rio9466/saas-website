package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	sessionKeyPrefix      = "easy-admin:admin-session:"
	sessionIndexKeyPrefix = "easy-admin:admin-session-index:"
	// Business-user session namespaces are separate from the administrator
	// namespaces above; tokens, cookies, and Redis keys never cross families.
	userSessionKeyPrefix      = "easy-admin:user-session:"
	userSessionIndexKeyPrefix = "easy-admin:user-session-index:"
	sessionWatchTries         = 8
)

// Session status values stored in Redis.
const (
	SessionStatusActive  = "active"
	SessionStatusRevoked = "revoked"
)

var (
	ErrSessionInvalid = errors.New("session invalid")
	ErrRefreshReplay  = errors.New("refresh token replay")
)

// Session is the Redis-backed session document for an administrator or a
// business user. The subject field (AdminID) holds the owning account row id;
// the store namespace decides which account family a document belongs to.
// Only a hash of the current refresh JTI/verifier is stored.
type Session struct {
	AdminID        int64     `json:"admin_id"`
	RefreshJTIHash string    `json:"refresh_jti_hash"`
	Status         string    `json:"status"`
	AuthEpoch      int64     `json:"auth_epoch"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// SessionStore manages Redis sessions and fail-closed validation. Two
// namespace families exist: administrator sessions (easy-admin:admin-session:*)
// and business-user sessions (easy-admin:user-session:*). The document shape is
// shared and the subject id holds the row id of the owning account type.
type SessionStore struct {
	rdb         *goredis.Client
	keyPrefix   string
	indexPrefix string
}

// NewSessionStore wraps a go-redis client for administrator sessions. rdb must be non-nil.
func NewSessionStore(rdb *goredis.Client) (*SessionStore, error) {
	if rdb == nil {
		return nil, errors.New("redis client is required")
	}
	return &SessionStore{rdb: rdb, keyPrefix: sessionKeyPrefix, indexPrefix: sessionIndexKeyPrefix}, nil
}

// NewUserSessionStore wraps a go-redis client for business-user sessions. The
// namespace is distinct from administrator sessions so the two account types
// can never share or collide on session state.
func NewUserSessionStore(rdb *goredis.Client) (*SessionStore, error) {
	if rdb == nil {
		return nil, errors.New("redis client is required")
	}
	return &SessionStore{rdb: rdb, keyPrefix: userSessionKeyPrefix, indexPrefix: userSessionIndexKeyPrefix}, nil
}

// SessionKey returns the Redis key for a session id.
func SessionKey(sid string) string {
	return sessionKeyPrefix + sid
}

// SessionIndexKey returns the Redis set that indexes session ids for an administrator.
func SessionIndexKey(adminID int64) string {
	return sessionIndexKeyPrefix + strconv.FormatInt(adminID, 10)
}

// UserSessionKey returns the Redis key for a business-user session id.
func UserSessionKey(sid string) string {
	return userSessionKeyPrefix + sid
}

// UserSessionIndexKey returns the Redis set that indexes business-user session ids.
func UserSessionIndexKey(userID int64) string {
	return userSessionIndexKeyPrefix + strconv.FormatInt(userID, 10)
}

// HashRefreshVerifier returns the sha256 hex digest of a refresh JTI or token verifier.
func HashRefreshVerifier(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (s *SessionStore) key(key string) string {
	if s == nil || s.keyPrefix == "" {
		return sessionKeyPrefix + key
	}
	return s.keyPrefix + key
}

func (s *SessionStore) indexKey(subjectID int64) string {
	if s == nil || s.indexPrefix == "" {
		return sessionIndexKeyPrefix + strconv.FormatInt(subjectID, 10)
	}
	return s.indexPrefix + strconv.FormatInt(subjectID, 10)
}

// Create stores a new active session. TTL is capped to refresh expiry.
// authEpoch is the administrator's auth_epoch at issue time; protected
// operations compare it with the current primary-database epoch and fail
// closed when a password/account transition invalidated the session.
func (s *SessionStore) Create(ctx context.Context, sid string, adminID int64, refreshJTIHash string, authEpoch int64, expiresAt time.Time) error {
	if err := s.require(); err != nil {
		return err
	}
	if strings.TrimSpace(sid) == "" || adminID <= 0 || strings.TrimSpace(refreshJTIHash) == "" {
		return ErrSessionInvalid
	}
	expiresAt = expiresAt.UTC()
	ttl, err := remainingTTL(expiresAt)
	if err != nil {
		return err
	}

	doc := Session{
		AdminID:        adminID,
		RefreshJTIHash: refreshJTIHash,
		Status:         SessionStatusActive,
		AuthEpoch:      authEpoch,
		ExpiresAt:      expiresAt,
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	if err := s.rdb.Set(ctx, s.key(sid), raw, ttl).Err(); err != nil {
		return fmt.Errorf("store session: %w", err)
	}
	pipe := s.rdb.TxPipeline()
	pipe.SAdd(ctx, s.indexKey(adminID), sid)
	pipe.Expire(ctx, s.indexKey(adminID), ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		_ = s.rdb.Del(ctx, s.key(sid)).Err()
		return fmt.Errorf("index session: %w", err)
	}
	return nil
}

// Get loads a session. Missing, revoked, corrupt, or Redis errors fail closed.
func (s *SessionStore) Get(ctx context.Context, sid string) (*Session, error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(sid) == "" {
		return nil, ErrSessionInvalid
	}

	raw, err := s.rdb.Get(ctx, s.key(sid)).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, ErrSessionInvalid
		}
		return nil, fmt.Errorf("get session: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(raw, &sess); err != nil {
		return nil, ErrSessionInvalid
	}
	if sess.Status != SessionStatusActive {
		return nil, ErrSessionInvalid
	}
	if sess.AdminID <= 0 || sess.RefreshJTIHash == "" || sess.ExpiresAt.IsZero() {
		return nil, ErrSessionInvalid
	}
	if sess.AuthEpoch < 0 {
		return nil, ErrSessionInvalid
	}
	if !sess.ExpiresAt.After(time.Now().UTC()) {
		return nil, ErrSessionInvalid
	}
	return &sess, nil
}

// RotateRefresh atomically verifies the old refresh JTI hash and installs the
// new one. The session's absolute expiry is never changed by rotation: Redis
// TTLs are recomputed as the remaining time until the original expiry, so
// repeated refreshes can never extend a session beyond the login-time 30-day
// lifetime. On hash mismatch the session is revoked and ErrRefreshReplay is
// returned.
func (s *SessionStore) RotateRefresh(ctx context.Context, sid, oldHash, newHash string) error {
	if err := s.require(); err != nil {
		return err
	}
	if strings.TrimSpace(sid) == "" || strings.TrimSpace(oldHash) == "" || strings.TrimSpace(newHash) == "" {
		return ErrSessionInvalid
	}

	key := s.key(sid)
	var replay bool

	for attempt := 0; attempt < sessionWatchTries; attempt++ {
		err := s.rdb.Watch(ctx, func(tx *goredis.Tx) error {
			raw, err := tx.Get(ctx, key).Bytes()
			if err != nil {
				if errors.Is(err, goredis.Nil) {
					return ErrSessionInvalid
				}
				return fmt.Errorf("get session for rotate: %w", err)
			}

			var sess Session
			if err := json.Unmarshal(raw, &sess); err != nil {
				return ErrSessionInvalid
			}
			if sess.Status != SessionStatusActive {
				return ErrSessionInvalid
			}

			// Rotating never extends the absolute expiry; sessions whose
			// remaining lifetime is gone are dropped instead of renewed.
			ttl, ttlErr := remainingTTL(sess.ExpiresAt.UTC())
			if ttlErr != nil {
				_, delErr := tx.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
					pipe.Del(ctx, key)
					pipe.SRem(ctx, s.indexKey(sess.AdminID), sid)
					return nil
				})
				if delErr != nil {
					return fmt.Errorf("drop expired session on rotate: %w", delErr)
				}
				return ErrSessionInvalid
			}

			if sess.RefreshJTIHash != oldHash {
				sess.Status = SessionStatusRevoked
				payload, err := json.Marshal(sess)
				if err != nil {
					return fmt.Errorf("marshal revoked session: %w", err)
				}
				_, err = tx.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
					pipe.Set(ctx, key, payload, ttl)
					return nil
				})
				if err != nil {
					return fmt.Errorf("revoke replayed session: %w", err)
				}
				replay = true
				return nil
			}

			sess.RefreshJTIHash = newHash
			payload, err := json.Marshal(sess)
			if err != nil {
				return fmt.Errorf("marshal rotated session: %w", err)
			}
			_, err = tx.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
				pipe.Set(ctx, key, payload, ttl)
				pipe.Expire(ctx, s.indexKey(sess.AdminID), ttl)
				return nil
			})
			if err != nil {
				return fmt.Errorf("rotate session: %w", err)
			}
			replay = false
			return nil
		}, key)
		if err == nil {
			if replay {
				return ErrRefreshReplay
			}
			return nil
		}
		if errors.Is(err, goredis.TxFailedErr) {
			continue
		}
		return err
	}
	return fmt.Errorf("rotate session: exceeded optimistic lock retries")
}

// Revoke marks a session revoked. Missing keys are treated as already invalid.
func (s *SessionStore) Revoke(ctx context.Context, sid string) error {
	if err := s.require(); err != nil {
		return err
	}
	if strings.TrimSpace(sid) == "" {
		return ErrSessionInvalid
	}

	key := s.key(sid)
	raw, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil
		}
		return fmt.Errorf("get session for revoke: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(raw, &sess); err != nil {
		// Corrupt session: delete to fail closed for future use.
		if delErr := s.rdb.Del(ctx, key).Err(); delErr != nil {
			return fmt.Errorf("delete corrupt session: %w", delErr)
		}
		return nil
	}

	sess.Status = SessionStatusRevoked
	payload, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("marshal revoked session: %w", err)
	}

	ttl, ttlErr := remainingTTL(sess.ExpiresAt.UTC())
	if ttlErr != nil {
		// Expired: drop the key.
		if delErr := s.rdb.Del(ctx, key).Err(); delErr != nil {
			return fmt.Errorf("delete expired session: %w", delErr)
		}
		_ = s.rdb.SRem(ctx, s.indexKey(sess.AdminID), sid).Err()
		return nil
	}
	if err := s.rdb.Set(ctx, key, payload, ttl).Err(); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	_ = s.rdb.SRem(ctx, s.indexKey(sess.AdminID), sid).Err()
	return nil
}

// RevokeAllForAdmin revokes every indexed session for the administrator.
func (s *SessionStore) RevokeAllForAdmin(ctx context.Context, adminID int64) error {
	if err := s.require(); err != nil {
		return err
	}
	if adminID <= 0 {
		return ErrSessionInvalid
	}

	indexKey := s.indexKey(adminID)
	sids, err := s.rdb.SMembers(ctx, indexKey).Result()
	if err != nil {
		return fmt.Errorf("list admin sessions: %w", err)
	}
	for _, sid := range sids {
		if revErr := s.Revoke(ctx, sid); revErr != nil && !errors.Is(revErr, ErrSessionInvalid) {
			return revErr
		}
	}
	if err := s.rdb.Del(ctx, indexKey).Err(); err != nil {
		return fmt.Errorf("clear admin session index: %w", err)
	}
	return nil
}

func (s *SessionStore) require() error {
	if s == nil || s.rdb == nil {
		return errors.New("session store is not initialized")
	}
	return nil
}

func remainingTTL(expiresAt time.Time) (time.Duration, error) {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return 0, ErrSessionInvalid
	}
	return ttl, nil
}

// Raw returns the underlying go-redis client for namespaced auxiliary keys
// (verification tokens) that share the user session store's client.
func (s *SessionStore) Raw() *goredis.Client {
	if s == nil {
		return nil
	}
	return s.rdb
}
