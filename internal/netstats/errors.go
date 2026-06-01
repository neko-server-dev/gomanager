package netstats

import "errors"

var ErrNotSupported = errors.New("network statistics are only supported on Linux")
