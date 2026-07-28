package usecase

import "time"

type CheckRateLimitOptions struct {
	Now func() time.Time
}

func NewDefaultCheckRateLimitOptions() *CheckRateLimitOptions {
	return &CheckRateLimitOptions{
		Now: time.Now,
	}
}
