package usersvc

import (
	"context"
	"strings"

	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
)

// AssignUserLevelInput: manual mode requires an enabled level id; auto mode
// recalculates immediately from cumulative consumption.
type AssignUserLevelInput struct {
	UserID  int64
	LevelID int64
	Mode    string
}

// CreateUserLevelInput for level management.
type CreateUserLevelInput struct {
	Code            string
	Name            string
	IconURL         string
	ThresholdPoints userdomain.Decimal4
	SortOrder       int
	Enabled         bool
}

// UpdateUserLevelInput with pointers for selective updates.
type UpdateUserLevelInput struct {
	Name            *string
	IconURL         *string
	ThresholdPoints *userdomain.Decimal4
	SortOrder       *int
	Enabled         *bool
}

// ListUserLevels returns every level (disabled included for history).
func (s *Service) ListUserLevels(ctx context.Context, actor Actor) ([]userdomain.UserLevel, error) {
	if err := s.requirePermission(ctx, actor, permissionUserLevelRead); err != nil {
		return nil, err
	}
	out, err := s.users.ListUserLevels(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return out, nil
}

// GetUserLevel returns one level.
func (s *Service) GetUserLevel(ctx context.Context, actor Actor, id int64) (*userdomain.UserLevel, error) {
	if err := s.requirePermission(ctx, actor, permissionUserLevelRead); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, validationError("invalid level id")
	}
	level, err := s.users.GetUserLevelByID(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return level, nil
}

// CreateUserLevel inserts a level. Enabled-threshold ambiguity is rejected by
// the partial unique index on (threshold_points) WHERE enabled.
func (s *Service) CreateUserLevel(ctx context.Context, actor Actor, in CreateUserLevelInput) (*userdomain.UserLevel, error) {
	if err := s.requirePermission(ctx, actor, permissionUserLevelManage); err != nil {
		return nil, err
	}
	code := strings.TrimSpace(in.Code)
	name := strings.TrimSpace(in.Name)
	if err := validateLevelCode(code); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, validationError("level name is required")
	}
	if in.ThresholdPoints.IsNegative() {
		return nil, validationError("threshold points must be zero or greater")
	}

	level := &userdomain.UserLevel{
		Code:            code,
		Name:            name,
		IconURL:         strings.TrimSpace(in.IconURL),
		ThresholdPoints: in.ThresholdPoints,
		SortOrder:       in.SortOrder,
		Enabled:         in.Enabled,
	}
	newID, err := s.createAudited(ctx, actor, ActionUserLevelCreate, ResourceUserLevel, map[string]any{
		"level_code":       code,
		"level_name":       name,
		"threshold_points": in.ThresholdPoints.String(),
		"sort_order":       in.SortOrder,
		"enabled":          in.Enabled,
	}, func() (int64, error) {
		if err := s.users.CreateUserLevel(ctx, level); err != nil {
			return 0, err
		}
		return level.ID, nil
	})
	if err != nil {
		return nil, err
	}
	level.ID = newID
	return level, nil
}

// UpdateUserLevel updates a level. The settings default level cannot be
// disabled because new users resolve their level from it.
func (s *Service) UpdateUserLevel(ctx context.Context, actor Actor, id int64, in UpdateUserLevelInput) (*userdomain.UserLevel, error) {
	if err := s.requirePermission(ctx, actor, permissionUserLevelManage); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, validationError("invalid level id")
	}
	if in.Name != nil && strings.TrimSpace(*in.Name) == "" {
		return nil, validationError("level name must not be empty")
	}
	if in.ThresholdPoints != nil && in.ThresholdPoints.IsNegative() {
		return nil, validationError("threshold points must be zero or greater")
	}

	var saved *userdomain.UserLevel
	err := s.withPendingAudit(ctx, actor, ActionUserLevelUpdate, ResourceUserLevel, idString(id), map[string]any{
		"level_id": idString(id),
	}, func() error {
		// Block disabling the settings default level (new-user assignment).
		if in.Enabled != nil && !*in.Enabled {
			settings, err := s.users.GetSystemSettings(ctx)
			if err != nil {
				return err
			}
			if settings.DefaultLevelID == id {
				return apperr.New(400, apperr.CodeValidation, "the default user level cannot be disabled", apperr.ErrValidation)
			}
		}
		if err := s.users.UpdateUserLevel(ctx, id, in.Name, in.IconURL, in.ThresholdPoints, in.SortOrder, in.Enabled); err != nil {
			return err
		}
		level, err := s.users.GetUserLevelByID(ctx, id)
		if err != nil {
			return err
		}
		saved = level
		return nil
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}

// AssignUserLevel sets a manual level or switches back to automatic
// recalculation (computed in the same transaction).
func (s *Service) AssignUserLevel(ctx context.Context, actor Actor, in AssignUserLevelInput) (*userdomain.User, error) {
	if err := s.requirePermission(ctx, actor, permissionCustomerLevelAssign); err != nil {
		return nil, err
	}
	if in.UserID <= 0 {
		return nil, validationError("invalid user id")
	}
	mode := strings.TrimSpace(in.Mode)
	switch mode {
	case userdomain.LevelModeAuto:
		// levelID ignored; recompute below.
	case userdomain.LevelModeManual:
		if in.LevelID <= 0 {
			return nil, validationError("manual level requires a level id")
		}
		level, err := s.users.GetUserLevelByID(ctx, in.LevelID)
		if err != nil {
			return nil, mapError(err)
		}
		if !level.Enabled {
			return nil, validationError("a disabled level cannot be assigned")
		}
	default:
		return nil, validationError("level mode must be auto or manual")
	}

	// Auto mode resolves the level inside the repository transaction.
	effectiveLevelID := in.LevelID
	if mode == userdomain.LevelModeAuto {
		u, err := s.users.GetUserByID(ctx, in.UserID)
		if err != nil {
			return nil, mapError(err)
		}
		effectiveLevelID, err = s.resolveAutoLevel(ctx, u.ConsumptionPoints)
		if err != nil {
			return nil, err
		}
	}

	var saved *userdomain.User
	err := s.withPendingAudit(ctx, actor, ActionUserLevelAssign, ResourceUser, idString(in.UserID), map[string]any{
		"user_id":    idString(in.UserID),
		"level_id":   idString(effectiveLevelID),
		"level_mode": mode,
	}, func() error {
		u, err := s.users.AssignUserLevel(ctx, in.UserID, effectiveLevelID, mode)
		if err != nil {
			return err
		}
		saved = u
		return nil
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}

func (s *Service) resolveAutoLevel(ctx context.Context, consumption userdomain.Decimal4) (int64, error) {
	level, err := s.users.EnabledLevelAtOrBelow(ctx, consumption)
	if err != nil {
		return 0, mapError(err)
	}
	return level.ID, nil
}

func validateLevelCode(code string) error {
	if len(code) < 1 || len(code) > 64 {
		return validationError("level code must be 1-64 characters")
	}
	for _, r := range code {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') &&
			r != '.' && r != '_' && r != '-' {
			return validationError("level code may only contain letters, digits, '.', '_', '-'")
		}
	}
	return nil
}
