package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ossf/gemara/layer1/pipeline/types"
)

func TestStorageCreation(t *testing.T) {
	tempDir := t.TempDir()
	
	store, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	
	if store.GetBaseDir() != tempDir {
		t.Errorf("Expected base dir %s, got %s", tempDir, store.GetBaseDir())
	}
}

func TestSaveAndLoadParsed(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	
	// Create test document
	doc := &types.ParsedDocument{
		Metadata: types.ParsedMetadata{
			SourceFile: "test.pdf",
			Parser:     "test-parser",
			ParsedAt:   time.Now(),
			DocumentID: "test-doc",
		},
		Pages: []types.Page{
			{
				PageNumber: 1,
				Blocks: []types.Block{
					{
						Type: types.BlockTypeParagraph,
						Text: "Test content",
					},
				},
			},
		},
	}
	
	// Save
	if err := store.SaveParsed(doc); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}
	
	if doc.Metadata.Version != 1 {
		t.Errorf("Expected version 1, got %d", doc.Metadata.Version)
	}
	
	// Load latest
	loaded, err := store.LoadParsed("test-doc", 0)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}
	
	if loaded.Metadata.DocumentID != "test-doc" {
		t.Errorf("Expected ID test-doc, got %s", loaded.Metadata.DocumentID)
	}
	
	if len(loaded.Pages) != 1 {
		t.Errorf("Expected 1 page, got %d", len(loaded.Pages))
	}
}

func TestSaveAndLoadSegmented(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	
	// Create test document
	doc := &types.SegmentedDocument{
		Metadata: types.SegmentedMetadata{
			DocumentID:  "test-doc",
			Segmenter:   "test-segmenter",
			SegmentedAt: time.Now(),
		},
		DocumentMetadata: types.DocumentMetadata{
			ID:          "TEST",
			Title:       "Test Document",
			Description: "Test description",
			Author:      "Test Author",
		},
		Categories: []types.SegmentCategory{
			{
				ID:          "CAT1",
				Title:       "Category 1",
				Description: "Test category",
			},
		},
	}
	
	// Save
	if err := store.SaveSegmented(doc); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}
	
	// Load
	loaded, err := store.LoadSegmented("test-doc", 0)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}
	
	if loaded.DocumentMetadata.Title != "Test Document" {
		t.Errorf("Expected title 'Test Document', got '%s'", loaded.DocumentMetadata.Title)
	}
}

func TestVersioning(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	
	// Save multiple versions
	for i := 1; i <= 3; i++ {
		doc := &types.ParsedDocument{
			Metadata: types.ParsedMetadata{
				SourceFile: "test.pdf",
				Parser:     "test-parser",
				ParsedAt:   time.Now(),
				DocumentID: "multi-version-doc",
			},
			Pages: []types.Page{
				{PageNumber: i},
			},
		}
		
		if err := store.SaveParsed(doc); err != nil {
			t.Fatalf("Failed to save version %d: %v", i, err)
		}
		
		if doc.Metadata.Version != i {
			t.Errorf("Expected version %d, got %d", i, doc.Metadata.Version)
		}
	}
	
	// List versions
	versions, err := store.ListVersions("multi-version-doc", "parsed")
	if err != nil {
		t.Fatalf("Failed to list versions: %v", err)
	}
	
	if len(versions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(versions))
	}
	
	// Verify sorted descending
	if versions[0].Version != 3 || versions[1].Version != 2 || versions[2].Version != 1 {
		t.Error("Versions not sorted correctly")
	}
	
	// Load specific version
	v2, err := store.LoadParsed("multi-version-doc", 2)
	if err != nil {
		t.Fatalf("Failed to load version 2: %v", err)
	}
	
	if v2.Metadata.Version != 2 {
		t.Errorf("Expected version 2, got %d", v2.Metadata.Version)
	}
	
	if len(v2.Pages) != 1 || v2.Pages[0].PageNumber != 2 {
		t.Error("Loaded wrong version content")
	}
}

func TestSaveFinal(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	
	data := map[string]interface{}{
		"id":    "TEST",
		"title": "Test Document",
	}
	
	// Save as YAML
	if err := store.SaveFinal("test-doc", data, "yaml"); err != nil {
		t.Fatalf("Failed to save YAML: %v", err)
	}
	
	yamlPath := filepath.Join(tempDir, "final", "test-doc.yaml")
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		t.Error("YAML file not created")
	}
	
	// Save as JSON
	if err := store.SaveFinal("test-doc-json", data, "json"); err != nil {
		t.Fatalf("Failed to save JSON: %v", err)
	}
	
	jsonPath := filepath.Join(tempDir, "final", "test-doc-json.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Error("JSON file not created")
	}
}

func TestListVersionsEmptyDir(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	
	// List versions for non-existent document
	versions, err := store.ListVersions("non-existent", "parsed")
	if err != nil {
		t.Fatalf("Expected no error for non-existent doc, got: %v", err)
	}
	
	if len(versions) != 0 {
		t.Errorf("Expected 0 versions, got %d", len(versions))
	}
}

func TestMultipleDocumentsSeparateVersions(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	
	// Save doc1 twice
	for i := 0; i < 2; i++ {
		doc := &types.ParsedDocument{
			Metadata: types.ParsedMetadata{
				SourceFile: "doc1.pdf",
				Parser:     "test",
				ParsedAt:   time.Now(),
				DocumentID: "doc1",
			},
			Pages: []types.Page{},
		}
		if err := store.SaveParsed(doc); err != nil {
			t.Fatalf("Failed to save doc1: %v", err)
		}
	}
	
	// Save doc2 three times
	for i := 0; i < 3; i++ {
		doc := &types.ParsedDocument{
			Metadata: types.ParsedMetadata{
				SourceFile: "doc2.pdf",
				Parser:     "test",
				ParsedAt:   time.Now(),
				DocumentID: "doc2",
			},
			Pages: []types.Page{},
		}
		if err := store.SaveParsed(doc); err != nil {
			t.Fatalf("Failed to save doc2: %v", err)
		}
	}
	
	// Verify separate version counts
	doc1Versions, _ := store.ListVersions("doc1", "parsed")
	doc2Versions, _ := store.ListVersions("doc2", "parsed")
	
	if len(doc1Versions) != 2 {
		t.Errorf("Expected 2 versions for doc1, got %d", len(doc1Versions))
	}
	
	if len(doc2Versions) != 3 {
		t.Errorf("Expected 3 versions for doc2, got %d", len(doc2Versions))
	}
}

func TestParsedAndSegmentedSameVersion(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	
	docID := "same-version-test"
	
	// Save parsed
	parsed := &types.ParsedDocument{
		Metadata: types.ParsedMetadata{
			DocumentID: docID,
			SourceFile: "test.pdf",
			Parser:     "test",
			ParsedAt:   time.Now(),
		},
		Pages: []types.Page{},
	}
	if err := store.SaveParsed(parsed); err != nil {
		t.Fatalf("Failed to save parsed: %v", err)
	}
	
	// Save segmented to same version directory
	segmented := &types.SegmentedDocument{
		Metadata: types.SegmentedMetadata{
			DocumentID:  docID,
			Segmenter:   "test",
			SegmentedAt: time.Now(),
		},
		DocumentMetadata: types.DocumentMetadata{
			ID:          "TEST",
			Title:       "Test",
			Description: "Test",
			Author:      "Test",
		},
		Categories: []types.SegmentCategory{},
	}
	if err := store.SaveSegmented(segmented); err != nil {
		t.Fatalf("Failed to save segmented: %v", err)
	}
	
	// Both should be in same version directory but with separate metadata files
	parsedVersions, _ := store.ListVersions(docID, "parsed")
	segmentedVersions, _ := store.ListVersions(docID, "segmented")
	
	if len(parsedVersions) != 1 {
		t.Errorf("Expected 1 parsed version, got %d", len(parsedVersions))
	}
	
	if len(segmentedVersions) != 1 {
		t.Errorf("Expected 1 segmented version, got %d", len(segmentedVersions))
	}
	
	// Verify both files exist in same directory
	versionDir := filepath.Join(tempDir, "intermediate", docID, "v1")
	parsedFile := filepath.Join(versionDir, "parsed.json")
	segmentedFile := filepath.Join(versionDir, "segmented.json")
	
	if _, err := os.Stat(parsedFile); os.IsNotExist(err) {
		t.Error("parsed.json not found")
	}
	
	if _, err := os.Stat(segmentedFile); os.IsNotExist(err) {
		t.Error("segmented.json not found")
	}
}

