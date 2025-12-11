package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/ossf/gemara/layer1"
	"github.com/ossf/gemara/layer1/pipeline/converter"
	"github.com/ossf/gemara/layer1/pipeline/llm"
	"github.com/ossf/gemara/layer1/pipeline/parser"
	"github.com/ossf/gemara/layer1/pipeline/segmenter"
	"github.com/ossf/gemara/layer1/pipeline/storage"
	"github.com/ossf/gemara/layer1/pipeline/types"
	"github.com/ossf/gemara/layer1/pipeline/validator"
)

var (
	// Common flags
	baseDir    = flag.String("base-dir", "./layer1/pipeline/test-data", "Base directory for storage")
	documentID = flag.String("document-id", "", "Document ID (required for most operations)")
	verbose    = flag.Bool("verbose", false, "Enable verbose output")
	
	// Parse flags
	inputFile    = flag.String("input", "", "Input PDF file path")
	parserType   = flag.String("parser", "simple", "Parser type (simple, docling, pymupdf)")
	_ = flag.String("parser-config", "", "Parser configuration file") // Reserved for future use
	
	// Segment flags
	segmenterType   = flag.String("segmenter", "generic", "Segmenter type (generic, pci-dss, nist-800-53)")
	_ = flag.String("segmenter-config", "", "Segmenter configuration file") // Reserved for future use
	sourceVersion   = flag.Int("source-version", 0, "Source version (0 = latest)")
	
	// Convert flags
	outputFile   = flag.String("output", "", "Output file path")
	outputFormat = flag.String("format", "yaml", "Output format (yaml, json)")
	
	// Enhance flags
	llmProvider = flag.String("llm-provider", "mock", "LLM provider (openai, anthropic, mock)")
	llmModel    = flag.String("llm-model", "", "LLM model name")
	llmAPIKey   = flag.String("llm-api-key", "", "LLM API key (or set env var)")
	temperature = flag.Float64("temperature", 0.3, "LLM temperature")
	maxTokens   = flag.Int("max-tokens", 2000, "LLM max tokens")

	// Validate flags
	strictValidation = flag.Bool("strict", true, "Enable strict validation mode")
	validateFile     = flag.String("validate-file", "", "Path to Layer-1 file to validate (optional)")
	saveReport       = flag.Bool("save-report", true, "Save validation reports for audit trail")
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	
	command := os.Args[1]
	_ = flag.CommandLine.Parse(os.Args[2:]) // Error intentionally ignored; invalid flags will be handled by flag package
	
	// Initialize storage
	store, err := storage.NewStorage(*baseDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	
	ctx := context.Background()
	
	switch command {
	case "parse":
		if err := cmdParse(ctx, store); err != nil {
			fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
			os.Exit(1)
		}
	case "segment":
		if err := cmdSegment(ctx, store); err != nil {
			fmt.Fprintf(os.Stderr, "Segment error: %v\n", err)
			os.Exit(1)
		}
	case "convert":
		if err := cmdConvert(ctx, store); err != nil {
			fmt.Fprintf(os.Stderr, "Convert error: %v\n", err)
			os.Exit(1)
		}
	case "enhance":
		if err := cmdEnhance(ctx, store); err != nil {
			fmt.Fprintf(os.Stderr, "Enhance error: %v\n", err)
			os.Exit(1)
		}
	case "run-all":
		if err := cmdRunAll(ctx, store); err != nil {
			fmt.Fprintf(os.Stderr, "Pipeline error: %v\n", err)
			os.Exit(1)
		}
	case "list":
		if err := cmdList(store); err != nil {
			fmt.Fprintf(os.Stderr, "List error: %v\n", err)
			os.Exit(1)
		}
	case "validate":
		if err := cmdValidate(ctx, store); err != nil {
			fmt.Fprintf(os.Stderr, "Validation error: %v\n", err)
			os.Exit(1)
		}
	case "coverage":
		if err := cmdCoverage(ctx, store); err != nil {
			fmt.Fprintf(os.Stderr, "Coverage analysis error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func cmdParse(ctx context.Context, store *storage.Storage) error {
	if *inputFile == "" {
		return fmt.Errorf("--input is required")
	}
	if *documentID == "" {
		// Generate document ID from filename
		*documentID = filepath.Base(*inputFile)
		*documentID = (*documentID)[:len(*documentID)-len(filepath.Ext(*documentID))]
	}
	
	log("Parsing %s with %s parser...\n", *inputFile, *parserType)
	
	// Configure parser
	config := types.ParserConfig{
		Provider:      *parserType,
		TempDir:       filepath.Join(*baseDir, "temp"),
		KeepTempFiles: *verbose,
	}
	
	// Create parser
	p, err := parser.NewParser(config)
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}
	
	// Parse PDF
	doc, err := p.Parse(*inputFile)
	if err != nil {
		return fmt.Errorf("parsing failed: %w", err)
	}
	
	doc.Metadata.DocumentID = *documentID
	
	// Save parsed document
	if err := store.SaveParsed(doc); err != nil {
		return fmt.Errorf("failed to save parsed document: %w", err)
	}
	
	log("Parsed document saved: %s v%d\n", *documentID, doc.Metadata.Version)
	log("  Pages: %d\n", len(doc.Pages))
	log("  Total blocks: %d\n", countBlocks(doc))
	
	return nil
}

func cmdSegment(ctx context.Context, store *storage.Storage) error {
	if *documentID == "" {
		return fmt.Errorf("--document-id is required")
	}
	
	log("Loading parsed document %s...\n", *documentID)
	
	// Load parsed document
	parsed, err := store.LoadParsed(*documentID, *sourceVersion)
	if err != nil {
		return fmt.Errorf("failed to load parsed document: %w", err)
	}
	
	log("Segmenting with %s segmenter...\n", *segmenterType)
	
	// Configure segmenter
	config := types.SegmenterConfig{
		DocumentType: *segmenterType,
	}
	
	// Create segmenter
	seg, err := segmenter.NewSegmenter(config)
	if err != nil {
		return fmt.Errorf("failed to create segmenter: %w", err)
	}
	
	// Segment document
	segmented, err := seg.Segment(parsed)
	if err != nil {
		return fmt.Errorf("segmentation failed: %w", err)
	}
	
	// Save segmented document
	if err := store.SaveSegmented(segmented); err != nil {
		return fmt.Errorf("failed to save segmented document: %w", err)
	}
	
	log("Segmented document saved: %s v%d\n", *documentID, segmented.Metadata.Version)
	log("  Categories: %d\n", len(segmented.Categories))
	log("  Guidelines: %d\n", countSegmentedGuidelines(segmented))
	
	return nil
}

func cmdConvert(ctx context.Context, store *storage.Storage) error {
	if *documentID == "" {
		return fmt.Errorf("--document-id is required")
	}
	
	log("Loading segmented document %s...\n", *documentID)
	
	// Load segmented document
	segmented, err := store.LoadSegmented(*documentID, *sourceVersion)
	if err != nil {
		return fmt.Errorf("failed to load segmented document: %w", err)
	}
	
	log("Converting to Layer-1 format...\n")
	
	// Create converter
	conv := converter.NewConverter()
	
	// Convert to Layer-1
	layer1Doc, err := conv.Convert(segmented)
	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}
	
	// Validate against Layer-1 schema
	log("Validating against Layer-1 schema...\n")
	v := validator.NewValidator(validator.WithStrictMode(*strictValidation))
	result := v.Validate(layer1Doc)
	
	// Create validation report for audit trail
	var report *storage.ValidationReport
	if *saveReport {
		report = &storage.ValidationReport{
			DocumentID:    *documentID,
			Timestamp:     time.Now(),
			StrictMode:    *strictValidation,
			Valid:         result.Valid,
			ErrorCount:    len(result.Errors),
			SourceVersion: segmented.Metadata.Version,
			Stage:         "convert",
		}
		// Convert errors for storage
		for _, e := range result.Errors {
			report.Errors = append(report.Errors, storage.ValidationError{
				Path:    e.Path,
				Message: e.Message,
				Value:   e.Value,
			})
		}
	}
	
	if !result.Valid {
		log("Validation errors found:\n")
		for _, e := range result.Errors {
			log("  - %s\n", e.Error())
		}
		// Save the validation report even on failure for reference
		if *saveReport && report != nil {
			if err := store.SaveValidationReport(report); err != nil {
				log("Warning: failed to save validation report: %v\n", err)
			} else {
				log("  Validation report saved for reference\n")
			}
		}
		return fmt.Errorf("schema validation failed with %d errors", len(result.Errors))
	}
	log("  Schema validation passed ✓\n")
	
	// Save final document with validation report
	if err := store.SaveFinalWithValidation(*documentID, layer1Doc, *outputFormat, report); err != nil {
		return fmt.Errorf("failed to save final document: %w", err)
	}
	if *saveReport {
		log("  Validation report saved\n")
	}
	
	// Also save to custom output path if specified
	if *outputFile != "" {
		if err := saveToFile(*outputFile, layer1Doc, *outputFormat); err != nil {
			return fmt.Errorf("failed to save to output file: %w", err)
		}
		log("Saved to: %s\n", *outputFile)
	}
	
	log("Conversion complete: %s\n", *documentID)
	log("  Categories: %d\n", len(layer1Doc.Categories))
	log("  Total guidelines: %d\n", countLayer1Guidelines(layer1Doc))
	
	return nil
}

func cmdEnhance(ctx context.Context, store *storage.Storage) error {
	if *documentID == "" {
		return fmt.Errorf("--document-id is required")
	}
	
	log("Loading segmented document %s...\n", *documentID)
	
	// Load segmented document
	segmented, err := store.LoadSegmented(*documentID, *sourceVersion)
	if err != nil {
		return fmt.Errorf("failed to load segmented document: %w", err)
	}
	
	preEnhanceVersion := segmented.Metadata.Version
	log("  Loaded version %d (will be preserved as pre-enhance reference)\n", preEnhanceVersion)
	
	log("Enhancing with %s...\n", *llmProvider)
	
	// Configure LLM
	apiKey := *llmAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
		if apiKey == "" && *llmProvider != "mock" {
			return fmt.Errorf("LLM API key required (--llm-api-key or LLM_API_KEY env var)")
		}
	}
	
	config := types.LLMConfig{
		Provider:    *llmProvider,
		Model:       *llmModel,
		APIKey:      apiKey,
		Temperature: *temperature,
		MaxTokens:   *maxTokens,
	}
	
	// Create enhancer
	enhancer, err := llm.NewEnhancer(config)
	if err != nil {
		return fmt.Errorf("failed to create enhancer: %w", err)
	}
	
	// Enhance segmentation
	result, err := enhancer.EnhanceSegmentation(ctx, segmented)
	if err != nil {
		return fmt.Errorf("enhancement failed: %w", err)
	}
	
	log("Enhancement complete:\n")
	log("  Provider: %s\n", result.Provider)
	log("  Confidence: %.2f\n", result.Confidence)
	log("  Changes: %d\n", len(result.Changes))
	
	if *verbose {
		for i, change := range result.Changes {
			log("  %d. %s: %s (%s)\n", i+1, change.Path, change.Type, change.Reason)
		}
	}
	
	// Extract enhanced document from result
	enhancedDoc, ok := result.EnhancedData.(*types.SegmentedDocument)
	if !ok {
		return fmt.Errorf("enhanced data is not a SegmentedDocument")
	}
	
	// Save enhanced segmented document with descriptive label
	log("Saving enhanced segmented document...\n")
	enhanceLabel := fmt.Sprintf("post-enhance-%s (pre-enhance: v%d)", *llmProvider, preEnhanceVersion)
	if err := store.SaveSegmentedWithLabel(enhancedDoc, enhanceLabel); err != nil {
		return fmt.Errorf("failed to save enhanced document: %w", err)
	}
	log("  Saved as version %d (label: %s)\n", enhancedDoc.Metadata.Version, enhanceLabel)
	log("  Pre-enhance reference: version %d\n", preEnhanceVersion)
	
	// CRITICAL: Validate the enhanced document by converting to Layer-1 and checking schema
	log("Validating enhanced document against Layer-1 schema...\n")
	conv := converter.NewConverter()
	layer1Doc, err := conv.Convert(enhancedDoc)
	if err != nil {
		return fmt.Errorf("conversion of enhanced document failed: %w", err)
	}
	
	// Perform schema validation
	v := validator.NewValidator(validator.WithStrictMode(*strictValidation))
	validationResult := v.Validate(layer1Doc)
	if !validationResult.Valid {
		log("⚠ Validation WARNINGS after enhancement:\n")
		for _, e := range validationResult.Errors {
			log("  - %s\n", e.Error())
		}
		if *strictValidation {
			return fmt.Errorf("enhanced document failed schema validation with %d errors", len(validationResult.Errors))
		}
		log("  Continuing despite warnings (use --strict to fail on validation errors)\n")
	} else {
		log("  Schema validation passed ✓\n")
	}
	
	return nil
}

func cmdRunAll(ctx context.Context, store *storage.Storage) error {
	// Run complete pipeline: parse -> segment -> convert
	if err := cmdParse(ctx, store); err != nil {
		return err
	}
	
	if err := cmdSegment(ctx, store); err != nil {
		return err
	}
	
	if err := cmdConvert(ctx, store); err != nil {
		return err
	}
	
	log("\nPipeline complete!\n")
	return nil
}

func cmdList(store *storage.Storage) error {
	if *documentID == "" {
		return fmt.Errorf("--document-id is required")
	}
	
	// List all versions
	parsed, err := store.ListVersions(*documentID, "parsed")
	if err != nil {
		return err
	}
	
	segmented, err := store.ListVersions(*documentID, "segmented")
	if err != nil {
		return err
	}
	
	fmt.Printf("Document: %s\n\n", *documentID)
	
	fmt.Println("Parsed versions:")
	for _, v := range parsed {
		fmt.Printf("  v%d - %s (%d bytes)\n", v.Version, v.StoredAt.Format(time.RFC3339), v.Size)
	}
	
	fmt.Println("\nSegmented versions:")
	for _, v := range segmented {
		fmt.Printf("  v%d - %s (%d bytes)\n", v.Version, v.StoredAt.Format(time.RFC3339), v.Size)
	}
	
	return nil
}

func cmdValidate(ctx context.Context, store *storage.Storage) error {
	var layer1Doc *layer1.GuidanceDocument
	var err error
	
	// Load from file or from storage
	if *validateFile != "" {
		log("Loading Layer-1 document from file: %s\n", *validateFile)
		layer1Doc, err = loadLayer1FromFile(*validateFile)
		if err != nil {
			return fmt.Errorf("failed to load file: %w", err)
		}
	} else if *documentID != "" {
		log("Loading Layer-1 document from storage: %s\n", *documentID)
		layer1Doc, err = store.LoadFinal(*documentID)
		if err != nil {
			return fmt.Errorf("failed to load from storage: %w", err)
		}
	} else {
		return fmt.Errorf("either --document-id or --validate-file is required")
	}
	
	// Perform validation
	log("Validating against Layer-1 schema (strict=%v)...\n", *strictValidation)
	v := validator.NewValidator(validator.WithStrictMode(*strictValidation))
	result := v.Validate(layer1Doc)
	
	if result.Valid {
		log("\n✓ Validation PASSED\n")
		log("  Document ID: %s\n", layer1Doc.Metadata.Id)
		log("  Title: %s\n", layer1Doc.Metadata.Title)
		log("  Document Type: %s\n", layer1Doc.Metadata.DocumentType)
		log("  Categories: %d\n", len(layer1Doc.Categories))
		log("  Total guidelines: %d\n", countLayer1Guidelines(layer1Doc))
		return nil
	}
	
	// Report validation errors
	log("\n✗ Validation FAILED with %d errors:\n\n", len(result.Errors))
	for i, e := range result.Errors {
		log("  %d. [%s] %s", i+1, e.Path, e.Message)
		if e.Value != nil {
			log(" (got: %v)", e.Value)
		}
		log("\n")
	}
	
	return fmt.Errorf("schema validation failed")
}

func cmdCoverage(ctx context.Context, store *storage.Storage) error {
	var layer1Doc *layer1.GuidanceDocument
	var segmented *types.SegmentedDocument
	var parsed *types.ParsedDocument
	var err error
	
	// Load documents for coverage analysis
	if *validateFile != "" {
		log("Loading Layer-1 document from file: %s\n", *validateFile)
		layer1Doc, err = loadLayer1FromFile(*validateFile)
		if err != nil {
			return fmt.Errorf("failed to load file: %w", err)
		}
	} else if *documentID != "" {
		log("Loading documents for coverage analysis: %s\n", *documentID)
		
		// Try to load all available documents for comprehensive analysis
		layer1Doc, err = store.LoadFinal(*documentID)
		if err != nil {
			log("  Note: Final Layer-1 document not found, will analyze available data\n")
		}
		
		segmented, err = store.LoadSegmented(*documentID, *sourceVersion)
		if err != nil {
			log("  Note: Segmented document not found\n")
		}
		
		parsed, err = store.LoadParsed(*documentID, *sourceVersion)
		if err != nil {
			log("  Note: Parsed document not found\n")
		}
	} else {
		return fmt.Errorf("either --document-id or --validate-file is required")
	}
	
	// Perform coverage analysis
	analyzer := validator.NewCoverageAnalyzer(*strictValidation)
	
	var report *validator.CoverageReport
	
	if segmented != nil && parsed != nil {
		log("Analyzing schema coverage from parsed and segmented documents...\n")
		report = analyzer.AnalyzeFromSegmented(parsed, segmented)
	} else if layer1Doc != nil {
		log("Analyzing schema coverage from Layer-1 document...\n")
		report = analyzer.AnalyzeLayer1(layer1Doc)
	} else {
		return fmt.Errorf("no documents available for coverage analysis")
	}
	
	// Display coverage report
	printCoverageReport(report)
	
	// Save report if requested
	if *saveReport {
		reportPath := filepath.Join(store.GetBaseDir(), "coverage-reports")
		if err := os.MkdirAll(reportPath, 0755); err == nil {
			filename := fmt.Sprintf("%s-%s.json", report.DocumentID, report.Timestamp.Format("20060102-150405"))
			filePath := filepath.Join(reportPath, filename)
			if data, err := json.MarshalIndent(report, "", "  "); err == nil {
				if err := os.WriteFile(filePath, data, 0644); err == nil {
					log("\nCoverage report saved to: %s\n", filePath)
				}
			}
		}
	}
	
	return nil
}

func printCoverageReport(report *validator.CoverageReport) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("SCHEMA COVERAGE REPORT: %s\n", report.DocumentID)
	fmt.Println(strings.Repeat("=", 60))
	
	// Source stats
	if report.SourceStats.TotalBlocks > 0 {
		fmt.Println("\n📄 SOURCE DOCUMENT:")
		fmt.Printf("  Pages: %d\n", report.SourceStats.TotalPages)
		fmt.Printf("  Blocks: %d\n", report.SourceStats.TotalBlocks)
		fmt.Printf("  Characters: %d\n", report.SourceStats.TotalCharacters)
		if len(report.SourceStats.BlocksByType) > 0 {
			fmt.Println("  Block types:")
			for typ, count := range report.SourceStats.BlocksByType {
				fmt.Printf("    - %s: %d\n", typ, count)
			}
		}
	}
	
	// Captured content
	fmt.Println("\n✅ CAPTURED CONTENT:")
	fmt.Printf("  Categories: %d\n", report.CapturedContent.Categories)
	fmt.Printf("  Guidelines: %d\n", report.CapturedContent.Guidelines)
	fmt.Printf("  Parts: %d\n", report.CapturedContent.Parts)
	fmt.Printf("  Recommendations: %d\n", report.CapturedContent.Recommendations)
	
	if len(report.CapturedContent.FieldsCaptured) > 0 {
		fmt.Println("  Fields populated:")
		for _, field := range report.CapturedContent.FieldsCaptured {
			fmt.Printf("    ✓ %s\n", field)
		}
	}
	
	if len(report.CapturedContent.FieldsEmpty) > 0 {
		fmt.Println("  Fields empty/missing:")
		for _, field := range report.CapturedContent.FieldsEmpty {
			fmt.Printf("    ✗ %s\n", field)
		}
	}
	
	// Coverage metrics
	fmt.Println("\n📊 COVERAGE METRICS:")
	fmt.Printf("  Overall Score: %.1f/100\n", report.CoverageMetrics.OverallScore)
	if report.CoverageMetrics.BlockCoverage > 0 {
		fmt.Printf("  Block Coverage: %.1f%%\n", report.CoverageMetrics.BlockCoverage)
	}
	fmt.Printf("  Required Fields: %d/%d\n", report.CoverageMetrics.RequiredFieldsCovered, report.CoverageMetrics.RequiredFieldsTotal)
	fmt.Printf("  Optional Fields: %d/%d\n", report.CoverageMetrics.OptionalFieldsCovered, report.CoverageMetrics.OptionalFieldsTotal)
	
	if len(report.CoverageMetrics.QualityIndicators) > 0 {
		fmt.Println("  Quality indicators:")
		for indicator, value := range report.CoverageMetrics.QualityIndicators {
			fmt.Printf("    %s: %s\n", indicator, value)
		}
	}
	
	// Unmapped content
	if len(report.UnmappedContent) > 0 {
		fmt.Println("\n⚠️  UNMAPPED CONTENT (Schema Gaps):")
		fmt.Printf("  Total unmapped items: %d\n", len(report.UnmappedContent))
		
		// Show first few examples
		maxShow := 5
		for i, unmapped := range report.UnmappedContent {
			if i >= maxShow {
				fmt.Printf("  ... and %d more\n", len(report.UnmappedContent)-maxShow)
				break
			}
			fmt.Printf("\n  [%d] Type: %s\n", i+1, unmapped.ContentType)
			fmt.Printf("      Location: %s\n", unmapped.SourceLocation)
			fmt.Printf("      Reason: %s\n", unmapped.Reason)
			if unmapped.SuggestedField != "" {
				fmt.Printf("      Suggested schema field: %s\n", unmapped.SuggestedField)
			}
			// Truncate content for display
			content := unmapped.Content
			if len(content) > 100 {
				content = content[:97] + "..."
			}
			fmt.Printf("      Content: %s\n", content)
		}
	}
	
	// Schema gaps
	if len(report.SchemaGaps) > 0 {
		fmt.Println("\n🔍 SCHEMA GAPS IDENTIFIED:")
		for _, gap := range report.SchemaGaps {
			fmt.Printf("\n  [%s priority] %s\n", gap.Priority, gap.SuggestedField)
			fmt.Printf("    %s\n", gap.Description)
			fmt.Printf("    Occurrences: %d\n", gap.OccurrenceCount)
			if len(gap.Examples) > 0 {
				fmt.Println("    Examples:")
				for _, ex := range gap.Examples {
					fmt.Printf("      - %s\n", ex)
				}
			}
		}
	}
	
	// Recommendations
	if len(report.Recommendations) > 0 {
		fmt.Println("\n💡 RECOMMENDATIONS:")
		for i, rec := range report.Recommendations {
			fmt.Printf("\n  %d. [%s] %s\n", i+1, rec.Priority, rec.Description)
			fmt.Printf("     Target: %s\n", rec.Target)
			fmt.Printf("     Rationale: %s\n", rec.Rationale)
		}
	}
	
	fmt.Println("\n" + strings.Repeat("=", 60))
}

// loadLayer1FromFile loads a Layer-1 document from a YAML or JSON file
func loadLayer1FromFile(path string) (*layer1.GuidanceDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var doc layer1.GuidanceDocument
	
	// Try YAML first (it's a superset of JSON)
	if err := yaml.Unmarshal(data, &doc); err != nil {
		// Try JSON
		if jsonErr := json.Unmarshal(data, &doc); jsonErr != nil {
			return nil, fmt.Errorf("failed to parse as YAML (%v) or JSON (%v)", err, jsonErr)
		}
	}
	
	return &doc, nil
}

func saveToFile(path string, data interface{}, format string) error {
	var bytes []byte
	var err error
	
	switch format {
	case "yaml", "yml":
		bytes, err = yaml.Marshal(data)
	case "json":
		bytes, err = json.MarshalIndent(data, "", "  ")
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, bytes, 0644)
}

func countBlocks(doc *types.ParsedDocument) int {
	count := 0
	for _, page := range doc.Pages {
		count += len(page.Blocks)
	}
	return count
}

func countSegmentedGuidelines(doc *types.SegmentedDocument) int {
	count := 0
	for _, cat := range doc.Categories {
		count += len(cat.Guidelines)
	}
	return count
}

func countLayer1Guidelines(doc *layer1.GuidanceDocument) int {
	count := 0
	for _, cat := range doc.Categories {
		count += len(cat.Guidelines)
	}
	return count
}

func log(format string, args ...interface{}) {
	if *verbose || true { // Always show for now
		fmt.Printf(format, args...)
	}
}

func printUsage() {
	fmt.Print(`
Gemara Layer-1 PDF Conversion Pipeline

Usage: pipeline <command> [options]

Commands:
  parse       Parse PDF into structured blocks
  segment     Segment parsed data into categories/guidelines
  convert     Convert segmented data to Layer-1 format (includes validation)
  enhance     Enhance with LLM (can be re-run on existing data)
  validate    Validate Layer-1 document against schema
  coverage    Analyze schema coverage (what info couldn't be captured)
  run-all     Run complete pipeline (parse -> segment -> convert)
  list        List all versions of a document

Parse Options:
  --input <file>           Input PDF file (required)
  --document-id <id>       Document ID (default: filename)
  --parser <type>          Parser type (simple, docling) [default: simple]

Segment Options:
  --document-id <id>       Document ID (required)
  --segmenter <type>       Segmenter type (generic, pci-dss, nist-800-53) [default: generic]
  --source-version <n>     Source version (0 = latest) [default: 0]

Convert Options:
  --document-id <id>       Document ID (required)
  --output <file>          Output file path (optional)
  --format <fmt>           Output format (yaml, json) [default: yaml]
  --strict                 Enable strict validation [default: true]

Enhance Options:
  --document-id <id>       Document ID (required)
  --llm-provider <name>    LLM provider (openai, anthropic, mock) [default: mock]
  --llm-model <model>      LLM model name
  --llm-api-key <key>      LLM API key (or set LLM_API_KEY env var)
  --temperature <t>        Temperature [default: 0.3]
  --max-tokens <n>         Max tokens [default: 2000]

Validate Options:
  --document-id <id>       Document ID to validate from storage
  --validate-file <path>   Path to external Layer-1 file to validate
  --strict                 Enable strict validation [default: true]
  --save-report            Save validation report for audit [default: true]

Coverage Options:
  --document-id <id>       Document ID to analyze from storage
  --validate-file <path>   Path to external Layer-1 file to analyze
  --save-report            Save coverage report [default: true]

Global Options:
  --base-dir <dir>         Base directory for storage [default: ./layer1/pipeline/test-data]
  --verbose                Enable verbose output

Examples:
  # Complete pipeline
  pipeline run-all --input PCI_DSS_v3-2-1.pdf --document-id pci-dss-3.2.1 --segmenter pci-dss

  # Step by step
  pipeline parse --input PCI_DSS_v3-2-1.pdf --document-id pci-dss-3.2.1
  pipeline segment --document-id pci-dss-3.2.1 --segmenter pci-dss
  pipeline convert --document-id pci-dss-3.2.1 --output pci-dss.yaml
  
  # Enhance with LLM (re-runnable)
  pipeline enhance --document-id pci-dss-3.2.1 --llm-provider openai
  
  # Validate final output
  pipeline validate --document-id pci-dss-3.2.1
  pipeline validate --validate-file ./my-document.yaml --strict
  
  # Analyze schema coverage (what info couldn't be captured)
  pipeline coverage --document-id pci-dss-3.2.1
  pipeline coverage --validate-file ./my-document.yaml
  
  # List versions
  pipeline list --document-id pci-dss-3.2.1
`)
}

