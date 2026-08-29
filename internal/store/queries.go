package store

const GetAdminPasswordHash = `SELECT password_hash FROM admin WHERE lock_key = TRUE`

const GetAdminSession = `
SELECT
	access_token_hash,
	refresh_token_hash,
	access_token_expires_at,
	refresh_token_expires_at
FROM admin
WHERE lock_key = TRUE
`

const UpdateAdminSession = `
UPDATE admin SET
	access_token_hash = $1,
	refresh_token_hash = $2,
	access_token_expires_at = $3,
	refresh_token_expires_at = $4,
	updated_at = NOW()
WHERE lock_key = TRUE
`
