package validator

import (
	"testing"

	"github.com/ossf/gemara/layer1"
)

func TestValidator_ValidDocument(t *testing.T) {
	doc := &layer1.GuidanceDocument{
		Metadata: layer1.Metadata{
			Id:           "test-doc-1",
			Title:        "Test Document",
			Description:  "A test document for validation",
			Author:       "Test Author",
			DocumentType: "Standard",
		},
		Categories: []layer1.Category{
			{
				Id:          "cat-1",
				Title:       "Category 1",
				Description: "First category",
				Guidelines: []layer1.Guideline{
					{
						Id:    "guide-1",
						Title: "Guideline 1",
						GuidelineParts: []layer1.Part{
							{
								Id:   "part-1",
								Text: "Part 1 text",
							},
						},
					},
				},
			},
		},
	}

	v := NewValidator()
	result := v.Validate(doc)

	if !result.Valid {
		t.Errorf("Expected valid document, got errors: %v", result.Errors)
	}
}

func TestValidator_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		doc         *layer1.GuidanceDocument
		expectedErr string
	}{
		{
			name:        "nil document",
			doc:         nil,
			expectedErr: "document is nil",
		},
		{
			name: "missing metadata id",
			doc: &layer1.GuidanceDocument{
				Metadata: layer1.Metadata{
					Title:       "Test",
					Description: "Test",
					Author:      "Test",
				},
			},
			expectedErr: "metadata.id",
		},
		{
			name: "missing metadata title",
			doc: &layer1.GuidanceDocument{
				Metadata: layer1.Metadata{
					Id:          "test",
					Description: "Test",
					Author:      "Test",
				},
			},
			expectedErr: "metadata.title",
		},
		{
			name: "missing category id",
			doc: &layer1.GuidanceDocument{
				Metadata: layer1.Metadata{
					Id:          "test",
					Title:       "Test",
					Description: "Test",
					Author:      "Test",
				},
				Categories: []layer1.Category{
					{
						Title:       "Cat",
						Description: "Cat desc",
					},
				},
			},
			expectedErr: "categories[0].id",
		},
		{
			name: "missing guideline id",
			doc: &layer1.GuidanceDocument{
				Metadata: layer1.Metadata{
					Id:          "test",
					Title:       "Test",
					Description: "Test",
					Author:      "Test",
				},
				Categories: []layer1.Category{
					{
						Id:          "cat-1",
						Title:       "Cat",
						Description: "Cat desc",
						Guidelines: []layer1.Guideline{
							{
								Title: "Guide without ID",
							},
						},
					},
				},
			},
			expectedErr: "categories[0].guidelines[0].id",
		},
		{
			name: "missing part text",
			doc: &layer1.GuidanceDocument{
				Metadata: layer1.Metadata{
					Id:          "test",
					Title:       "Test",
					Description: "Test",
					Author:      "Test",
				},
				Categories: []layer1.Category{
					{
						Id:          "cat-1",
						Title:       "Cat",
						Description: "Cat desc",
						Guidelines: []layer1.Guideline{
							{
								Id:    "guide-1",
								Title: "Guide 1",
								GuidelineParts: []layer1.Part{
									{
										Id: "part-1",
										// Missing Text
									},
								},
							},
						},
					},
				},
			},
			expectedErr: "categories[0].guidelines[0].guideline-parts[0].text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			result := v.Validate(tt.doc)

			if result.Valid {
				t.Error("Expected validation to fail")
				return
			}

			found := false
			for _, err := range result.Errors {
				if err.Path == tt.expectedErr || (tt.expectedErr == "document is nil" && err.Message == tt.expectedErr) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected error path %q, got errors: %v", tt.expectedErr, result.Errors)
			}
		})
	}
}

func TestValidator_InvalidDocumentType(t *testing.T) {
	doc := &layer1.GuidanceDocument{
		Metadata: layer1.Metadata{
			Id:           "test",
			Title:        "Test",
			Description:  "Test",
			Author:       "Test",
			DocumentType: "InvalidType",
		},
	}

	v := NewValidator()
	result := v.Validate(doc)

	if result.Valid {
		t.Error("Expected validation to fail for invalid document type")
	}

	found := false
	for _, err := range result.Errors {
		if err.Path == "metadata.document-type" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected error for invalid document type, got: %v", result.Errors)
	}
}

func TestValidator_ValidDocumentTypes(t *testing.T) {
	validTypes := []layer1.DocumentType{"Standard", "Regulation", "Best Practice", "Framework"}

	for _, docType := range validTypes {
		t.Run(string(docType), func(t *testing.T) {
			doc := &layer1.GuidanceDocument{
				Metadata: layer1.Metadata{
					Id:           "test",
					Title:        "Test",
					Description:  "Test",
					Author:       "Test",
					DocumentType: docType,
				},
				Categories: []layer1.Category{
					{
						Id:          "cat-1",
						Title:       "Cat",
						Description: "Desc",
					},
				},
			}

			v := NewValidator()
			result := v.Validate(doc)

			// Filter out errors not related to document type
			for _, err := range result.Errors {
				if err.Path == "metadata.document-type" {
					t.Errorf("Document type %s should be valid, got error: %v", docType, err)
				}
			}
		})
	}
}

func TestValidator_DuplicateIDs(t *testing.T) {
	doc := &layer1.GuidanceDocument{
		Metadata: layer1.Metadata{
			Id:          "test",
			Title:       "Test",
			Description: "Test",
			Author:      "Test",
		},
		Categories: []layer1.Category{
			{
				Id:          "cat-1",
				Title:       "Cat 1",
				Description: "Desc",
			},
			{
				Id:          "cat-1", // Duplicate
				Title:       "Cat 2",
				Description: "Desc",
			},
		},
	}

	v := NewValidator()
	result := v.Validate(doc)

	if result.Valid {
		t.Error("Expected validation to fail for duplicate category IDs")
	}

	found := false
	for _, err := range result.Errors {
		if err.Path == "categories[1].id" && err.Message == "duplicate category ID" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected duplicate ID error, got: %v", result.Errors)
	}
}

func TestValidator_StrictMode(t *testing.T) {
	doc := &layer1.GuidanceDocument{
		Metadata: layer1.Metadata{
			Id:          "test",
			Title:       "Test",
			Description: "Test",
			Author:      "Test",
			// No DocumentType - should fail in strict mode
		},
		// No categories - should fail in strict mode
	}

	// Non-strict mode should pass (for required fields)
	v := NewValidator()
	result := v.Validate(doc)
	// Note: still might have other errors, but check strict-specific ones

	// Strict mode should catch missing document type
	vStrict := NewValidator(WithStrictMode(true))
	resultStrict := vStrict.Validate(doc)

	if resultStrict.Valid {
		t.Error("Expected strict validation to fail")
	}

	foundDocType := false
	foundCategories := false
	for _, err := range resultStrict.Errors {
		if err.Path == "metadata.document-type" {
			foundDocType = true
		}
		if err.Path == "categories" {
			foundCategories = true
		}
	}

	if !foundDocType {
		t.Error("Strict mode should require document type")
	}
	if !foundCategories {
		t.Error("Strict mode should require at least one category")
	}

	_ = result // suppress unused warning
}

func TestValidator_RationaleValidation(t *testing.T) {
	doc := &layer1.GuidanceDocument{
		Metadata: layer1.Metadata{
			Id:          "test",
			Title:       "Test",
			Description: "Test",
			Author:      "Test",
		},
		Categories: []layer1.Category{
			{
				Id:          "cat-1",
				Title:       "Cat",
				Description: "Desc",
				Guidelines: []layer1.Guideline{
					{
						Id:    "guide-1",
						Title: "Guide 1",
						Rationale: &layer1.Rationale{
							Risks: []layer1.Risk{
								{
									Title: "", // Missing title
									Description: "Risk desc",
								},
							},
							Outcomes: []layer1.Outcome{
								{
									Title:       "Outcome",
									Description: "", // Missing description
								},
							},
						},
					},
				},
			},
		},
	}

	v := NewValidator()
	result := v.Validate(doc)

	if result.Valid {
		t.Error("Expected validation to fail for invalid rationale")
	}

	foundRiskTitle := false
	foundOutcomeDesc := false
	for _, err := range result.Errors {
		if err.Path == "categories[0].guidelines[0].rationale.risks[0].title" {
			foundRiskTitle = true
		}
		if err.Path == "categories[0].guidelines[0].rationale.outcomes[0].description" {
			foundOutcomeDesc = true
		}
	}

	if !foundRiskTitle {
		t.Error("Should detect missing risk title")
	}
	if !foundOutcomeDesc {
		t.Error("Should detect missing outcome description")
	}
}

func TestValidator_MappingValidation(t *testing.T) {
	doc := &layer1.GuidanceDocument{
		Metadata: layer1.Metadata{
			Id:          "test",
			Title:       "Test",
			Description: "Test",
			Author:      "Test",
			MappingReferences: []layer1.MappingReference{
				{
					Id:    "", // Missing ID
					Title: "Ref",
					Version: "1.0",
				},
			},
		},
		Categories: []layer1.Category{
			{
				Id:          "cat-1",
				Title:       "Cat",
				Description: "Desc",
				Guidelines: []layer1.Guideline{
					{
						Id:    "guide-1",
						Title: "Guide 1",
						GuidelineMappings: []layer1.Mapping{
							{
								ReferenceId: "", // Missing reference ID
								Entries: []layer1.MappingEntry{
									{
										ReferenceId: "ref-1",
										Strength:    -5, // Invalid negative strength
									},
								},
							},
						},
					},
				},
			},
		},
	}

	v := NewValidator()
	result := v.Validate(doc)

	if result.Valid {
		t.Error("Expected validation to fail for invalid mappings")
	}

	expectedErrors := map[string]bool{
		"metadata.mapping-references[0].id":                               false,
		"categories[0].guidelines[0].guideline-mappings[0].reference-id":  false,
		"categories[0].guidelines[0].guideline-mappings[0].entries[0].strength": false,
	}

	for _, err := range result.Errors {
		if _, ok := expectedErrors[err.Path]; ok {
			expectedErrors[err.Path] = true
		}
	}

	for path, found := range expectedErrors {
		if !found {
			t.Errorf("Expected error for path %s", path)
		}
	}
}

func TestQuickValidate(t *testing.T) {
	// Valid document
	validDoc := &layer1.GuidanceDocument{
		Metadata: layer1.Metadata{
			Id:           "test",
			Title:        "Test",
			Description:  "Test",
			Author:       "Test",
			DocumentType: "Standard",
		},
		Categories: []layer1.Category{
			{
				Id:          "cat-1",
				Title:       "Cat",
				Description: "Desc",
			},
		},
	}

	if err := QuickValidate(validDoc); err != nil {
		t.Errorf("Expected valid document to pass QuickValidate: %v", err)
	}

	// Invalid document
	invalidDoc := &layer1.GuidanceDocument{
		Metadata: layer1.Metadata{
			Id: "test",
			// Missing required fields
		},
	}

	if err := QuickValidate(invalidDoc); err == nil {
		t.Error("Expected invalid document to fail QuickValidate")
	}
}

func TestValidateJSON(t *testing.T) {
	validJSON := `{
		"metadata": {
			"id": "test",
			"title": "Test",
			"description": "Test desc",
			"author": "Test Author"
		},
		"categories": [
			{
				"id": "cat-1",
				"title": "Category 1",
				"description": "First category"
			}
		]
	}`

	v := NewValidator()
	result, err := v.ValidateJSON([]byte(validJSON))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Valid {
		t.Errorf("Expected valid JSON to pass, got errors: %v", result.Errors)
	}

	// Invalid JSON syntax
	invalidJSON := `{invalid json}`
	result, err = v.ValidateJSON([]byte(invalidJSON))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Valid {
		t.Error("Expected invalid JSON to fail")
	}
	
	// Valid JSON syntax but invalid schema (empty categories)
	invalidSchemaJSON := `{
		"metadata": {
			"id": "test",
			"title": "Test",
			"description": "Test desc",
			"author": "Test Author"
		},
		"categories": []
	}`
	result, err = v.ValidateJSON([]byte(invalidSchemaJSON))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result.Valid {
		t.Error("Expected empty categories to fail validation")
	}
}
