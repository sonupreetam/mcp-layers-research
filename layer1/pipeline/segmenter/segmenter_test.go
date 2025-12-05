package segmenter

import (
	"testing"
	"time"

	"github.com/ossf/gemara/layer1/pipeline/types"
)

func TestGenericSegmenter(t *testing.T) {
	// Create test parsed document
	doc := &types.ParsedDocument{
		Metadata: types.ParsedMetadata{
			SourceFile: "test.pdf",
			Parser:     "simple-v1.0",
			ParsedAt:   time.Now(),
			DocumentID: "test-doc",
			Version:    1,
		},
		Pages: []types.Page{
			{
				PageNumber: 1,
				Blocks: []types.Block{
					{
						Type:       types.BlockTypeHeading,
						Level:      1,
						Text:       "Test Security Standard",
						FontSize:   24,
						FontWeight: "bold",
					},
					{
						Type: types.BlockTypeParagraph,
						Text: "Version 1.0",
					},
					{
						Type:       types.BlockTypeHeading,
						Level:      1,
						Text:       "1. Access Control",
						FontSize:   18,
						FontWeight: "bold",
					},
					{
						Type: types.BlockTypeParagraph,
						Text: "Implement strong access control measures.",
					},
					{
						Type:       types.BlockTypeHeading,
						Level:      2,
						Text:       "1.1 User Authentication",
						FontSize:   16,
						FontWeight: "bold",
					},
					{
						Type: types.BlockTypeParagraph,
						Text: "Ensure all users are properly authenticated before accessing systems.",
					},
				},
			},
		},
	}
	
	// Create segmenter
	config := types.SegmenterConfig{
		DocumentType: "generic",
	}
	
	seg, err := NewGenericSegmenter(config)
	if err != nil {
		t.Fatalf("Failed to create segmenter: %v", err)
	}
	
	// Segment document
	segmented, err := seg.Segment(doc)
	if err != nil {
		t.Fatalf("Failed to segment document: %v", err)
	}
	
	// Verify results
	if segmented.DocumentMetadata.Title == "" {
		t.Error("Expected title to be extracted")
	}
	
	if len(segmented.Categories) == 0 {
		t.Error("Expected at least one category")
	} else {
		t.Logf("Found %d categories", len(segmented.Categories))
		for _, cat := range segmented.Categories {
			t.Logf("  Category: %s - %s", cat.ID, cat.Title)
			for _, guide := range cat.Guidelines {
				t.Logf("    Guideline: %s - %s", guide.ID, guide.Title)
			}
		}
	}
}

func TestPCIDSSSegmenter(t *testing.T) {
	// Create test parsed document with PCI-DSS structure
	doc := &types.ParsedDocument{
		Metadata: types.ParsedMetadata{
			SourceFile: "pci-dss.pdf",
			Parser:     "simple-v1.0",
			ParsedAt:   time.Now(),
			DocumentID: "pci-dss-3.2.1",
			Version:    1,
		},
		Pages: []types.Page{
			{
				PageNumber: 1,
				Blocks: []types.Block{
					{
						Type:       types.BlockTypeHeading,
						Level:      1,
						Text:       "Payment Card Industry Data Security Standard",
						FontSize:   24,
						FontWeight: "bold",
					},
					{
						Type: types.BlockTypeParagraph,
						Text: "Version 3.2.1",
					},
					{
						Type:       types.BlockTypeHeading,
						Level:      1,
						Text:       "Requirement 1: Install and maintain a firewall configuration",
						FontSize:   18,
						FontWeight: "bold",
					},
					{
						Type:       types.BlockTypeHeading,
						Level:      2,
						Text:       "1.1 Establish firewall and router configuration standards",
						FontSize:   16,
					},
					{
						Type: types.BlockTypeParagraph,
						Text: "Objective: Ensure proper firewall configuration.",
					},
					{
						Type:       types.BlockTypeHeading,
						Level:      3,
						Text:       "1.1.1 A formal process for approving and testing all network connections",
						FontSize:   14,
					},
				},
			},
		},
	}
	
	// Create PCI-DSS segmenter
	config := types.SegmenterConfig{
		DocumentType: "pci-dss",
	}
	
	seg, err := NewPCIDSSSegmenter(config)
	if err != nil {
		t.Fatalf("Failed to create segmenter: %v", err)
	}
	
	// Segment document
	segmented, err := seg.Segment(doc)
	if err != nil {
		t.Fatalf("Failed to segment document: %v", err)
	}
	
	// Verify PCI-DSS specific processing
	if segmented.DocumentMetadata.DocumentType != "Standard" {
		t.Errorf("Expected document type 'Standard', got '%s'", segmented.DocumentMetadata.DocumentType)
	}
	
	if len(segmented.DocumentMetadata.IndustrySectors) == 0 {
		t.Error("Expected industry sectors to be set")
	}
	
	// Check category ID format
	if len(segmented.Categories) > 0 {
		catID := segmented.Categories[0].ID
		if catID[:4] != "REQ-" {
			t.Errorf("Expected category ID to start with 'REQ-', got '%s'", catID)
		}
	}
}

func TestSegmenterFactory(t *testing.T) {
	tests := []struct {
		docType string
		wantErr bool
	}{
		{"generic", false},
		{"pci-dss", false},
		{"nist-800-53", false},
		{"unknown", false}, // Should default to generic
	}
	
	for _, tt := range tests {
		t.Run(tt.docType, func(t *testing.T) {
			config := types.SegmenterConfig{
				DocumentType: tt.docType,
			}
			
			_, err := NewSegmenter(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSegmenter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

