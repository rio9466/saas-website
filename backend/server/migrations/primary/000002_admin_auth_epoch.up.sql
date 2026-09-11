-- Auth epoch generation for administrators.
--
-- Every security-sensitive account/credential transition (password change,
-- password reset, enable, disable) increments auth_epoch on the affected
-- administrator. Issued Redis sessions record the epoch they were created
-- under, so protected requests and refreshes can reject sessions that were
-- issued before the transition. Re-enabling never restores an old epoch, so
-- pre-disable sessions can never become valid again.
ALTER TABLE administrators
    ADD COLUMN auth_epoch BIGINT NOT NULL DEFAULT 0;
