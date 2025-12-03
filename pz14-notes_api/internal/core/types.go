package core

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

// Now возвращает текущее время в UTC
func Now() time.Time {
	return time.Now().UTC()
}
