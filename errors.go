package shopify

import (
	"errors"

	"github.com/gempages/go-shopify-graphql/graphql"
)

func IsPaymentRequiredError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, graphql.ErrPaymentRequired)
}

func IsLockedError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, graphql.ErrLocked)
}

func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, graphql.ErrMaxCostExceeded)
}
