package utils

import (
	"fmt"
	"time"
)

func ExecWithRetries(retryCount int, f func() error) error {
	var (
		retries = 0
		err     error
	)
	for {
		err = f()
		if err != nil {
			retries++
			if retries > retryCount {
				return fmt.Errorf("query products after %v retries: %w", retries, err)
			}
			time.Sleep(time.Duration(retries) * time.Second)
			continue
		}
		break
	}
	return nil
}
