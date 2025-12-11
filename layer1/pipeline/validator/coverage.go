package validator

import (
	"fmt"
	"time"

	"github.com/ossf/gemara/layer1"
	"github.com/ossf/gemara/layer1/pipeline/types"
)

// CoverageReport provides a complete analysis of schema coverage
type CoverageReport struct {
	DocumentID        string              `json:"document_id" yaml:"document_id"`
	Timestamp         time.Time           `json:"timestamp" yaml:"timestamp"`
	
	// Source document stats
	SourceStats       SourceStats         `json:"source_stats" yaml:"source_stats"`
	
	// What was captured
	CapturedContent   CapturedContent     `json:"captured_content" yaml:"captured_content"`
	
	// What couldn't be captured
	UnmappedContent   []types.UnmappedContent `json:"unmapped_content,omitempty" yaml:"unmapped_content,omitempty"`
	
	// Coverage metrics
	CoverageMetrics   CoverageMetrics     `json:"coverage_metrics" yaml:"coverage_metrics"`
	
	// Schema gaps identified
	SchemaGaps        []types.SchemaGap   `json:"schema_gaps,omitempty" yaml:"schema_gaps,omitempty"`
	
	// Recommendations for schema improvements
	Recommendations   []SchemaRecommendation `json:"recommendations,omitempty" yaml:"recommendations,omitempty"`
}

// SourceStats tracks statistics from the source document
type SourceStats struct {
	TotalPages        int            `json:"total_pages" yaml:"total_pages"`
	TotalBlocks       int            `json:"total_blocks" yaml:"total_blocks"`
	BlocksByType      map[string]int `json:"blocks_by_type" yaml:"blocks_by_type"`
	TotalCharacters   int            `json:"total_characters" yaml:"total_characters"`
}

// CapturedContent summarizes what was successfully captured
type CapturedContent struct {
	Categories        int `json:"categories" yaml:"categories"`
	Guidelines        int `json:"guidelines" yaml:"guidelines"`
	Parts             int `json:"parts" yaml:"parts"`
	Recommendations   int `json:"recommendations" yaml:"recommendations"`
	
	// Field coverage
	FieldsCaptured    []string `json:"fields_captured" yaml:"fields_captured"`
	FieldsEmpty       []string `json:"fields_empty" yaml:"fields_empty"`
}

// CoverageMetrics provides numerical coverage analysis
type CoverageMetrics struct {
	// Block-level coverage
	BlockCoverage     float64 `json:"block_coverage" yaml:"block_coverage"`         // % of blocks mapped
	CharacterCoverage float64 `json:"character_coverage" yaml:"character_coverage"` // % of text captured
	
	// Field-level coverage  
	RequiredFieldsCovered int     `json:"required_fields_covered" yaml:"required_fields_covered"`
	RequiredFieldsTotal   int     `json:"required_fields_total" yaml:"required_fields_total"`
	OptionalFieldsCovered int     `json:"optional_fields_covered" yaml:"optional_fields_covered"`
	OptionalFieldsTotal   int     `json:"optional_fields_total" yaml:"optional_fields_total"`
	
	// Overall score
	OverallScore      float64 `json:"overall_score" yaml:"overall_score"` // 0-100
	
	// Quality indicators
	QualityIndicators map[string]string `json:"quality_indicators" yaml:"quality_indicators"`
}

// SchemaRecommendation suggests schema improvements
type SchemaRecommendation struct {
	Type        string `json:"type" yaml:"type"`               // "add_field", "modify_field", "add_type"
	Target      string `json:"target" yaml:"target"`           // Where to add/modify
	Description string `json:"description" yaml:"description"`
	Priority    string `json:"priority" yaml:"priority"`       // "high", "medium", "low"
	Rationale   string `json:"rationale" yaml:"rationale"`     // Why this is recommended
	Examples    []string `json:"examples,omitempty" yaml:"examples,omitempty"`
}

// CoverageAnalyzer analyzes schema coverage
type CoverageAnalyzer struct {
	strictMode bool
}

// NewCoverageAnalyzer creates a new coverage analyzer
func NewCoverageAnalyzer(strict bool) *CoverageAnalyzer {
	return &CoverageAnalyzer{strictMode: strict}
}

// AnalyzeFromSegmented analyzes coverage from a segmented document
func (a *CoverageAnalyzer) AnalyzeFromSegmented(parsed *types.ParsedDocument, segmented *types.SegmentedDocument) *CoverageReport {
	report := &CoverageReport{
		DocumentID: segmented.Metadata.DocumentID,
		Timestamp:  time.Now(),
	}
	
	// Calculate source stats
	report.SourceStats = a.calculateSourceStats(parsed)
	
	// Calculate captured content
	report.CapturedContent = a.calculateCapturedContent(segmented)
	
	// Copy unmapped content from segmented document
	report.UnmappedContent = segmented.UnmappedContent
	
	// Calculate coverage metrics
	report.CoverageMetrics = a.calculateCoverageMetrics(parsed, segmented)
	
	// Identify schema gaps
	report.SchemaGaps = a.identifySchemaGaps(segmented)
	
	// Generate recommendations
	report.Recommendations = a.generateRecommendations(report)
	
	return report
}

// AnalyzeLayer1 analyzes coverage of a Layer-1 document against expected schema
func (a *CoverageAnalyzer) AnalyzeLayer1(doc *layer1.GuidanceDocument) *CoverageReport {
	report := &CoverageReport{
		DocumentID: doc.Metadata.Id,
		Timestamp:  time.Now(),
	}
	
	// Analyze what's captured in Layer-1
	report.CapturedContent = a.analyzeLayer1Captured(doc)
	
	// Calculate field coverage
	report.CoverageMetrics = a.calculateLayer1Coverage(doc)
	
	// Generate recommendations based on empty/missing fields
	report.Recommendations = a.generateLayer1Recommendations(doc)
	
	return report
}

// calculateSourceStats calculates stats from parsed document
func (a *CoverageAnalyzer) calculateSourceStats(parsed *types.ParsedDocument) SourceStats {
	stats := SourceStats{
		BlocksByType: make(map[string]int),
	}
	
	if parsed == nil {
		return stats
	}
	
	stats.TotalPages = len(parsed.Pages)
	
	for _, page := range parsed.Pages {
		stats.TotalBlocks += len(page.Blocks)
		for _, block := range page.Blocks {
			stats.BlocksByType[string(block.Type)]++
			stats.TotalCharacters += len(block.Text)
		}
	}
	
	return stats
}

// calculateCapturedContent calculates what was captured in segmentation
func (a *CoverageAnalyzer) calculateCapturedContent(segmented *types.SegmentedDocument) CapturedContent {
	captured := CapturedContent{
		FieldsCaptured: []string{},
		FieldsEmpty:    []string{},
	}
	
	if segmented == nil {
		return captured
	}
	
	captured.Categories = len(segmented.Categories)
	
	for _, cat := range segmented.Categories {
		captured.Guidelines += len(cat.Guidelines)
		for _, guide := range cat.Guidelines {
			captured.Parts += len(guide.Parts)
			captured.Recommendations += len(guide.Recommendations)
		}
	}
	
	// Check document metadata fields
	meta := segmented.DocumentMetadata
	if meta.ID != "" {
		captured.FieldsCaptured = append(captured.FieldsCaptured, "metadata.id")
	} else {
		captured.FieldsEmpty = append(captured.FieldsEmpty, "metadata.id")
	}
	if meta.Title != "" {
		captured.FieldsCaptured = append(captured.FieldsCaptured, "metadata.title")
	} else {
		captured.FieldsEmpty = append(captured.FieldsEmpty, "metadata.title")
	}
	if meta.Description != "" {
		captured.FieldsCaptured = append(captured.FieldsCaptured, "metadata.description")
	} else {
		captured.FieldsEmpty = append(captured.FieldsEmpty, "metadata.description")
	}
	if meta.Author != "" {
		captured.FieldsCaptured = append(captured.FieldsCaptured, "metadata.author")
	} else {
		captured.FieldsEmpty = append(captured.FieldsEmpty, "metadata.author")
	}
	if meta.DocumentType != "" {
		captured.FieldsCaptured = append(captured.FieldsCaptured, "metadata.document_type")
	} else {
		captured.FieldsEmpty = append(captured.FieldsEmpty, "metadata.document_type")
	}
	
	return captured
}

// calculateCoverageMetrics calculates coverage percentages
func (a *CoverageAnalyzer) calculateCoverageMetrics(parsed *types.ParsedDocument, segmented *types.SegmentedDocument) CoverageMetrics {
	metrics := CoverageMetrics{
		QualityIndicators: make(map[string]string),
	}
	
	if parsed == nil || segmented == nil {
		return metrics
	}
	
	// Block coverage
	totalBlocks := 0
	for _, page := range parsed.Pages {
		totalBlocks += len(page.Blocks)
	}
	
	unmappedBlocks := len(segmented.UnmappedContent)
	mappedBlocks := totalBlocks - unmappedBlocks
	
	if totalBlocks > 0 {
		metrics.BlockCoverage = float64(mappedBlocks) / float64(totalBlocks) * 100
	}
	
	// Required fields (based on Layer-1 schema)
	metrics.RequiredFieldsTotal = 4 // id, title, description, author
	if segmented.DocumentMetadata.ID != "" {
		metrics.RequiredFieldsCovered++
	}
	if segmented.DocumentMetadata.Title != "" {
		metrics.RequiredFieldsCovered++
	}
	if segmented.DocumentMetadata.Description != "" {
		metrics.RequiredFieldsCovered++
	}
	if segmented.DocumentMetadata.Author != "" {
		metrics.RequiredFieldsCovered++
	}
	
	// Optional fields
	metrics.OptionalFieldsTotal = 4 // version, publication_date, document_type, jurisdictions
	if segmented.DocumentMetadata.Version != "" {
		metrics.OptionalFieldsCovered++
	}
	if segmented.DocumentMetadata.PublicationDate != "" {
		metrics.OptionalFieldsCovered++
	}
	if segmented.DocumentMetadata.DocumentType != "" {
		metrics.OptionalFieldsCovered++
	}
	if len(segmented.DocumentMetadata.Jurisdictions) > 0 {
		metrics.OptionalFieldsCovered++
	}
	
	// Calculate overall score
	requiredScore := 0.0
	if metrics.RequiredFieldsTotal > 0 {
		requiredScore = float64(metrics.RequiredFieldsCovered) / float64(metrics.RequiredFieldsTotal) * 60
	}
	optionalScore := 0.0
	if metrics.OptionalFieldsTotal > 0 {
		optionalScore = float64(metrics.OptionalFieldsCovered) / float64(metrics.OptionalFieldsTotal) * 20
	}
	blockScore := metrics.BlockCoverage * 0.2
	
	metrics.OverallScore = requiredScore + optionalScore + blockScore
	
	// Quality indicators
	if metrics.OverallScore >= 90 {
		metrics.QualityIndicators["overall"] = "excellent"
	} else if metrics.OverallScore >= 70 {
		metrics.QualityIndicators["overall"] = "good"
	} else if metrics.OverallScore >= 50 {
		metrics.QualityIndicators["overall"] = "fair"
	} else {
		metrics.QualityIndicators["overall"] = "needs_improvement"
	}
	
	if metrics.BlockCoverage >= 90 {
		metrics.QualityIndicators["content_capture"] = "comprehensive"
	} else if metrics.BlockCoverage >= 70 {
		metrics.QualityIndicators["content_capture"] = "substantial"
	} else {
		metrics.QualityIndicators["content_capture"] = "partial"
	}
	
	return metrics
}

// identifySchemaGaps identifies gaps in the schema based on unmapped content
func (a *CoverageAnalyzer) identifySchemaGaps(segmented *types.SegmentedDocument) []types.SchemaGap {
	if segmented == nil {
		return nil
	}
	
	// Group unmapped content by suggested field
	gapMap := make(map[string]*types.SchemaGap)
	
	for _, unmapped := range segmented.UnmappedContent {
		field := unmapped.SuggestedField
		if field == "" {
			field = unmapped.ContentType
		}
		
		if gap, exists := gapMap[field]; exists {
			gap.OccurrenceCount++
			if len(gap.Examples) < 3 {
				gap.Examples = append(gap.Examples, truncate(unmapped.Content, 100))
			}
		} else {
			gapMap[field] = &types.SchemaGap{
				SuggestedField:  field,
				Description:     fmt.Sprintf("Content of type '%s' cannot be captured by current schema", unmapped.ContentType),
				OccurrenceCount: 1,
				Examples:        []string{truncate(unmapped.Content, 100)},
				Priority:        "medium",
			}
		}
	}
	
	// Convert map to slice
	var gaps []types.SchemaGap
	for _, gap := range gapMap {
		// Set priority based on occurrence count
		if gap.OccurrenceCount >= 10 {
			gap.Priority = "high"
		} else if gap.OccurrenceCount >= 3 {
			gap.Priority = "medium"
		} else {
			gap.Priority = "low"
		}
		gaps = append(gaps, *gap)
	}
	
	return gaps
}

// generateRecommendations creates schema improvement recommendations
func (a *CoverageAnalyzer) generateRecommendations(report *CoverageReport) []SchemaRecommendation {
	var recs []SchemaRecommendation
	
	// Recommend based on schema gaps
	for _, gap := range report.SchemaGaps {
		rec := SchemaRecommendation{
			Type:        "add_field",
			Target:      gap.SuggestedField,
			Description: fmt.Sprintf("Add support for '%s' content type", gap.SuggestedField),
			Priority:    gap.Priority,
			Rationale:   fmt.Sprintf("Found %d instances of unmapped content", gap.OccurrenceCount),
			Examples:    gap.Examples,
		}
		recs = append(recs, rec)
	}
	
	// Recommend based on empty fields
	for _, field := range report.CapturedContent.FieldsEmpty {
		if field == "metadata.document_type" {
			recs = append(recs, SchemaRecommendation{
				Type:        "extraction_improvement",
				Target:      field,
				Description: "Improve document type extraction",
				Priority:    "high",
				Rationale:   "Document type is required for proper classification",
			})
		}
	}
	
	// Recommend if coverage is low
	if report.CoverageMetrics.BlockCoverage < 70 {
		recs = append(recs, SchemaRecommendation{
			Type:        "schema_extension",
			Target:      "categories",
			Description: "Consider adding more category types or flexible content containers",
			Priority:    "high",
			Rationale:   fmt.Sprintf("Only %.1f%% of content blocks were mapped to schema", report.CoverageMetrics.BlockCoverage),
		})
	}
	
	return recs
}

// analyzeLayer1Captured analyzes captured content in Layer-1 document
func (a *CoverageAnalyzer) analyzeLayer1Captured(doc *layer1.GuidanceDocument) CapturedContent {
	captured := CapturedContent{
		FieldsCaptured: []string{},
		FieldsEmpty:    []string{},
	}
	
	if doc == nil {
		return captured
	}
	
	captured.Categories = len(doc.Categories)
	
	for _, cat := range doc.Categories {
		captured.Guidelines += len(cat.Guidelines)
		for _, guide := range cat.Guidelines {
			captured.Parts += len(guide.GuidelineParts)
			captured.Recommendations += len(guide.Recommendations)
		}
	}
	
	// Check required fields
	checkField := func(name, value string) {
		if value != "" {
			captured.FieldsCaptured = append(captured.FieldsCaptured, name)
		} else {
			captured.FieldsEmpty = append(captured.FieldsEmpty, name)
		}
	}
	
	checkField("metadata.id", doc.Metadata.Id)
	checkField("metadata.title", doc.Metadata.Title)
	checkField("metadata.description", doc.Metadata.Description)
	checkField("metadata.author", doc.Metadata.Author)
	checkField("metadata.document_type", string(doc.Metadata.DocumentType))
	checkField("metadata.version", doc.Metadata.Version)
	checkField("metadata.publication_date", doc.Metadata.PublicationDate)
	checkField("front_matter", doc.FrontMatter)
	
	return captured
}

// calculateLayer1Coverage calculates coverage metrics for Layer-1 document
func (a *CoverageAnalyzer) calculateLayer1Coverage(doc *layer1.GuidanceDocument) CoverageMetrics {
	metrics := CoverageMetrics{
		QualityIndicators: make(map[string]string),
	}
	
	if doc == nil {
		return metrics
	}
	
	// Required fields per CUE schema
	metrics.RequiredFieldsTotal = 4
	if doc.Metadata.Id != "" {
		metrics.RequiredFieldsCovered++
	}
	if doc.Metadata.Title != "" {
		metrics.RequiredFieldsCovered++
	}
	if doc.Metadata.Description != "" {
		metrics.RequiredFieldsCovered++
	}
	if doc.Metadata.Author != "" {
		metrics.RequiredFieldsCovered++
	}
	
	// Optional fields
	metrics.OptionalFieldsTotal = 5
	if doc.Metadata.DocumentType != "" {
		metrics.OptionalFieldsCovered++
	}
	if doc.Metadata.Version != "" {
		metrics.OptionalFieldsCovered++
	}
	if doc.Metadata.PublicationDate != "" {
		metrics.OptionalFieldsCovered++
	}
	if doc.FrontMatter != "" {
		metrics.OptionalFieldsCovered++
	}
	if doc.Metadata.Applicability != nil {
		metrics.OptionalFieldsCovered++
	}
	
	// Calculate overall score
	requiredScore := float64(metrics.RequiredFieldsCovered) / float64(metrics.RequiredFieldsTotal) * 70
	optionalScore := float64(metrics.OptionalFieldsCovered) / float64(metrics.OptionalFieldsTotal) * 30
	metrics.OverallScore = requiredScore + optionalScore
	
	// Quality indicators
	if metrics.OverallScore >= 90 {
		metrics.QualityIndicators["completeness"] = "excellent"
	} else if metrics.OverallScore >= 70 {
		metrics.QualityIndicators["completeness"] = "good"
	} else if metrics.OverallScore >= 50 {
		metrics.QualityIndicators["completeness"] = "fair"
	} else {
		metrics.QualityIndicators["completeness"] = "incomplete"
	}
	
	// Check content depth
	if len(doc.Categories) == 0 {
		metrics.QualityIndicators["content_depth"] = "empty"
	} else {
		hasGuidelines := false
		hasParts := false
		for _, cat := range doc.Categories {
			if len(cat.Guidelines) > 0 {
				hasGuidelines = true
				for _, guide := range cat.Guidelines {
					if len(guide.GuidelineParts) > 0 {
						hasParts = true
						break
					}
				}
			}
		}
		if hasParts {
			metrics.QualityIndicators["content_depth"] = "detailed"
		} else if hasGuidelines {
			metrics.QualityIndicators["content_depth"] = "moderate"
		} else {
			metrics.QualityIndicators["content_depth"] = "shallow"
		}
	}
	
	return metrics
}

// generateLayer1Recommendations creates recommendations for Layer-1 document
func (a *CoverageAnalyzer) generateLayer1Recommendations(doc *layer1.GuidanceDocument) []SchemaRecommendation {
	var recs []SchemaRecommendation
	
	if doc == nil {
		return recs
	}
	
	// Check for missing required fields
	if doc.Metadata.Id == "" {
		recs = append(recs, SchemaRecommendation{
			Type:        "missing_required",
			Target:      "metadata.id",
			Description: "Document ID is missing",
			Priority:    "high",
			Rationale:   "ID is required for document identification and referencing",
		})
	}
	
	if doc.Metadata.DocumentType == "" {
		recs = append(recs, SchemaRecommendation{
			Type:        "missing_field",
			Target:      "metadata.document_type",
			Description: "Document type should be specified",
			Priority:    "medium",
			Rationale:   "Document type helps with classification (Standard, Regulation, Best Practice, Framework)",
		})
	}
	
	// Check for empty categories
	if len(doc.Categories) == 0 {
		recs = append(recs, SchemaRecommendation{
			Type:        "content_issue",
			Target:      "categories",
			Description: "No categories found in document",
			Priority:    "high",
			Rationale:   "A valid Layer-1 document should have at least one category",
		})
	}
	
	// Check for guidelines without objectives
	for i, cat := range doc.Categories {
		for j, guide := range cat.Guidelines {
			if guide.Objective == "" {
				recs = append(recs, SchemaRecommendation{
					Type:        "content_enhancement",
					Target:      fmt.Sprintf("categories[%d].guidelines[%d].objective", i, j),
					Description: fmt.Sprintf("Guideline '%s' is missing an objective", guide.Id),
					Priority:    "low",
					Rationale:   "Objectives help clarify the purpose of each guideline",
				})
			}
		}
	}
	
	return recs
}

// truncate shortens a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// boolToInt converts bool to int (Go doesn't have implicit bool->int conversion)
//
//nolint:unused // Utility function for metrics calculation
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

