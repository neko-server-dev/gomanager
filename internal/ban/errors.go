package ban

import "errors"

var (
	ErrInvalidTTL       = errors.New("ttl must be a positive duration (e.g. 1h, 30m)")
	ErrInvalidExpiresAt = errors.New("expires_at must be an RFC3339 timestamp in the future")
)
