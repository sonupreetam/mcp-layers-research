package converter

import (
	"fmt"

	"github.com/ossf/gemara/layer1"
	"github.com/ossf/gemara/layer1/pipeline/types"
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

// ValidateLayer1 validates a Layer-1 GuidanceDocument
func ValidateLayer1(doc *layer1.GuidanceDocument) error {
	if doc == nil {
		return fmt.Errorf("document is nil")
	}
	
	// Validate metadata
	if err := validateMetadata(&doc.Metadata); err != nil {
		return fmt.Errorf("metadata validation failed: %w", err)
	}
	
	// Validate categories
	if len(doc.Categories) == 0 {
		return fmt.Errorf("document must have at least one category")
	}
	
	for i, cat := range doc.Categories {
		if err := validateCategory(&cat, i); err != nil {
			return err
		}
	}
	
	return nil
}

// validateMetadata validates Layer-1 Metadata
func validateMetadata(meta *layer1.Metadata) error {
	if meta.Id == "" {
		return fmt.Errorf("metadata.id is required")
	}
	if meta.Title == "" {
		return fmt.Errorf("metadata.title is required")
	}
	if meta.Description == "" {
		return fmt.Errorf("metadata.description is required")
	}
	if meta.Author == "" {
		return fmt.Errorf("metadata.author is required")
	}
	return nil
}

// validateCategory validates Layer-1 Category
func validateCategory(cat *layer1.Category, index int) error {
	if cat.Id == "" {
		return fmt.Errorf("category[%d].id is required", index)
	}
	if cat.Title == "" {
		return fmt.Errorf("category[%d].title is required", index)
	}
	if cat.Description == "" {
		return fmt.Errorf("category[%d].description is required", index)
	}
	
	for j, guide := range cat.Guidelines {
		if err := validateGuideline(&guide, index, j); err != nil {
			return err
		}
	}
	
	return nil
}

// validateGuideline validates Layer-1 Guideline
func validateGuideline(guide *layer1.Guideline, catIndex, guideIndex int) error {
	if guide.Id == "" {
		return fmt.Errorf("category[%d].guideline[%d].id is required", catIndex, guideIndex)
	}
	if guide.Title == "" {
		return fmt.Errorf("category[%d].guideline[%d].title is required", catIndex, guideIndex)
	}
	
	for k, part := range guide.GuidelineParts {
		if err := validatePart(&part, catIndex, guideIndex, k); err != nil {
			return err
		}
	}
	
	return nil
}

// validatePart validates Layer-1 Part
func validatePart(part *layer1.Part, catIndex, guideIndex, partIndex int) error {
	if part.Id == "" {
		return fmt.Errorf("category[%d].guideline[%d].part[%d].id is required", catIndex, guideIndex, partIndex)
	}
	if part.Text == "" {
		return fmt.Errorf("category[%d].guideline[%d].part[%d].text is required", catIndex, guideIndex, partIndex)
	}
	
	return nil
}

