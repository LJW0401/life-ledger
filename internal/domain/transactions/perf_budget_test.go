//go:build !race

package transactions

import "time"

func releasePerfBudget() time.Duration {
	return 300 * time.Millisecond
}
