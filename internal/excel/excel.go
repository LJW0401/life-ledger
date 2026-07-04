// Package excel owns the bill spreadsheet exchange format. It never persists
// uploaded files and returns structured row/column validation errors.
package excel

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"life-ledger/internal/domain/transactions"

	"github.com/xuri/excelize/v2"
)

var Headers = []string{"日期", "时间", "类型", "金额", "分类", "计入收支", "计入预算", "所属账本", "对象", "账户", "标签", "备注"}

type RowError struct {
	Row    int    `json:"row"`
	Column string `json:"column"`
	Reason string `json:"reason"`
}

type ValidationError struct {
	Errors []RowError
}

func (e ValidationError) Error() string {
	return "excel validation failed"
}

func Template() ([]byte, error) {
	file := excelize.NewFile()
	defer file.Close()
	sheet := file.GetSheetName(0)
	for i, header := range Headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheet, cell, header)
	}
	var buf bytes.Buffer
	if err := file.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Export(items []transactions.Transaction) ([]byte, error) {
	file := excelize.NewFile()
	defer file.Close()
	sheet := file.GetSheetName(0)
	for i, header := range Headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheet, cell, header)
	}
	for rowIndex, item := range items {
		values := []string{
			item.Date,
			item.Time,
			item.Type,
			item.Amount,
			item.Category,
			yesNo(item.IncludeIncome),
			yesNo(item.IncludeBudget),
			item.Ledger,
			item.Counterparty,
			item.Account,
			strings.Join(item.Tags, ","),
			item.Note,
		}
		for colIndex, value := range values {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)
			file.SetCellValue(sheet, cell, value)
		}
	}
	var buf bytes.Buffer
	if err := file.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func ParseImport(r io.Reader, maxRows int) ([]transactions.Input, error) {
	file, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer file.Close()
	sheet := file.GetSheetName(0)
	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ValidationError{Errors: []RowError{{Row: 1, Column: "表头", Reason: "缺少表头"}}}
	}
	if !headerMatches(rows[0]) {
		return nil, ValidationError{Errors: []RowError{{Row: 1, Column: "表头", Reason: "表头与模板不一致"}}}
	}
	if len(rows)-1 > maxRows {
		return nil, ValidationError{Errors: []RowError{{Row: maxRows + 2, Column: "行数", Reason: "超过行数上限"}}}
	}
	inputs := []transactions.Input{}
	errors := []RowError{}
	for i, row := range rows[1:] {
		rowNumber := i + 2
		input, rowErrors := parseRow(rowNumber, row)
		if len(rowErrors) > 0 {
			errors = append(errors, rowErrors...)
			continue
		}
		inputs = append(inputs, input)
	}
	if len(errors) > 0 {
		return nil, ValidationError{Errors: errors}
	}
	return inputs, nil
}

func parseRow(rowNumber int, row []string) (transactions.Input, []RowError) {
	value := func(index int) string {
		if index >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[index])
	}
	input := transactions.Input{
		Date:          value(0),
		Time:          value(1),
		Type:          value(2),
		Amount:        value(3),
		Category:      value(4),
		IncludeIncome: value(5) == "是",
		IncludeBudget: value(6) == "是",
		Ledger:        value(7),
		Counterparty:  value(8),
		Account:       value(9),
		Tags:          splitTags(value(10)),
		Note:          value(11),
	}
	errors := []RowError{}
	required := map[int]string{0: "日期", 1: "时间", 2: "类型", 3: "金额", 4: "分类", 5: "计入收支", 6: "计入预算", 7: "所属账本"}
	for index, column := range required {
		if value(index) == "" {
			errors = append(errors, RowError{Row: rowNumber, Column: column, Reason: "必填"})
		}
	}
	if input.Type != "" && input.Type != "收入" && input.Type != "支出" {
		errors = append(errors, RowError{Row: rowNumber, Column: "类型", Reason: "必须是收入或支出"})
	}
	for _, pair := range []struct {
		index  int
		column string
	}{
		{5, "计入收支"},
		{6, "计入预算"},
	} {
		if v := value(pair.index); v != "" && v != "是" && v != "否" {
			errors = append(errors, RowError{Row: rowNumber, Column: pair.column, Reason: "必须是是或否"})
		}
	}
	if input.Amount != "" {
		if err := transactions.ValidateAmount(input.Amount); err != nil {
			errors = append(errors, RowError{Row: rowNumber, Column: "金额", Reason: "金额必须为大于 0 且最多两位小数的普通数字"})
		}
	}
	return input, errors
}

func headerMatches(row []string) bool {
	if len(row) < len(Headers) {
		return false
	}
	for i, header := range Headers {
		if strings.TrimSpace(row[i]) != header {
			return false
		}
	}
	return true
}

func yesNo(value bool) string {
	if value {
		return "是"
	}
	return "否"
}

func splitTags(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	tags := []string{}
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			tags = append(tags, trimmed)
		}
	}
	return tags
}
