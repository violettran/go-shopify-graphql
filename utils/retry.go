package utils

import (
	"fmt"
	"strings"
	"time"
)

func IsOperationUrlEmpty(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Operation result URL is empty")
}

func IsInvalidTokenError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Invalid API key or access token")
}

func IsInvalidStorefrontTokenError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "401 Unauthorized body")
}

func IsMaxCostLimitError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "max cost limit")
}

func ExecWithRetries(retryCount int, f func() error) error {
	var (
		retries = 0
		err     error
	)
	for {
		err = f()
		if IsInvalidTokenError(err) || IsInvalidStorefrontTokenError(err) || IsOperationUrlEmpty(err) || IsMaxCostLimitError(err) {
			return err
		} else if err != nil {
			retries++
			if retries > retryCount {
				return fmt.Errorf("after %v tries: %w", retries, err)
			}
			time.Sleep(time.Duration(retries) * time.Second)
			continue
		}
		break
	}
	return nil
}
