package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/acrbaran/rag/internal/types"
)

type fakeLiveSchema struct {
	schema   *TableSchema
	err      error
	cleanups int
}

func (f *fakeLiveSchema) LoadFromKnowledge(context.Context, *types.Knowledge) (*TableSchema, error) {
	return f.schema, f.err
}

func (f *fakeLiveSchema) Cleanup(context.Context) { f.cleanups++ }

func newDataSchemaFixture(chunks []*types.Chunk) (*DataSchemaTool, json.RawMessage) {
	knowledge := &types.Knowledge{ID: "doc-1", FileType: "xlsx"}
	tool := NewDataSchemaTool(
		&mapKnowledgeService{docs: map[string]*types.Knowledge{knowledge.ID: knowledge}},
		&readDocChunkRepo{ordered: chunks},
	)
	args, _ := json.Marshal(DataSchemaInput{KnowledgeID: knowledge.ID})
	return tool, args
}

func storedSchemaChunks() []*types.Chunk {
	return []*types.Chunk{
		{ChunkType: types.ChunkTypeTableSummary, Content: "# Table Summary\n\nfirst sheet only"},
		{ChunkType: types.ChunkTypeTableColumn, Content: "# Table Column Information\n\ncolumns"},
	}
}

func multiSheetSchema() *TableSchema {
	return &TableSchema{
		TableName: "k_doc_1",
		Columns:   []ColumnInfo{{Name: "no", Type: "VARCHAR"}, {Name: excelSheetNameColumn, Type: "VARCHAR"}},
		RowCount:  7,
		Sheets:    []SheetInfo{{Name: "Sorular", RowCount: 4}, {Name: "Cevaplar", RowCount: 3}},
	}
}

func TestDataSchemaAppendsLiveSheetStructure(t *testing.T) {
	tool, args := newDataSchemaFixture(storedSchemaChunks())
	live := &fakeLiveSchema{schema: multiSheetSchema()}
	tool.WithLiveSchema(live)

	result, err := tool.Execute(context.Background(), args)
	if err != nil || !result.Success {
		t.Fatalf("execute: result=%+v err=%v", result, err)
	}
	for _, want := range []string{"first sheet only", "# Current Table Structure", `"Sorular": 4 rows`, `"Cevaplar": 3 rows`, `table "dataset"`} {
		if !strings.Contains(result.Output, want) {
			t.Errorf("output missing %q:\n%s", want, result.Output)
		}
	}
	tool.Cleanup(context.Background())
	if live.cleanups != 1 {
		t.Fatalf("live schema loader was not cleaned up")
	}
}

func TestDataSchemaUsesLiveSchemaWhenNothingIsStored(t *testing.T) {
	tool, args := newDataSchemaFixture(nil)
	tool.WithLiveSchema(&fakeLiveSchema{schema: multiSheetSchema()})

	result, err := tool.Execute(context.Background(), args)
	if err != nil || !result.Success {
		t.Fatalf("execute: result=%+v err=%v", result, err)
	}
	if !strings.Contains(result.Output, "combines 2 Excel sheets") {
		t.Fatalf("output missing live structure:\n%s", result.Output)
	}
}

func TestDataSchemaFallsBackToStoredSummaryWhenLiveLoadFails(t *testing.T) {
	tool, args := newDataSchemaFixture(storedSchemaChunks())
	tool.WithLiveSchema(&fakeLiveSchema{err: errors.New("excel extension missing")})

	result, err := tool.Execute(context.Background(), args)
	if err != nil || !result.Success {
		t.Fatalf("execute: result=%+v err=%v", result, err)
	}
	if strings.Contains(result.Output, "Current Table Structure") || !strings.Contains(result.Output, "first sheet only") {
		t.Fatalf("unexpected output:\n%s", result.Output)
	}
}

func TestDataSchemaWithoutAnySchemaFails(t *testing.T) {
	tool, args := newDataSchemaFixture(nil)
	result, err := tool.Execute(context.Background(), args)
	if err == nil || result.Success {
		t.Fatalf("expected failure without stored or live schema: result=%+v", result)
	}
}
