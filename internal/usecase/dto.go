package usecase

import "github.com/jsperandio/RateLimiter/internal/entity"

type CheckRateLimitInputDTO struct {
	IP    string
	Token string
}

type CheckRateLimitOutputDTO struct {
	Allowed   bool
	Kind      entity.RateLimitKind
	Limit     int
	Remaining int
}
