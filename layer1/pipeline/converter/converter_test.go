package converter

import (
	"testing"
	"time"

	"github.com/ossf/gemara/layer1/pipeline/types"
)

func TestDefaultConverter(t *testing.T) {
	// Create test segmented document
	doc := &types.SegmentedDocument{
		Metadata: types.SegmentedMetadata{
			SourceVersion: 1,
			Segmenter:     "generic-v1.0",
			SegmentedAt:   time.Now(),
			Version:       1,
			DocumentID:    "test-doc",
		},
		DocumentMetadata: types.DocumentMetadata{
			ID:          "TEST-STD",
			Title:       "Test Security Standard",
			Description: "A test security standard",
			Author:      "Test Author",
			Version:     "1.0",
		},
		Categories: []types.SegmentCategory{
			{
				ID:          "AC",
				Title:       "Access Control",
				Description: "Access control requirements",
				Guidelines: []types.SegmentGuideline{
					{
						ID:        "AC-1",
						Title:     "User Authentication",
						Objective: "Ensure proper user authentication",
						Recommendations: []string{
							"Use strong passwords",
							"Implement MFA",
						},
						Parts: []types.SegmentPart{
							{
								ID:   "AC-1.1",
								Text: "All users must authenticate",
							},
						},
					},
				},
			},
		},
	}
	
	// Create converter
	conv := NewConverter()
	
	// Convert
	layer1Doc, err := conv.Convert(doc)
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}
	
	// Validate
	if err := ValidateLayer1(layer1Doc); err != nil {
		t.Fatalf("Validation failed: %v", err)
	}
	
	// Verify structure
	if layer1Doc.Metadata.Id != "TEST-STD" {
		t.Errorf("Expected ID 'TEST-STD', got '%s'", layer1Doc.Metadata.Id)
	}
	
	if len(layer1Doc.Categories) != 1 {
		t.Errorf("Expected 1 category, got %d", len(layer1Doc.Categories))
	}
	
	if len(layer1Doc.Categories[0].Guidelines) != 1 {
		t.Errorf("Expected 1 guideline, got %d", len(layer1Doc.Categories[0].Guidelines))
	}
	
	guideline := layer1Doc.Categories[0].Guidelines[0]
	if len(guideline.Recommendations) != 2 {
		t.Errorf("Expected 2 recommendations, got %d", len(guideline.Recommendations))
	}
	
	if len(guideline.GuidelineParts) != 1 {
		t.Errorf("Expected 1 part, got %d", len(guideline.GuidelineParts))
	}
}

func TestValidateLayer1(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*types.SegmentedDocument)
		wantErr bool
	}{
		{
			name:    "valid document",
			modify:  func(d *types.SegmentedDocument) {},
			wantErr: false,
		},
		{
			name: "missing metadata ID",
			modify: func(d *types.SegmentedDocument) {
				d.DocumentMetadata.ID = ""
			},
			wantErr: true,
		},
		{
			name: "missing category ID",
			modify: func(d *types.SegmentedDocument) {
				d.Categories[0].ID = ""
			},
			wantErr: true,
		},
		{
			name: "missing guideline ID",
			modify: func(d *types.SegmentedDocument) {
				d.Categories[0].Guidelines[0].ID = ""
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create base document
			doc := &types.SegmentedDocument{
				Metadata: types.SegmentedMetadata{
					DocumentID: "test",
				},
				DocumentMetadata: types.DocumentMetadata{
					ID:          "TEST",
					Title:       "Test",
					Description: "Test",
					Author:      "Test",
				},
				Categories: []types.SegmentCategory{
					{
						ID:          "C1",
						Title:       "Category 1",
						Description: "Test category",
						Guidelines: []types.SegmentGuideline{
							{
								ID:    "G1",
								Title: "Guideline 1",
							},
						},
					},
				},
			}
			
			// Apply modification
			tt.modify(doc)
			
			// Convert and validate
			conv := NewConverter()
			layer1Doc, err := conv.Convert(doc)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("Conversion failed: %v", err)
				}
				return
			}
			
			err = ValidateLayer1(layer1Doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLayer1() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

