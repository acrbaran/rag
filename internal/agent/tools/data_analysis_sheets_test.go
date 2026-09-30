package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestListXLSXSheetNamesKeepsWorkbookOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "order.xlsx")
	order := []string{"Özet", "Q1'24", "报表", "Ham Veri"}
	sheets := map[string][][]any{}
	for _, name := range order {
		sheets[name] = [][]any{{"k"}, {"v"}}
	}
	writeWorkbook(t, path, sheets, order)

	got, err := listXLSXSheetNames(path)
	if err != nil {
		t.Fatalf("listXLSXSheetNames: %v", err)
	}
	if !reflect.DeepEqual(got, order) {
		t.Fatalf("sheet names = %q, want %q", got, order)
	}
}

func TestListXLSXSheetNamesRejectsNonPackage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.xls")
	if err := os.WriteFile(path, []byte("not a zip package"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := listXLSXSheetNames(path); err == nil {
		t.Fatal("expected an error for a non-OOXML file so the st_read_meta fallback runs")
	}
}

func TestAlignSheetsWithWorkbookReportsEmptySheets(t *testing.T) {
	stats := []SheetInfo{{Name: "B", RowCount: 3}, {Name: "A", RowCount: 2}}
	got := alignSheetsWithWorkbook(stats, []string{"A", "Empty", "B"})
	want := []SheetInfo{{Name: "A", RowCount: 2}, {Name: "Empty"}, {Name: "B", RowCount: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("aligned = %+v, want %+v", got, want)
	}
	if got := alignSheetsWithWorkbook(stats, nil); !reflect.DeepEqual(got, stats) {
		t.Fatalf("without workbook order the statistics must be kept as-is, got %+v", got)
	}
}

func TestTableSchemaDescriptionListsEverySheet(t *testing.T) {
	schema := &TableSchema{
		TableName: "k_doc",
		Columns: []ColumnInfo{
			{Name: "id", Type: "VARCHAR"}, {Name: "amount", Type: "VARCHAR"},
			{Name: "sku", Type: "VARCHAR"}, {Name: excelSheetNameColumn, Type: "VARCHAR"},
		},
		RowCount: 5,
		Sheets: []SheetInfo{
			{Name: "Sales", RowCount: 2, Columns: []string{"id", "amount"}},
			{Name: "Inventory", RowCount: 3, Columns: []string{"sku"}},
			{Name: "Notes", RowCount: 0},
		},
		Notes: []string{`Sheet "Chart" could not be read and was skipped (x).`},
	}
	got := schema.Description()
	for _, want := range []string{
		"Table name: dataset",
		"This table combines 3 Excel sheets",
		`- "Sales": 2 rows; columns: id, amount`,
		`- "Inventory": 3 rows; columns: sku`,
		`- "Notes": 0 rows (empty)`,
		`WHERE "__sheet_name" = '<sheet name>'`,
		`GROUP BY "__sheet_name"`,
		`Sheet "Chart" could not be read`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("description missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "k_doc") {
		t.Fatalf("description leaks the physical table name:\n%s", got)
	}
}

func TestTableSchemaDescriptionWithoutSheetsIsUnchanged(t *testing.T) {
	schema := &TableSchema{TableName: "k_doc", Columns: []ColumnInfo{{Name: "id", Type: "VARCHAR"}}, RowCount: 1}
	want := "Table name: dataset\nColumns: 1\nRows: 1\n\nColumn info:\n- id (VARCHAR)\n"
	if got := schema.Description(); got != want {
		t.Fatalf("description = %q, want %q", got, want)
	}
}

// createSheetTable builds a table shaped like a multi-sheet Excel load without
// needing the excel extension.
func createSheetTable(t *testing.T, tool *DataAnalysisTool, table string, perSheet map[string]int, order []string) {
	t.Helper()
	ctx := context.Background()
	if _, err := tool.db.ExecContext(ctx, fmt.Sprintf(
		`CREATE TABLE "%s" (id VARCHAR, amount VARCHAR, sku VARCHAR, "%s" VARCHAR)`, table, excelSheetNameColumn,
	)); err != nil {
		t.Fatalf("create table: %v", err)
	}
	tool.recordCreatedTable(table)
	for _, sheet := range order {
		for i := 0; i < perSheet[sheet]; i++ {
			var err error
			if sheet == "Inventory" {
				_, err = tool.db.ExecContext(ctx, fmt.Sprintf(
					`INSERT INTO "%s" (sku, "%s") VALUES (?, ?)`, table, excelSheetNameColumn), fmt.Sprint("S", i), sheet)
			} else {
				_, err = tool.db.ExecContext(ctx, fmt.Sprintf(
					`INSERT INTO "%s" (id, amount, "%s") VALUES (?, ?, ?)`, table, excelSheetNameColumn), fmt.Sprint(i), "10", sheet)
			}
			if err != nil {
				t.Fatalf("insert: %v", err)
			}
		}
	}
}

func TestLoadFromTableComputesPerSheetStatistics(t *testing.T) {
	tool := &DataAnalysisTool{BaseTool: dataAnalysisTool, db: newPlainDuckDB(t), sessionID: "test-sheet-stats"}
	ctx := context.Background()
	t.Cleanup(func() { tool.Cleanup(ctx) })
	createSheetTable(t, tool, "k_sheets", map[string]int{"Sales": 2, "Inventory": 3}, []string{"Sales", "Inventory"})

	schema, err := tool.LoadFromTable(ctx, "k_sheets")
	if err != nil {
		t.Fatalf("LoadFromTable: %v", err)
	}
	want := []SheetInfo{
		{Name: "Sales", RowCount: 2, Columns: []string{"id", "amount"}},
		{Name: "Inventory", RowCount: 3, Columns: []string{"sku"}},
	}
	if !reflect.DeepEqual(schema.Sheets, want) {
		t.Fatalf("sheets = %+v, want %+v", schema.Sheets, want)
	}
}

func TestSampleRowsCoversEverySheet(t *testing.T) {
	tool := &DataAnalysisTool{BaseTool: dataAnalysisTool, db: newPlainDuckDB(t), sessionID: "test-sample"}
	ctx := context.Background()
	t.Cleanup(func() { tool.Cleanup(ctx) })
	order := []string{"Sales", "Inventory", "Returns"}
	createSheetTable(t, tool, "k_sample", map[string]int{"Sales": 20, "Inventory": 20, "Returns": 20}, order)
	schema, err := tool.LoadFromTable(ctx, "k_sample")
	if err != nil {
		t.Fatalf("LoadFromTable: %v", err)
	}

	countBySheet := func(rows []map[string]string) map[string]int {
		counts := map[string]int{}
		for _, row := range rows {
			counts[row[excelSheetNameColumn]]++
		}
		return counts
	}

	rows, err := tool.SampleRows(ctx, schema, 5, 40)
	if err != nil {
		t.Fatalf("SampleRows: %v", err)
	}
	if got := countBySheet(rows); !reflect.DeepEqual(got, map[string]int{"Sales": 5, "Inventory": 5, "Returns": 5}) {
		t.Fatalf("per-sheet sample = %v", got)
	}

	// A tight cap takes fewer rows per sheet instead of dropping sheets.
	rows, err = tool.SampleRows(ctx, schema, 5, 6)
	if err != nil {
		t.Fatalf("SampleRows capped: %v", err)
	}
	if got := countBySheet(rows); !reflect.DeepEqual(got, map[string]int{"Sales": 2, "Inventory": 2, "Returns": 2}) {
		t.Fatalf("capped per-sheet sample = %v", got)
	}
}

func TestSampleRowsWithoutSheetsHonoursLimit(t *testing.T) {
	tool := &DataAnalysisTool{BaseTool: dataAnalysisTool, db: newPlainDuckDB(t), sessionID: "test-sample-csv"}
	ctx := context.Background()
	t.Cleanup(func() { tool.Cleanup(ctx) })
	schema, err := tool.LoadFromCSV(ctx, writeCSV(t, "s.csv", "id", "1", "2", "3", "4"), "k_csv")
	if err != nil {
		t.Fatalf("LoadFromCSV: %v", err)
	}
	if len(schema.Sheets) != 0 {
		t.Fatalf("CSV tables must not report sheets, got %+v", schema.Sheets)
	}
	rows, err := tool.SampleRows(ctx, schema, 5, 3)
	if err != nil {
		t.Fatalf("SampleRows: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
}

// TestLoadFromExcel_ReportsEverySheet guards the ingest/description path: the
// schema must list every sheet (including an empty one) in workbook order, so
// the model is told the workbook has more than its first sheet.
func TestLoadFromExcel_ReportsEverySheet(t *testing.T) {
	db := newTestDuckDB(t)
	path := filepath.Join(t.TempDir(), "report.xlsx")
	writeWorkbook(t, path,
		map[string][][]any{
			"Sorular":  {{"no", "soru"}, {1, "a"}, {2, "b"}},
			"Boş":      {},
			"Cevaplar": {{"no", "cevap"}, {1, "x"}},
		},
		[]string{"Sorular", "Boş", "Cevaplar"},
	)
	tool := &DataAnalysisTool{BaseTool: dataAnalysisTool, db: db, sessionID: "test-report-sheets"}
	ctx := context.Background()
	t.Cleanup(func() { tool.Cleanup(ctx) })

	schema, err := tool.LoadFromExcel(ctx, path, "t_report")
	if err != nil {
		t.Fatalf("LoadFromExcel: %v", err)
	}
	if schema.RowCount != 3 {
		t.Fatalf("row count = %d, want 3", schema.RowCount)
	}
	names := make([]string, 0, len(schema.Sheets))
	for _, sheet := range schema.Sheets {
		names = append(names, sheet.Name)
	}
	if !reflect.DeepEqual(names, []string{"Sorular", "Boş", "Cevaplar"}) {
		t.Fatalf("sheets = %+v", schema.Sheets)
	}
	desc := schema.Description()
	for _, want := range []string{`"Sorular": 2 rows`, `"Cevaplar": 1 rows`, "combines 3 Excel sheets"} {
		if !strings.Contains(desc, want) {
			t.Errorf("description missing %q:\n%s", want, desc)
		}
	}
}
