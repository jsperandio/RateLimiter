package usecase

import (
	"context"
	"time"

	"github.com/jsperandio/RateLimiter/internal/entity"
)

type CheckRateLimitUseCase struct {
	storage entity.LimiterStorage
	limits  entity.Limits

	options *CheckRateLimitOptions
}

func NewCheckRateLimitUseCase(storage entity.LimiterStorage, limits entity.Limits, opt *CheckRateLimitOptions) *CheckRateLimitUseCase {
	if opt == nil {
		opt = NewDefaultCheckRateLimitOptions()
	}

	if opt.Now == nil {
		opt.Now = time.Now
	}

	return &CheckRateLimitUseCase{
		storage: storage,
		limits:  limits,
		options: opt,
	}
}

func (uc *CheckRateLimitUseCase) Execute(ctx context.Context, input CheckRateLimitInputDTO) (CheckRateLimitOutputDTO, error) {
	rt, err := entity.ResolveLimit(input.IP, input.Token, uc.limits)
	if err != nil {
		return CheckRateLimitOutputDTO{}, err
	}

	output := CheckRateLimitOutputDTO{
		Kind:  rt.Kind,
		Limit: rt.MaxRequests,
	}

	blocked, err := uc.storage.IsBlocked(ctx, rt.BlockKey())
	if err != nil {
		return CheckRateLimitOutputDTO{}, err
	}

	if blocked {
		return output, nil
	}

	count, err := uc.storage.Increment(ctx, rt.CounterKey(uc.options.Now()), entity.CounterWindow)
	if err != nil {
		return CheckRateLimitOutputDTO{}, err
	}

	if count > int64(rt.MaxRequests) {
		if err := uc.storage.Block(ctx, rt.BlockKey(), rt.BlockDuration); err != nil {
			return CheckRateLimitOutputDTO{}, err
		}

		return output, nil
	}

	output.Allowed = true
	output.Remaining = rt.MaxRequests - int(count)

	return output, nil
}
