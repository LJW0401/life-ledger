//go:build race

package transactions

import "time"

func releasePerfBudget() time.Duration {
	return time.Second
}
