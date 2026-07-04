// These tests lock the spreadsheet import contract at the package boundary.
package excel

import (
	"bytes"
	"errors"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseImportRejectsMalformedAmountWithRowDetails(t *testing.T) {
	for _, amount := range []string{"1.234", "1e3"} {
		inputs, err := ParseImport(bytes.NewReader(importWorkbook(t, amount)), 100)
		if err == nil {
			t.Fatalf("amount %q returned nil error and %#v", amount, inputs)
		}
		var validation ValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("amount %q error = %T, want ValidationError", amount, err)
		}
		if !hasRowError(validation.Errors, 2, "金额") {
			t.Fatalf("amount %q errors = %#v, want row 2 金额", amount, validation.Errors)
		}
	}
}

func importWorkbook(t *testing.T, amount string) []byte {
	t.Helper()
	file := excelize.NewFile()
	defer file.Close()
	sheet := file.GetSheetName(0)
	for i, header := range Headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheet, cell, header)
	}
	values := []string{"2026-07-04", "08:30", "支出", amount, "餐饮", "是", "是", "默认账本", "", "", "", ""}
	for i, value := range values {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		file.SetCellValue(sheet, cell, value)
	}
	var buf bytes.Buffer
	if err := file.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func hasRowError(errors []RowError, row int, column string) bool {
	for _, rowError := range errors {
		if rowError.Row == row && rowError.Column == column {
			return true
		}
	}
	return false
}
