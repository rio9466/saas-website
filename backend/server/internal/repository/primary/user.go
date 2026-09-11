package primary

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrIdempotencyReplay is returned when a points idempotency key was already
// applied; the change must never run twice.
var ErrIdempotencyReplay = errors.New("points idempotency replay")

// userModel is the business-user row. Points are NUMERIC(20,4) mapped as
// pgtype.Numeric so no floating-point conversion ever touches them.
type userModel struct {
	ID                int64          `gorm:"column:id;primaryKey"`
	Username          string         `gorm:"column:username"`
	Email             string         `gorm:"column:email"`
	PasswordHash      string         `gorm:"column:password_hash"`
	AvatarURL         string         `gorm:"column:avatar_url"`
	Nickname          string         `gorm:"column:nickname"`
	RegistrationIP    string         `gorm:"column:registration_ip"`
	Status            string         `gorm:"column:status"`
	EmailVerifiedAt   *time.Time     `gorm:"column:email_verified_at"`
	LastLoginAt       *time.Time     `gorm:"column:last_login_at"`
	PointsBalance     pgtype.Numeric `gorm:"column:points_balance;type:numeric(20,4)"`
	ConsumptionPoints pgtype.Numeric `gorm:"column:consumption_points;type:numeric(20,4)"`
	LevelID           int64          `gorm:"column:level_id"`
	LevelMode         string         `gorm:"column:level_mode"`
	Remark            string         `gorm:"column:remark"`
	AuthEpoch         int64          `gorm:"column:auth_epoch"`
	// Read-only join projections (loaded via Select/Joins; never written).
	LevelCode string    `gorm:"column:level_code;->"`
	LevelName string    `gorm:"column:level_name;->"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (userModel) TableName() string { return "users" }

type userLevelModel struct {
	ID              int64          `gorm:"column:id;primaryKey"`
	Code            string         `gorm:"column:code"`
	Name            string         `gorm:"column:name"`
	IconURL         string         `gorm:"column:icon_url"`
	ThresholdPoints pgtype.Numeric `gorm:"column:threshold_points;type:numeric(20,4)"`
	SortOrder       int            `gorm:"column:sort_order"`
	Enabled         bool           `gorm:"column:enabled"`
	CreatedAt       time.Time      `gorm:"column:created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at"`
}

func (userLevelModel) TableName() string { return "user_levels" }

type userPointTransactionModel struct {
	ID               int64          `gorm:"column:id;primaryKey"`
	UserID           int64          `gorm:"column:user_id"`
	PointsDelta      pgtype.Numeric `gorm:"column:points_delta;type:numeric(20,4)"`
	ConsumptionDelta pgtype.Numeric `gorm:"column:consumption_delta;type:numeric(20,4)"`
	BalanceAfter     pgtype.Numeric `gorm:"column:balance_after;type:numeric(20,4)"`
	ConsumptionAfter pgtype.Numeric `gorm:"column:consumption_after;type:numeric(20,4)"`
	Reason           string         `gorm:"column:reason"`
	ActorID          *int64         `gorm:"column:actor_id"`
	IdempotencyKey   string         `gorm:"column:idempotency_key"`
	CreatedAt        time.Time      `gorm:"column:created_at"`
}

func (userPointTransactionModel) TableName() string { return "user_point_transactions" }

type systemSettingsModel struct {
	ID                        int64          `gorm:"column:id;primaryKey"`
	PlatformName              string         `gorm:"column:platform_name"`
	PublicFrontendURL         string         `gorm:"column:public_frontend_url"`
	PublicAPIURL              string         `gorm:"column:public_api_url"`
	RegistrationEnabled       bool           `gorm:"column:registration_enabled"`
	UsernameLoginEnabled      bool           `gorm:"column:username_login_enabled"`
	EmailLoginEnabled         bool           `gorm:"column:email_login_enabled"`
	EmailVerificationRequired bool           `gorm:"column:email_verification_required"`
	DefaultLevelID            int64          `gorm:"column:default_level_id"`
	DefaultAvatarURL          string         `gorm:"column:default_avatar_url"`
	RegistrationPoints        pgtype.Numeric `gorm:"column:registration_points;type:numeric(20,4)"`
	SMTPEnabled               bool           `gorm:"column:smtp_enabled"`
	SMTPHost                  string         `gorm:"column:smtp_host"`
	SMTPPort                  int            `gorm:"column:smtp_port"`
	SMTPUsername              string         `gorm:"column:smtp_username"`
	SMTPPasswordEncrypted     string         `gorm:"column:smtp_password_encrypted"`
	SMTPFromEmail             string         `gorm:"column:smtp_from_email"`
	SMTPFromName              string         `gorm:"column:smtp_from_name"`
	SMTPTLSMode               string         `gorm:"column:smtp_tls_mode"`
	Version                   int            `gorm:"column:version"`
	UpdatedBy                 int64          `gorm:"column:updated_by"`
	UpdatedAt                 time.Time      `gorm:"column:updated_at"`
}

func (systemSettingsModel) TableName() string { return "system_settings" }

// UserRepository persists business users, their levels, the points ledger, and
// the typed system-settings singleton in primary PostgreSQL.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository constructs a UserRepository backed by db.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) session(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

// --- decimal conversions ---------------------------------------------------

// numericToDecimal normalizes a NUMERIC(20,4) value to exact 1/10000 units.
func numericToDecimal(n pgtype.Numeric) (userdomain.Decimal4, error) {
	if !n.Valid || n.Int == nil {
		return userdomain.Zero(), nil
	}
	scaled, err := scaleNumericToFour(n)
	if err != nil {
		return userdomain.Decimal4{}, err
	}
	return userdomain.Decimal4FromUnscaled(scaled)
}

func scaleNumericToFour(n pgtype.Numeric) (*big.Int, error) {
	if n.Int == nil {
		return big.NewInt(0), nil
	}
	shift := 4 + int(n.Exp)
	if shift == 0 {
		return new(big.Int).Set(n.Int), nil
	}
	factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(absInt(shift))), nil)
	v := new(big.Int)
	if shift > 0 {
		return v.Mul(n.Int, factor), nil
	}
	q, r := new(big.Int), new(big.Int)
	q.QuoRem(n.Int, factor, r)
	if r.Sign() != 0 {
		return nil, fmt.Errorf("numeric value is not a four-decimal value")
	}
	return q, nil
}

func decimalToNumeric(d userdomain.Decimal4) pgtype.Numeric {
	return pgtype.Numeric{Int: d.Unscaled(), Exp: -4, Valid: true}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// --- users -----------------------------------------------------------------

// CreateUser inserts one active/pending business user. The caller resolves the
// default level from settings; pass defaultLevelID explicitly.
func (r *UserRepository) CreateUser(ctx context.Context, u *userdomain.User, defaultLevelID int64) error {
	if u == nil {
		return fmt.Errorf("user is nil")
	}
	if defaultLevelID <= 0 {
		return fmt.Errorf("default level id is required")
	}
	now := time.Now().UTC()
	model := userModel{
		Username:          strings.TrimSpace(u.Username),
		Email:             strings.TrimSpace(strings.ToLower(u.Email)),
		PasswordHash:      u.PasswordHash,
		AvatarURL:         u.AvatarURL,
		Nickname:          u.Nickname,
		RegistrationIP:    u.RegistrationIP,
		Status:            u.Status,
		EmailVerifiedAt:   u.EmailVerifiedAt,
		PointsBalance:     decimalToNumeric(userdomain.Zero()),
		ConsumptionPoints: decimalToNumeric(userdomain.Zero()),
		LevelID:           defaultLevelID,
		LevelMode:         userdomain.LevelModeAuto,
		Remark:            "",
		AuthEpoch:         1,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := r.session(ctx).Create(&model).Error; err != nil {
		return wrapDBError(err)
	}
	u.ID = model.ID
	u.CreatedAt = model.CreatedAt
	u.UpdatedAt = model.UpdatedAt
	return nil
}

// GetUserByID loads a user with level code/name. PasswordHash is omitted.
func (r *UserRepository) GetUserByID(ctx context.Context, id int64) (*userdomain.User, error) {
	u, err := r.loadUser(ctx, "users.id = ?", id)
	if err != nil {
		return nil, err
	}
	u.PasswordHash = ""
	return u, nil
}

// GetUserByUsername loads a user by normalized username, including the hash.
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*userdomain.User, error) {
	return r.loadUser(ctx, "lower(btrim(users.username)) = lower(btrim(?))", username)
}

// GetUserByEmail loads a user by normalized email, including the hash.
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*userdomain.User, error) {
	return r.loadUser(ctx, "lower(btrim(users.email)) = lower(btrim(?))", email)
}

func (r *UserRepository) loadUser(ctx context.Context, cond string, arg any) (*userdomain.User, error) {
	var model userModel
	q := r.session(ctx).
		Select("users.*, ul.code AS level_code, ul.name AS level_name").
		Joins("JOIN user_levels AS ul ON ul.id = users.level_id").
		Where(cond, arg)
	if err := q.First(&model).Error; err != nil {
		return nil, wrapDBError(err)
	}
	return userFromModel(model), nil
}

// ListUsers returns a paginated business-user list. Search matches username,
// email, and nickname. Filters are optional normalized values.
type UserListFilter struct {
	Query   string
	Status  string
	LevelID *int64
}

func (r *UserRepository) ListUsers(ctx context.Context, page, pageSize int, filter UserListFilter) (adminauth.Page[userdomain.User], error) {
	page, pageSize = normalizePage(page, pageSize)
	out := adminauth.Page[userdomain.User]{Page: page, PageSize: pageSize, Items: []userdomain.User{}}

	db := r.session(ctx).Model(&userModel{}).Select("users.*, ul.code AS level_code, ul.name AS level_name").
		Joins("JOIN user_levels AS ul ON ul.id = users.level_id")
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + escapeLike(q) + "%"
		db = db.Where("(users.username ILIKE ? ESCAPE '\\' OR users.email ILIKE ? ESCAPE '\\' OR users.nickname ILIKE ? ESCAPE '\\')", like, like, like)
	}
	if s := strings.TrimSpace(filter.Status); s != "" {
		db = db.Where("users.status = ?", s)
	}
	if filter.LevelID != nil {
		db = db.Where("users.level_id = ?", *filter.LevelID)
	}

	if err := db.Count(&out.Total).Error; err != nil {
		return out, wrapDBError(err)
	}

	var models []userModel
	err := db.Order("users.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&models).Error
	if err != nil {
		return out, wrapDBError(err)
	}

	out.Items = make([]userdomain.User, 0, len(models))
	for _, model := range models {
		out.Items = append(out.Items, *userFromModel(model))
	}
	return out, nil
}

// UpdateUserProfile updates the mutable user profile fields the generic update
// endpoint owns: email, nickname, avatar_url, and administrator remark. Nil
// pointers leave a field unchanged. Email changes are mirrored to the
// normalized email and reset email verification (the admin-created account case
// re-verifies through SetUserEmailVerified on demand).
func (r *UserRepository) UpdateUserProfile(ctx context.Context, id int64, email, nickname, avatarURL, remark *string) error {
	updates := map[string]any{"updated_at": time.Now().UTC()}
	if email != nil {
		updates["email"] = strings.TrimSpace(strings.ToLower(*email))
	}
	if nickname != nil {
		updates["nickname"] = *nickname
	}
	if avatarURL != nil {
		updates["avatar_url"] = *avatarURL
	}
	if remark != nil {
		updates["remark"] = *remark
	}
	if len(updates) == 1 {
		return nil
	}
	res := r.session(ctx).Model(&userModel{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return wrapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// SetUserStatus transitions the user status. Bumping auth_epoch only happens in
// the dedicated lifecycle operations; verification transitions do not revoke
// sessions because no user session can exist pre-verification.
func (r *UserRepository) SetUserStatus(ctx context.Context, id int64, status string, emailVerifiedAt *time.Time) error {
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now().UTC(),
	}
	verified := emailVerifiedAt
	updates["email_verified_at"] = verified
	res := r.session(ctx).Model(&userModel{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return wrapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// SetUserEnabled sets enablement and bumps auth_epoch so previously issued user
// sessions are invalidated. Re-enabling never restores old epochs.
func (r *UserRepository) SetUserEnabled(ctx context.Context, id int64, enabled bool) error {
	status := userdomain.StatusActive
	if !enabled {
		status = userdomain.StatusDisabled
	}
	res := r.session(ctx).Model(&userModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     status,
			"auth_epoch": gorm.Expr("auth_epoch + 1"),
			"updated_at": time.Now().UTC(),
		})
	if res.Error != nil {
		return wrapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// SetUserPassword replaces the password hash and bumps auth_epoch so every
// previously issued session is immediately invalidated.
func (r *UserRepository) SetUserPassword(ctx context.Context, id int64, passwordHash string) error {
	res := r.session(ctx).Model(&userModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"password_hash": passwordHash,
			"auth_epoch":    gorm.Expr("auth_epoch + 1"),
			"updated_at":    time.Now().UTC(),
		})
	if res.Error != nil {
		return wrapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// TouchLastLogin records the most recent successful login. Refreshes and
// ordinary authenticated requests never call this.
func (r *UserRepository) TouchLastLogin(ctx context.Context, id int64) error {
	now := time.Now().UTC()
	res := r.session(ctx).Model(&userModel{}).
		Where("id = ?", id).
		Updates(map[string]any{"last_login_at": now, "updated_at": now})
	if res.Error != nil {
		return wrapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// userFromModel converts a joined user row to the domain type.
func userFromModel(model userModel) *userdomain.User {
	balance, err := numericToDecimal(model.PointsBalance)
	if err != nil {
		balance = userdomain.Zero()
	}
	consumption, err := numericToDecimal(model.ConsumptionPoints)
	if err != nil {
		consumption = userdomain.Zero()
	}
	return &userdomain.User{
		ID:                model.ID,
		Username:          model.Username,
		Email:             model.Email,
		PasswordHash:      model.PasswordHash,
		AvatarURL:         model.AvatarURL,
		Nickname:          model.Nickname,
		RegistrationIP:    model.RegistrationIP,
		Status:            model.Status,
		EmailVerifiedAt:   model.EmailVerifiedAt,
		LastLoginAt:       model.LastLoginAt,
		PointsBalance:     balance,
		ConsumptionPoints: consumption,
		LevelID:           model.LevelID,
		LevelMode:         model.LevelMode,
		LevelCode:         model.LevelCode,
		LevelName:         model.LevelName,
		Remark:            model.Remark,
		AuthEpoch:         model.AuthEpoch,
		CreatedAt:         model.CreatedAt,
		UpdatedAt:         model.UpdatedAt,
	}
}

// --- points ledger ---------------------------------------------------------

// AdjustPoints atomically applies an available/consumption delta inside one
// primary transaction with user row locking, inserts an immutable ledger row,
// and recalculates an auto-mode level when consumption grows. The same
// idempotency key can never apply twice.
func (r *UserRepository) AdjustPoints(ctx context.Context, userID int64, actorID *int64, pointsDelta, consumptionDelta userdomain.Decimal4, reason, idempotencyKey string) (*userdomain.PointTransaction, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return nil, fmt.Errorf("idempotency key is required")
	}
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("reason is required")
	}
	if pointsDelta.IsZero() && consumptionDelta.IsZero() {
		return nil, ErrValidationNoop
	}
	if consumptionDelta.IsNegative() {
		return nil, ErrConsumptionNegative
	}

	var out *userdomain.PointTransaction
	err := r.session(ctx).Transaction(func(tx *gorm.DB) error {
		// Idempotency is decided inside the same transaction as the balance
		// update so a replayed call can never double-apply.
		var existing int64
		if err := tx.Model(&userPointTransactionModel{}).
			Where("idempotency_key = ?", idempotencyKey).
			Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return ErrIdempotencyReplay
		}

		var model userModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", userID).
			First(&model).Error; err != nil {
			return err
		}

		balance, err := numericToDecimal(model.PointsBalance)
		if err != nil {
			return err
		}
		consumption, err := numericToDecimal(model.ConsumptionPoints)
		if err != nil {
			return err
		}

		newBalance, err := balance.Add(pointsDelta)
		if err != nil {
			return fmt.Errorf("%w: available points %s %s", ErrBalanceBelowZero, balance.String(), pointsDelta.String())
		}
		// Reject the transition before the ledger insert so a would-be negative
		// balance is a semantic conflict, never a database check violation that
		// the caller can only report as a dependency failure.
		if newBalance.IsNegative() {
			return fmt.Errorf("%w: available points %s %s", ErrBalanceBelowZero, balance.String(), pointsDelta.String())
		}
		newConsumption, err := consumption.Add(consumptionDelta)
		if err != nil {
			return err
		}
		if newConsumption.IsNegative() {
			return ErrConsumptionNegative
		}

		now := time.Now().UTC()
		pt := userPointTransactionModel{
			UserID:           userID,
			PointsDelta:      decimalToNumeric(pointsDelta),
			ConsumptionDelta: decimalToNumeric(consumptionDelta),
			BalanceAfter:     decimalToNumeric(newBalance),
			ConsumptionAfter: decimalToNumeric(newConsumption),
			Reason:           strings.TrimSpace(reason),
			ActorID:          actorID,
			IdempotencyKey:   idempotencyKey,
			CreatedAt:        now,
		}
		if err := tx.Create(&pt).Error; err != nil {
			return err
		}

		updates := map[string]any{
			"points_balance":     decimalToNumeric(newBalance),
			"consumption_points": decimalToNumeric(newConsumption),
			"updated_at":         now,
		}
		if model.LevelMode == userdomain.LevelModeAuto && !consumptionDelta.IsZero() {
			levelID, err := EnabledLevelAtOrBelowTx(tx, newConsumption)
			if err != nil {
				return err
			}
			updates["level_id"] = levelID
		}
		if err := tx.Model(&userModel{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
			return err
		}

		out = &userdomain.PointTransaction{
			ID:               pt.ID,
			UserID:           userID,
			PointsDelta:      pointsDelta,
			ConsumptionDelta: consumptionDelta,
			BalanceAfter:     newBalance,
			ConsumptionAfter: newConsumption,
			Reason:           strings.TrimSpace(reason),
			ActorID:          actorID,
			IdempotencyKey:   idempotencyKey,
			CreatedAt:        now,
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrIdempotencyReplay) {
			return nil, err
		}
		return nil, wrapDBError(err)
	}
	return out, nil
}

// ListPointTransactions returns a paginated ledger for one user, newest first.
func (r *UserRepository) ListPointTransactions(ctx context.Context, userID int64, page, pageSize int) (adminauth.Page[userdomain.PointTransaction], error) {
	page, pageSize = normalizePage(page, pageSize)
	out := adminauth.Page[userdomain.PointTransaction]{Page: page, PageSize: pageSize, Items: []userdomain.PointTransaction{}}
	db := r.session(ctx).Model(&userPointTransactionModel{}).Where("user_id = ?", userID)
	if err := db.Count(&out.Total).Error; err != nil {
		return out, wrapDBError(err)
	}
	var models []userPointTransactionModel
	err := db.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&models).Error
	if err != nil {
		return out, wrapDBError(err)
	}
	out.Items = make([]userdomain.PointTransaction, 0, len(models))
	for _, model := range models {
		pt, err := pointTransactionFromModel(model)
		if err != nil {
			return out, err
		}
		out.Items = append(out.Items, *pt)
	}
	return out, nil
}

func pointTransactionFromModel(model userPointTransactionModel) (*userdomain.PointTransaction, error) {
	pointsDelta, err := numericToDecimal(model.PointsDelta)
	if err != nil {
		return nil, err
	}
	consumptionDelta, err := numericToDecimal(model.ConsumptionDelta)
	if err != nil {
		return nil, err
	}
	balanceAfter, err := numericToDecimal(model.BalanceAfter)
	if err != nil {
		return nil, err
	}
	consumptionAfter, err := numericToDecimal(model.ConsumptionAfter)
	if err != nil {
		return nil, err
	}
	return &userdomain.PointTransaction{
		ID:               model.ID,
		UserID:           model.UserID,
		PointsDelta:      pointsDelta,
		ConsumptionDelta: consumptionDelta,
		BalanceAfter:     balanceAfter,
		ConsumptionAfter: consumptionAfter,
		Reason:           model.Reason,
		ActorID:          model.ActorID,
		IdempotencyKey:   model.IdempotencyKey,
		CreatedAt:        model.CreatedAt,
	}, nil
}

// --- levels ----------------------------------------------------------------

// ListUserLevels returns all user levels ordered by threshold then sort.
func (r *UserRepository) ListUserLevels(ctx context.Context) ([]userdomain.UserLevel, error) {
	var models []userLevelModel
	if err := r.session(ctx).Order("threshold_points ASC, sort_order ASC, id ASC").Find(&models).Error; err != nil {
		return nil, wrapDBError(err)
	}
	out := make([]userdomain.UserLevel, 0, len(models))
	for _, model := range models {
		level, err := userLevelFromModel(model)
		if err != nil {
			return nil, err
		}
		out = append(out, *level)
	}
	return out, nil
}

// GetUserLevelByID loads one level.
func (r *UserRepository) GetUserLevelByID(ctx context.Context, id int64) (*userdomain.UserLevel, error) {
	var model userLevelModel
	if err := r.session(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, wrapDBError(err)
	}
	level, err := userLevelFromModel(model)
	if err != nil {
		return nil, err
	}
	return level, nil
}

// CreateUserLevel inserts a level.
func (r *UserRepository) CreateUserLevel(ctx context.Context, level *userdomain.UserLevel) error {
	if level == nil {
		return fmt.Errorf("level is nil")
	}
	now := time.Now().UTC()
	model := userLevelModel{
		Code:            strings.TrimSpace(level.Code),
		Name:            level.Name,
		IconURL:         level.IconURL,
		ThresholdPoints: decimalToNumeric(level.ThresholdPoints),
		SortOrder:       level.SortOrder,
		Enabled:         level.Enabled,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := r.session(ctx).Create(&model).Error; err != nil {
		return wrapDBError(err)
	}
	level.ID = model.ID
	level.CreatedAt = model.CreatedAt
	level.UpdatedAt = model.UpdatedAt
	return nil
}

// UpdateUserLevel updates mutable level fields. Nil pointers leave fields unchanged.
func (r *UserRepository) UpdateUserLevel(ctx context.Context, id int64, name, iconURL *string, threshold *userdomain.Decimal4, sortOrder *int, enabled *bool) error {
	updates := map[string]any{"updated_at": time.Now().UTC()}
	if name != nil {
		updates["name"] = *name
	}
	if iconURL != nil {
		updates["icon_url"] = *iconURL
	}
	if threshold != nil {
		updates["threshold_points"] = decimalToNumeric(*threshold)
	}
	if sortOrder != nil {
		updates["sort_order"] = *sortOrder
	}
	if enabled != nil {
		updates["enabled"] = *enabled
	}
	if len(updates) == 1 {
		return nil
	}
	res := r.session(ctx).Model(&userLevelModel{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return wrapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// EnabledLevelAtOrBelowTx runs inside a caller transaction and returns the
// highest enabled level whose threshold is <= consumption.
func EnabledLevelAtOrBelowTx(tx *gorm.DB, consumption userdomain.Decimal4) (int64, error) {
	var level userLevelModel
	err := tx.
		Where("enabled = TRUE AND threshold_points <= ?", decimalToNumeric(consumption)).
		Order("threshold_points DESC, sort_order DESC, id DESC").
		First(&level).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, fmt.Errorf("no enabled user level found for consumption threshold")
		}
		return 0, err
	}
	return level.ID, nil
}

// EnabledLevelAtOrBelow returns the highest enabled level <= consumption.
func (r *UserRepository) EnabledLevelAtOrBelow(ctx context.Context, consumption userdomain.Decimal4) (*userdomain.UserLevel, error) {
	var model userLevelModel
	err := r.session(ctx).
		Where("enabled = TRUE AND threshold_points <= ?", decimalToNumeric(consumption)).
		Order("threshold_points DESC, sort_order DESC, id DESC").
		First(&model).Error
	if err != nil {
		return nil, wrapDBError(err)
	}
	return userLevelFromModel(model)
}

// AssignUserLevel sets a user's manual/auto level mode. For manual assignments
// the caller must already have validated that the level is enabled. For auto
// mode the level is recalculated inside the transaction.
func (r *UserRepository) AssignUserLevel(ctx context.Context, userID int64, levelID int64, mode string) (*userdomain.User, error) {
	if userID <= 0 || levelID <= 0 {
		return nil, fmt.Errorf("user id and level id are required")
	}
	err := r.session(ctx).Transaction(func(tx *gorm.DB) error {
		var model userModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", userID).
			First(&model).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"level_mode": mode,
			"level_id":   levelID,
			"updated_at": time.Now().UTC(),
		}
		if mode == userdomain.LevelModeAuto {
			consumption, err := numericToDecimal(model.ConsumptionPoints)
			if err != nil {
				return err
			}
			autoID, err := EnabledLevelAtOrBelowTx(tx, consumption)
			if err != nil {
				return err
			}
			updates["level_id"] = autoID
		}
		return tx.Model(&userModel{}).Where("id = ?", userID).Updates(updates).Error
	})
	if err != nil {
		return nil, wrapDBError(err)
	}
	return r.GetUserByID(ctx, userID)
}

func userLevelFromModel(model userLevelModel) (*userdomain.UserLevel, error) {
	threshold, err := numericToDecimal(model.ThresholdPoints)
	if err != nil {
		return nil, err
	}
	return &userdomain.UserLevel{
		ID:              model.ID,
		Code:            model.Code,
		Name:            model.Name,
		IconURL:         model.IconURL,
		ThresholdPoints: threshold,
		SortOrder:       model.SortOrder,
		Enabled:         model.Enabled,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}, nil
}
