package usersvc

import (
	"context"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

// AdjustPointsInput for the atomic ledger operation.
type AdjustPointsInput struct {
	UserID           int64
	PointsDelta      userdomain.Decimal4
	ConsumptionDelta userdomain.Decimal4
	Reason           string
	IdempotencyKey   string
}

// AdjustPoints applies an exact, atomic, idempotent available/consumption
// delta with a required reason and unique idempotency key, under the
// pending-first audit policy. Retrying the same key is rejected as an
// idempotency conflict and can never apply the change twice.
func (s *Service) AdjustPoints(ctx context.Context, actor Actor, in AdjustPointsInput) (*userdomain.PointTransaction, error) {
	if err := s.requirePermission(ctx, actor, permissionCustomerPoints); err != nil {
		return nil, err
	}
	if in.UserID <= 0 {
		return nil, validationError("invalid user id")
	}
	if in.Reason == "" {
		return nil, validationError("reason is required")
	}
	if len(in.Reason) > 200 {
		return nil, validationError("reason must be at most 200 characters")
	}
	if in.IdempotencyKey == "" {
		return nil, validationError("idempotency key is required")
	}
	if len(in.IdempotencyKey) > 128 {
		return nil, validationError("idempotency key must be at most 128 characters")
	}

	actorID := int64(0)
	if actor.AdminID > 0 {
		actorID = actor.AdminID
	} else {
		actorID = actor.ID
	}

	var result *userdomain.PointTransaction
	err := s.withPendingAudit(ctx, actor, ActionUserPointsAdjust, ResourceUser, idString(in.UserID), map[string]any{
		"user_id":           idString(in.UserID),
		"points_delta":      in.PointsDelta.String(),
		"consumption_delta": in.ConsumptionDelta.String(),
		"reason":            in.Reason,
		"idempotency_key":   in.IdempotencyKey,
	}, func() error {
		pt, err := s.users.AdjustPoints(ctx, in.UserID, &actorID, in.PointsDelta, in.ConsumptionDelta, in.Reason, in.IdempotencyKey)
		if err != nil {
			if err == primary.ErrIdempotencyReplay {
				return apperr.New(409, apperr.CodeIdempotencyConflict, "idempotency conflict", err)
			}
			return err
		}
		result = pt
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ListPointTransactions returns the immutable ledger page for one user.
func (s *Service) ListPointTransactions(ctx context.Context, actor Actor, userID int64, page, pageSize int) (adminauth.Page[userdomain.PointTransaction], error) {
	if err := s.requirePermission(ctx, actor, permissionCustomerPoints); err != nil {
		return adminauth.Page[userdomain.PointTransaction]{}, err
	}
	if userID <= 0 {
		return adminauth.Page[userdomain.PointTransaction]{}, validationError("invalid user id")
	}
	out, err := s.users.ListPointTransactions(ctx, userID, page, pageSize)
	if err != nil {
		return adminauth.Page[userdomain.PointTransaction]{}, mapError(err)
	}
	return out, nil
}
