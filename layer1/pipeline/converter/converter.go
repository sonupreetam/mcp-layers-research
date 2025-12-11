package converter

import (
	"fmt"

	"github.com/ossf/gemara/layer1"
	"github.com/ossf/gemara/layer1/pipeline/types"
	"github.com/ossf/gemara/layer1/pipeline/validator"
)

// Converter transforms segmented documents into Layer-1 format
type Converter interface {
	// Convert transforms segmented document to Layer-1 GuidanceDocument
	Convert(doc *types.SegmentedDocument) (*layer1.GuidanceDocument, error)
	
	// Name returns the converter name
	Name() string
}

// DefaultConverter provides standard conversion logic
type DefaultConverter struct {
	preserveIDs bool
}

// NewConverter creates a new converter
func NewConverter() *DefaultConverter {
	return &DefaultConverter{
		preserveIDs: true,
	}
}

// Name returns the converter name
func (c *DefaultConverter) Name() string {
	return "default-v1.0"
}

// Convert transforms segmented document to Layer-1 GuidanceDocument
func (c *DefaultConverter) Convert(doc *types.SegmentedDocument) (*layer1.GuidanceDocument, error) {
	if doc == nil {
		return nil, fmt.Errorf("segmented document is nil")
	}
	
	// Convert metadata
	metadata := c.convertMetadata(&doc.DocumentMetadata)
	
	// Convert categories
	categories := make([]layer1.Category, 0, len(doc.Categories))
	for _, segCat := range doc.Categories {
		cat := c.convertCategory(&segCat)
		categories = append(categories, cat)
	}
	
	guidanceDoc := &layer1.GuidanceDocument{
		Metadata:    metadata,
		FrontMatter: doc.FrontMatter,
		Categories:  categories,
	}
	
	return guidanceDoc, nil
}

// convertMetadata converts DocumentMetadata to Layer-1 Metadata
func (c *DefaultConverter) convertMetadata(meta *types.DocumentMetadata) layer1.Metadata {
	l1Meta := layer1.Metadata{
		Id:              meta.ID,
		Title:           meta.Title,
		Description:     meta.Description,
		Author:          meta.Author,
		Version:         meta.Version,
		PublicationDate: meta.PublicationDate,
	}
	
	// Convert document type
	if meta.DocumentType != "" {
		l1Meta.DocumentType = layer1.DocumentType(meta.DocumentType)
	}
	
	// Convert applicability
	if len(meta.IndustrySectors) > 0 || len(meta.Jurisdictions) > 0 {
		l1Meta.Applicability = &layer1.Applicability{
			IndustrySectors: meta.IndustrySectors,
			Jurisdictions:   meta.Jurisdictions,
		}
	}
	
	return l1Meta
}

// convertCategory converts SegmentCategory to Layer-1 Category
func (c *DefaultConverter) convertCategory(cat *types.SegmentCategory) layer1.Category {
	guidelines := make([]layer1.Guideline, 0, len(cat.Guidelines))
	for _, segGuide := range cat.Guidelines {
		guide := c.convertGuideline(&segGuide)
		guidelines = append(guidelines, guide)
	}
	
	return layer1.Category{
		Id:          cat.ID,
		Title:       cat.Title,
		Description: cat.Description,
		Guidelines:  guidelines,
	}
}

// convertGuideline converts SegmentGuideline to Layer-1 Guideline
func (c *DefaultConverter) convertGuideline(guide *types.SegmentGuideline) layer1.Guideline {
	parts := make([]layer1.Part, 0, len(guide.Parts))
	for _, segPart := range guide.Parts {
		part := c.convertPart(&segPart)
		parts = append(parts, part)
	}
	
	l1Guide := layer1.Guideline{
		Id:              guide.ID,
		Title:           guide.Title,
		Objective:       guide.Objective,
		Recommendations: guide.Recommendations,
		GuidelineParts:  parts,
	}
	
	return l1Guide
}

// convertPart converts SegmentPart to Layer-1 Part
func (c *DefaultConverter) convertPart(part *types.SegmentPart) layer1.Part {
	return layer1.Part{
		Id:              part.ID,
		Title:           part.Title,
		Text:            part.Text,
		Recommendations: part.Recommendations,
	}
}

// ValidateLayer1 validates a Layer-1 GuidanceDocument using the schema validator
func ValidateLayer1(doc *layer1.GuidanceDocument) error {
	v := validator.NewValidator()
	result := v.Validate(doc)
	if !result.Valid {
		return fmt.Errorf("validation failed: %s", result.Error())
	}
	return nil
}

// ValidateLayer1Strict performs strict schema validation
func ValidateLayer1Strict(doc *layer1.GuidanceDocument) error {
	v := validator.NewValidator(validator.WithStrictMode(true))
	result := v.Validate(doc)
	if !result.Valid {
		return fmt.Errorf("strict validation failed: %s", result.Error())
	}
	return nil
}

// ValidateWithResult returns the full validation result with details
func ValidateWithResult(doc *layer1.GuidanceDocument) *validator.ValidationResult {
	v := validator.NewValidator(validator.WithStrictMode(true))
	return v.Validate(doc)
}

// ConvertAndValidate converts a segmented document and validates the result
func (c *DefaultConverter) ConvertAndValidate(doc *types.SegmentedDocument, strict bool) (*layer1.GuidanceDocument, *validator.ValidationResult, error) {
	// First perform conversion
	layer1Doc, err := c.Convert(doc)
	if err != nil {
		return nil, nil, fmt.Errorf("conversion failed: %w", err)
	}

	// Then validate the result
	var v *validator.Validator
	if strict {
		v = validator.NewValidator(validator.WithStrictMode(true))
	} else {
		v = validator.NewValidator()
	}

	result := v.Validate(layer1Doc)
	return layer1Doc, result, nil
}


