package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/types"
	"github.com/acrbaran/rag/internal/types/interfaces"
	"github.com/acrbaran/rag/internal/utils"
)

var dataSchemaTool = BaseTool{
	name: ToolDataSchema,
	description: "Use this tool to get the schema information of a CSV or Excel file loaded into DuckDB. " +
		"It returns the columns and row count, and for Excel workbooks every sheet with its row count " +
		"and columns. When querying the document with data_analysis, " +
		"reference it as the table \"" + DataAnalysisTableName + "\".",
	schema: utils.GenerateSchema[DataSchemaInput](),
}

type DataSchemaInput struct {
	KnowledgeID string `json:"knowledge_id" jsonschema:"short dN document ID to query"`
}

type DataSchemaTool struct {
	BaseTool
	knowledgeService interfaces.KnowledgeService
	chunkRepo        interfaces.ChunkRepository
	targetChunkTypes []types.ChunkType
	searchTargets    types.SearchTargets
	scopeEnforced    bool
	// liveSchema, when set, loads the document itself so the answer reflects
	// its current structure (every sheet, exact column names) even when the
	// stored summary predates multi-sheet support or covers only part of it.
	liveSchema LiveSchemaLoader
}

// LiveSchemaLoader loads a CSV/Excel document and returns its schema.
// DataAnalysisTool implements it.
type LiveSchemaLoader interface {
	LoadFromKnowledge(ctx context.Context, knowledge *types.Knowledge) (*TableSchema, error)
	Cleanup(ctx context.Context)
}

// WithLiveSchema makes data_schema load the document and append its live
// structure. Sharing the loader with data_analysis lets both reuse one loaded
// table per session.
func (t *DataSchemaTool) WithLiveSchema(loader LiveSchemaLoader) *DataSchemaTool {
	t.liveSchema = loader
	return t
}

// Cleanup releases the tables loaded for the live schema. The loader's
// Cleanup is idempotent, so a loader shared with data_analysis is safe.
func (t *DataSchemaTool) Cleanup(ctx context.Context) {
	if t.liveSchema != nil {
		t.liveSchema.Cleanup(ctx)
	}
}

// WithSearchTargets enables Agent request-scope authorization. An Agent turn
// with no search target must reject every document rather than fall back to
// the unrestricted service-owned lookup.
func (t *DataSchemaTool) WithSearchTargets(searchTargets types.SearchTargets) *DataSchemaTool {
	t.searchTargets = searchTargets
	t.scopeEnforced = true
	return t
}

func NewDataSchemaTool(knowledgeService interfaces.KnowledgeService, chunkRepo interfaces.ChunkRepository, targetChunkTypes ...types.ChunkType) *DataSchemaTool {
	if len(targetChunkTypes) == 0 {
		targetChunkTypes = []types.ChunkType{types.ChunkTypeTableSummary, types.ChunkTypeTableColumn}
	}
	return &DataSchemaTool{
		BaseTool:         dataSchemaTool,
		knowledgeService: knowledgeService,
		chunkRepo:        chunkRepo,
		targetChunkTypes: targetChunkTypes,
	}
}

// Execute executes the tool logic
func (t *DataSchemaTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	var input DataSchemaInput
	if err := json.Unmarshal(args, &input); err != nil {
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse input args: %v", err),
		}, err
	}

	// Get knowledge to get TenantID (use IDOnly to support cross-tenant shared KB)
	var knowledge *types.Knowledge
	var err error
	if t.scopeEnforced {
		knowledge, err = authorizeKnowledgeInSearchTargets(ctx, t.searchTargets, input.KnowledgeID, t.knowledgeService)
	} else {
		knowledge, err = t.knowledgeService.GetKnowledgeByIDOnly(ctx, input.KnowledgeID)
	}
	if err != nil || knowledge == nil {
		if err == nil {
			err = fmt.Errorf("knowledge service returned an empty result")
		}
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to get knowledge '%s': %v", input.KnowledgeID, err),
		}, err
	}

	// Get chunks for the knowledge ID using ChunkRepository
	// We only need table summary and column chunks
	chunkTypes := t.targetChunkTypes
	page := &types.Pagination{
		Page:     1,
		PageSize: 100, // Should be enough for schema chunks
	}
	enabled := true

	chunks, _, err := t.chunkRepo.ListPagedChunksByKnowledgeID(
		ctx,
		knowledge.TenantID,
		input.KnowledgeID,
		page,
		chunkTypes,
		nil, // tagIDs
		"",  // keyword
		"",  // searchField
		"",  // sortOrder
		"",  // knowledgeType
		&enabled,
	)
	if err != nil {
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to list chunks for knowledge ID '%s': %v", input.KnowledgeID, err),
		}, err
	}

	var summaryContent, columnContent string
	for _, chunk := range chunks {
		if chunk.ChunkType == types.ChunkTypeTableSummary {
			summaryContent = chunk.Content
		} else if chunk.ChunkType == types.ChunkTypeTableColumn {
			columnContent = chunk.Content
		}
	}

	liveSchema := t.loadLiveSchema(ctx, knowledge)
	hasStored := summaryContent != "" && columnContent != ""
	if !hasStored && liveSchema == nil {
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("No table schema information found for knowledge ID '%s'", input.KnowledgeID),
		}, fmt.Errorf("no schema info found")
	}

	var output strings.Builder
	if hasStored {
		fmt.Fprintf(&output, "%s\n\n%s\n\n", summaryContent, columnContent)
	}
	if liveSchema != nil {
		// The live structure is authoritative for names and counts; the stored
		// summary above only adds business context.
		fmt.Fprintf(&output, "# Current Table Structure\n\n%s\n", liveSchema.Description())
	}
	fmt.Fprintf(&output, "Query this document with data_analysis as the table %q.", DataAnalysisTableName)

	data := map[string]interface{}{
		"summary": summaryContent,
		"columns": columnContent,
	}
	if liveSchema != nil {
		data["row_count"] = liveSchema.RowCount
		if len(liveSchema.Sheets) > 0 {
			data["sheets"] = liveSchema.Sheets
		}
	}

	return &types.ToolResult{
		Success: true,
		Output:  output.String(),
		Data:    data,
	}, nil
}

// loadLiveSchema returns the document's current schema, or nil when no loader
// is configured or loading fails; the stored summary is still returned then.
func (t *DataSchemaTool) loadLiveSchema(ctx context.Context, knowledge *types.Knowledge) *TableSchema {
	if t.liveSchema == nil {
		return nil
	}
	schema, err := t.liveSchema.LoadFromKnowledge(ctx, knowledge)
	if err != nil {
		logger.Warnf(ctx, "[Tool][DataSchema] Live schema unavailable for knowledge '%s', using stored summary only: %v",
			knowledge.ID, err)
		return nil
	}
	return schema
}
