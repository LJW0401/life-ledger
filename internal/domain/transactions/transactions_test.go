package transactions

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"life-ledger/internal/config"
	"life-ledger/internal/db"
	"life-ledger/internal/domain/tags"
)

func TestListAndSummaryHandleReleaseDataVolume(t *testing.T) {
	conn := testDB(t)
	service := Service{DB: conn, Tags: tags.Store{DB: conn}}
	inputs := make([]Input, 0, 5000)
	for i := 0; i < 5000; i++ {
		kind := "支出"
		category := "餐饮"
		if i%10 == 0 {
			kind = "收入"
			category = "工资"
		}
		inputs = append(inputs, Input{
			Date:          "2026-07-04",
			Time:          "08:30",
			Type:          kind,
			Amount:        "1.00",
			Category:      category,
			IncludeIncome: true,
			IncludeBudget: true,
			Ledger:        "默认账本",
		})
	}
	if err := service.CreateMany(context.Background(), inputs); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	list, err := service.List(context.Background(), Filter{Page: 1, PageSize: 500})
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > releasePerfBudget() {
		t.Fatalf("list exceeded release budget: %s", elapsed)
	}
	if list.PageSize != 200 || len(list.Items) != 200 || list.Total != 5000 {
		t.Fatalf("unexpected list page: page_size=%d len=%d total=%d", list.PageSize, len(list.Items), list.Total)
	}

	start = time.Now()
	summary, err := service.Summary(context.Background(), Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > releasePerfBudget() {
		t.Fatalf("summary exceeded release budget: %s", elapsed)
	}
	if summary.Income != "500.00" || summary.Expense != "4500.00" || summary.Balance != "-4000.00" {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if len(summary.ByCategory) != 1 || summary.ByCategory[0].Category != "餐饮" || summary.ByCategory[0].Expense != "4500.00" {
		t.Fatalf("unexpected category summary: %#v", summary.ByCategory)
	}
}

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	conn, err := db.Open(config.Config{
		Data: config.DataConfig{Dir: filepath.Join(dir), Database: "life-ledger.db"},
		Auth: config.AuthConfig{Username: "admin", PasswordHash: "hash", SessionSecret: "01234567890123456789012345678901", SessionDays: 7},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}
