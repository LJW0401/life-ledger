// These tests cover date-only decision status rules without database setup.
package decisions

import (
	"testing"
	"time"
)

func TestEffectiveStatusUsesLocalDateBoundary(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 7, 4, 0, 30, 0, 0, location)
	item := Decision{Status: "进行中", ReviewDate: "2026-07-04"}
	if got := effectiveStatusAt(item, now, location); got != "待复盘" {
		t.Fatalf("status = %s, want 待复盘", got)
	}

	archived := Decision{Status: "已归档", ReviewDate: "2026-07-04"}
	if got := effectiveStatusAt(archived, now, location); got != "已归档" {
		t.Fatalf("archived status = %s, want 已归档", got)
	}
}

func TestEffectiveStatusUsesConfiguredLocationInsteadOfHostLocal(t *testing.T) {
	oldLocal := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = oldLocal })

	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 7, 3, 16, 30, 0, 0, time.UTC)
	item := Decision{Status: "进行中", ReviewDate: "2026-07-04"}
	if got := effectiveStatusAt(item, now, shanghai); got != "待复盘" {
		t.Fatalf("status = %s, want 待复盘", got)
	}
}
