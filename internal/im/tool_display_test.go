package im

import (
	"strings"
	"testing"
)

func TestFormatIMToolLine_pendingWithQuery(t *testing.T) {
	line := FormatIMToolLine(IMToolStep{
		ToolName:  "knowledge_search",
		Pending:   true,
		Arguments: map[string]any{"query": "文明6"},
	})
	if line != "Bilgi bankası araması çalıştırılıyor..." {
		t.Fatalf("pending line = %q", line)
	}
}

func TestFormatIMToolLine_searchDoneWithQueryAndSummary(t *testing.T) {
	line := FormatIMToolLine(IMToolStep{
		ToolName: "knowledge_search",
		Success:  true,
		Arguments: map[string]any{
			"query": "文明6",
		},
		Data: map[string]interface{}{
			"results":   []interface{}{map[string]interface{}{}, map[string]interface{}{}, map[string]interface{}{}},
			"kb_counts": map[string]interface{}{"a": 1, "b": 1},
		},
	})
	if !strings.Contains(line, "Bilgi bankası araması: \"文明6\"") {
		t.Fatalf("title missing query: %q", line)
	}
	if !strings.Contains(line, "2 dosyadan 3 sonuç") {
		t.Fatalf("summary missing: %q", line)
	}
}

func TestFormatIMToolLine_grepPatterns(t *testing.T) {
	line := FormatIMToolLine(IMToolStep{
		ToolName: "grep_chunks",
		Success:  true,
		Arguments: map[string]any{
			"patterns": []any{"文明", "策略"},
		},
		Data: map[string]interface{}{
			"total_matches":  float64(5),
			"document_count": float64(2),
		},
	})
	if line != "Anahtar kelime araması: \"文明, 策略\" · 2 belgeden 5 eşleşen parça" {
		t.Fatalf("grep line = %q", line)
	}
}

func TestFormatIMRagPipelineLine_queryUnderstand(t *testing.T) {
	pending := FormatIMRagPipelineLine(IMToolStep{
		ToolName: "query_understand",
		Pending:  true,
	})
	if pending != "Soru anlaşılıyor..." {
		t.Fatalf("pending = %q", pending)
	}
	done := FormatIMRagPipelineLine(IMToolStep{
		ToolName: "query_understand",
		Success:  true,
	})
	if done != "Soru anlaşıldı" {
		t.Fatalf("done = %q", done)
	}
}

func TestFormatIMRagPipelineLine_searchWithQuery(t *testing.T) {
	line := FormatIMRagPipelineLine(IMToolStep{
		ToolName:  "knowledge_search",
		Pending:   true,
		Arguments: map[string]any{"query": "讯飞开放平台"},
	})
	if line != "Bilgi bankalarında aranıyor: \"讯飞开放平台\"" {
		t.Fatalf("line = %q", line)
	}
}

func TestFormatIMRagPipelineLine_webSearchWithQuery(t *testing.T) {
	line := FormatIMRagPipelineLine(IMToolStep{
		ToolName:  "knowledge_search",
		Pending:   true,
		Arguments: map[string]any{"query": "任素汐演唱会", "search_source": "web"},
	})
	if line != "Web'de aranıyor: \"任素汐演唱会\"" {
		t.Fatalf("line = %q", line)
	}
}

func TestIMGetQueryText_joinsUniqueQueries(t *testing.T) {
	got := imGetQueryText(map[string]any{
		"query":   "foo",
		"queries": []any{"foo", "bar"},
	})
	if got != "foo, bar" {
		t.Fatalf("query text = %q", got)
	}
}

func TestFormatIMToolLine_writeSandboxPendingShowsDiffStat(t *testing.T) {
	line := FormatIMToolLine(IMToolStep{
		ToolName: "write_sandbox_file",
		Pending:  true,
		Arguments: map[string]any{
			"path":          "/workspace/output/a.py",
			"added_lines":   12,
			"removed_lines": 0,
		},
	})
	if line != "Yalıtım alanı dosyasını yaz: \"/workspace/output/a.py\"... +12" {
		t.Fatalf("pending write line = %q", line)
	}
}
