package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ossf/gemara/layer1/pipeline/types"
	"github.com/ossf/gemara/layer1/pipeline/converter"
	"github.com/ossf/gemara/layer1/pipeline/parser"
	"github.com/ossf/gemara/layer1/pipeline/segmenter"
	"github.com/ossf/gemara/layer1/pipeline/storage"
)

func TestFullPipeline(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	
	// Create sample document
	testFile := filepath.Join(tempDir, "sample.txt")
	sampleContent := `Payment Card Industry Data Security Standard
Version 3.2.1

Author: PCI Security Standards Council
Publication Date: May 2018

Introduction

The PCI Data Security Standard (PCI DSS) was developed to encourage and enhance
payment account data security and facilitate the broad adoption of consistent
data security measures globally.

Requirement 1: Install and maintain a firewall configuration to protect cardholder data

Firewalls are devices that control computer traffic allowed between an entity's
networks (internal) and untrusted networks (external), as well as traffic into
and out of more sensitive areas within an entity's internal trusted networks.

1.1 Establish firewall and router configuration standards

Objective: Build firewall and router configuration standards that formalize
testing whenever configurations change.

1.1.1 A formal process for approving and testing all network connections and
changes to the firewall and router configurations.

Guidance: A documented and implemented process for approving and testing all
connections and changes to the firewalls and routers will help prevent security
problems caused by misconfiguration of the network, router, or firewall.

1.1.2 Current network diagram that identifies all connections between the
cardholder data environment and other networks, including any wireless networks.

Requirement 2: Do not use vendor-supplied defaults for system passwords

2.1 Always change vendor-supplied defaults and remove or disable unnecessary
default accounts before installing a system on the network.

Objective: Malicious individuals often use vendor default passwords and other
vendor default settings to compromise systems.
`
	
	if err := os.WriteFile(testFile, []byte(sampleContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// Initialize storage
	storageDir := filepath.Join(tempDir, "storage")
	store, err := storage.NewStorage(storageDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	
	documentID := "pci-dss-test"
	
	// Step 1: Parse
	t.Log("Step 1: Parsing...")
	parserConfig := types.ParserConfig{
		Provider: "simple",
		TempDir:  tempDir,
	}
	
	simpleParser, err := parser.NewSimpleParser(parserConfig)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	
	parsed, err := simpleParser.ParseTextFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	
	parsed.Metadata.DocumentID = documentID
	
	t.Logf("Saving parsed document with ID: %s", documentID)
	if err := store.SaveParsed(parsed); err != nil {
		t.Fatalf("Failed to save parsed: %v", err)
	}
	t.Logf("Saved parsed document as version %d", parsed.Metadata.Version)
	
	t.Logf("  Parsed: %d pages, %d blocks", len(parsed.Pages), countTestBlocks(parsed))
	
	// Step 2: Segment
	t.Log("Step 2: Segmenting...")
	segmenterConfig := types.SegmenterConfig{
		DocumentType: "pci-dss",
	}
	
	seg, err := segmenter.NewSegmenter(segmenterConfig)
	if err != nil {
		t.Fatalf("Failed to create segmenter: %v", err)
	}
	
	segmented, err := seg.Segment(parsed)
	if err != nil {
		t.Fatalf("Failed to segment: %v", err)
	}
	
	t.Logf("Saving segmented document with ID: %s", segmented.Metadata.DocumentID)
	if err := store.SaveSegmented(segmented); err != nil {
		t.Fatalf("Failed to save segmented: %v", err)
	}
	t.Logf("Saved segmented document as version %d", segmented.Metadata.Version)
	
	t.Logf("  Segmented: %d categories, %d guidelines",
		len(segmented.Categories),
		countTestSegmentedGuidelines(segmented))
	
	// Step 3: Convert
	t.Log("Step 3: Converting to Layer-1...")
	conv := converter.NewConverter()
	
	layer1Doc, err := conv.Convert(segmented)
	if err != nil {
		t.Fatalf("Failed to convert: %v", err)
	}
	
	if err := converter.ValidateLayer1(layer1Doc); err != nil {
		t.Fatalf("Validation failed: %v", err)
	}
	
	if err := store.SaveFinal(documentID, layer1Doc, "yaml"); err != nil {
		t.Fatalf("Failed to save final: %v", err)
	}
	
	t.Logf("  Converted: %d categories, %d guidelines",
		len(layer1Doc.Categories),
		countTestLayer1Guidelines(layer1Doc))
	
	// Verify final output exists
	finalPath := filepath.Join(storageDir, "final", documentID+".yaml")
	if _, err := os.Stat(finalPath); err != nil {
		t.Errorf("Final output not found: %v", err)
	}
	
	// Step 4: List versions
	t.Log("Step 4: Listing versions...")
	
	// First, check if files actually exist
	expectedDir := filepath.Join(storageDir, "intermediate", documentID, "v1")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("Expected directory does not exist: %s", expectedDir)
	} else {
		t.Logf("Directory exists: %s", expectedDir)
		entries, _ := os.ReadDir(expectedDir)
		t.Logf("Files in directory:")
		for _, e := range entries {
			t.Logf("  - %s", e.Name())
		}
	}
	
	parsedVersions, err := store.ListVersions(documentID, "parsed")
	if err != nil {
		t.Fatalf("Failed to list parsed versions: %v", err)
	}
	
	t.Logf("Found %d parsed versions", len(parsedVersions))
	for _, v := range parsedVersions {
		t.Logf("  Version %d: %s", v.Version, v.StoredAt)
	}
	
	if len(parsedVersions) != 1 {
		// Check if files exist
		t.Logf("Storage base dir: %s", store.GetBaseDir())
		t.Logf("Expected path: %s/intermediate/%s/v1/", store.GetBaseDir(), documentID)
		t.Errorf("Expected 1 parsed version, got %d", len(parsedVersions))
	}
	
	segmentedVersions, err := store.ListVersions(documentID, "segmented")
	if err != nil {
		t.Fatalf("Failed to list segmented versions: %v", err)
	}
	
	if len(segmentedVersions) != 1 {
		t.Errorf("Expected 1 segmented version, got %d", len(segmentedVersions))
	}
	
	t.Log("Pipeline test complete!")
}

func TestStorageVersioning(t *testing.T) {
	tempDir := t.TempDir()
	store, err := storage.NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	
	documentID := "version-test"
	
	// Save multiple versions
	for i := 1; i <= 3; i++ {
		doc := &types.ParsedDocument{
			Metadata: types.ParsedMetadata{
				SourceFile: "test.pdf",
				Parser:     "simple-v1.0",
				ParsedAt:   time.Now(),
				DocumentID: documentID,
			},
			Pages: []types.Page{
				{PageNumber: i},
			},
		}
		
		if err := store.SaveParsed(doc); err != nil {
			t.Fatalf("Failed to save version %d: %v", i, err)
		}
	}
	
	// List versions
	versions, err := store.ListVersions(documentID, "parsed")
	if err != nil {
		t.Fatalf("Failed to list versions: %v", err)
	}
	
	if len(versions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(versions))
	}
	
	// Load latest
	latest, err := store.LoadParsed(documentID, 0)
	if err != nil {
		t.Fatalf("Failed to load latest: %v", err)
	}
	
	if latest.Metadata.Version != 3 {
		t.Errorf("Expected latest version 3, got %d", latest.Metadata.Version)
	}
	
	// Load specific version
	v2, err := store.LoadParsed(documentID, 2)
	if err != nil {
		t.Fatalf("Failed to load version 2: %v", err)
	}
	
	if v2.Metadata.Version != 2 {
		t.Errorf("Expected version 2, got %d", v2.Metadata.Version)
	}
}

func countTestBlocks(doc *types.ParsedDocument) int {
	count := 0
	for _, page := range doc.Pages {
		count += len(page.Blocks)
	}
	return count
}

func countTestSegmentedGuidelines(doc *types.SegmentedDocument) int {
	count := 0
	for _, cat := range doc.Categories {
		count += len(cat.Guidelines)
	}
	return count
}

func countTestLayer1Guidelines(doc interface{}) int {
	// This is a placeholder - would need proper type assertion
	return 0
}

