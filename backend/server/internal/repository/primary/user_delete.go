package primary

import "context"

// DeleteUserBestEffort removes a business user row when a registration could
// not be completed (e.g. the verification email could not be sent). Only rows
// created moments ago in the same request are ever candidates; ledger rows
// cascade on delete. Errors are returned as-is so the caller can log safely.
func (r *UserRepository) DeleteUserBestEffort(ctx context.Context, userID int64) error {
	res := r.session(ctx).Where("id = ?", userID).Delete(&userModel{})
	if res.Error != nil {
		return wrapDBError(res.Error)
	}
	return nil
}
