package nftables

import "errors"

var (
	ErrInvalidIP = errors.New("invalid IP address")
	ErrNotFound  = errors.New("IP address not in blacklist")
)
