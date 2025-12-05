package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/ossf/gemara/layer1"
	"github.com/ossf/gemara/layer1/pipeline/types"
	"github.com/ossf/gemara/layer1/pipeline/converter"
	"github.com/ossf/gemara/layer1/pipeline/llm"
	"github.com/ossf/gemara/layer1/pipeline/parser"
	"github.com/ossf/gemara/layer1/pipeline/segmenter"
	"github.com/ossf/gemara/layer1/pipeline/storage"
)

var (
	// Common flags
	baseDir    = flag.String("base-dir", "./layer1/pipeline/test-data", "Base directory for storage")
	documentID = flag.String("document-id", "", "Document ID (required for most operations)")
	verbose    = flag.Bool("verbose", false, "Enable verbose output")
	
	// Parse flags
	inputFile    = flag.String("input", "", "Input PDF file path")
	parserType   = flag.String("parser", "simple", "Parser type (simple, docling, pymupdf)")
	parserConfig = flag.String("parser-config", "", "Parser configuration file")
	
	// Segment flags
	segmenterType   = flag.String("segmenter", "generic", "Segmenter type (generic, pci-dss, nist-800-53)")
	segmenterConfig = flag.String("segmenter-config", "", "Segmenter configuration file")
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
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	
	command := os.Args[1]
	flag.CommandLine.Parse(os.Args[2:])
	
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
	
	// Validate
	if err := converter.ValidateLayer1(layer1Doc); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	
	// Save final document
	if err := store.SaveFinal(*documentID, layer1Doc, *outputFormat); err != nil {
		return fmt.Errorf("failed to save final document: %w", err)
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
  convert     Convert segmented data to Layer-1 format
  enhance     Enhance with LLM (can be re-run on existing data)
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

Enhance Options:
  --document-id <id>       Document ID (required)
  --llm-provider <name>    LLM provider (openai, anthropic, mock) [default: mock]
  --llm-model <model>      LLM model name
  --llm-api-key <key>      LLM API key (or set LLM_API_KEY env var)
  --temperature <t>        Temperature [default: 0.3]
  --max-tokens <n>         Max tokens [default: 2000]

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
  
  # List versions
  pipeline list --document-id pci-dss-3.2.1
`)
}

