package customerqa

import "errors"

var (
	ErrInvalidArgument = errors.New("invalid_argument")
	ErrNotFound        = errors.New("not_found")
)
