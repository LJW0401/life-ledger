// These tests cover date-only decision status rules without database setup.
package decisions

import (
	"testing"
	"time"
)

func TestEffectiveStatusUsesLocalDateBoundary(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	oldLocal := time.Local
	time.Local = location
	t.Cleanup(func() { time.Local = oldLocal })

	now := time.Date(2026, 7, 4, 0, 30, 0, 0, location)
	item := Decision{Status: "进行中", ReviewDate: "2026-07-04"}
	if got := effectiveStatusAt(item, now); got != "待复盘" {
		t.Fatalf("status = %s, want 待复盘", got)
	}

	archived := Decision{Status: "已归档", ReviewDate: "2026-07-04"}
	if got := effectiveStatusAt(archived, now); got != "已归档" {
		t.Fatalf("archived status = %s, want 已归档", got)
	}
}
