package utils

import (
	"context"
	"time"
)

const DefaultTimeout = 10 * time.Minute

func NewContext() (ctx context.Context, cancel func()) {
	return NewContextWithTimeout(DefaultTimeout)
}

func NewContextWithTimeout(timeout time.Duration) (ctx context.Context, cancel func()) {
	return context.WithTimeout(context.TODO(), timeout)
}
