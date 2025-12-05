package segmenter

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ossf/gemara/layer1/pipeline/types"
)

// Segmenter applies rules to segment parsed documents into structured blocks
type Segmenter interface {
	// Segment converts parsed document into segmented structure
	Segment(doc *types.ParsedDocument) (*types.SegmentedDocument, error)
	
	// Name returns the segmenter name
	Name() string
	
	// Configure sets segmenter-specific options
	Configure(config types.SegmenterConfig) error
}

// NewSegmenter creates a segmenter based on document type
func NewSegmenter(config types.SegmenterConfig) (Segmenter, error) {
	switch config.DocumentType {
	case "pci-dss":
		return NewPCIDSSSegmenter(config)
	case "nist-800-53":
		return NewNIST80053Segmenter(config)
	case "generic":
		return NewGenericSegmenter(config)
	default:
		// Default to generic segmenter
		config.DocumentType = "generic"
		return NewGenericSegmenter(config)
	}
}

// SegmenterBase provides common functionality
type SegmenterBase struct {
	config types.SegmenterConfig
	rules  *SegmentationRules
}

// Configure sets the segmenter configuration
func (s *SegmenterBase) Configure(config types.SegmenterConfig) error {
	s.config = config
	return nil
}

// GetConfig returns the segmenter configuration
func (s *SegmenterBase) GetConfig() types.SegmenterConfig {
	return s.config
}

// SegmentationRules defines patterns for identifying document structure
type SegmentationRules struct {
	// Regex patterns
	CategoryPattern  *regexp.Regexp
	GuidelinePattern *regexp.Regexp
	PartPattern      *regexp.Regexp
	
	// Metadata patterns
	TitlePatterns       []*regexp.Regexp
	VersionPatterns     []*regexp.Regexp
	AuthorPatterns      []*regexp.Regexp
	PublicationPatterns []*regexp.Regexp
	
	// Content patterns
	ObjectiveKeywords     []string
	RecommendationKeywords []string
	RequirementKeywords   []string
	
	// Structure hints
	CategoryHeadingLevel  int
	GuidelineHeadingLevel int
	PartHeadingLevel      int
}

// GenericSegmenter uses generic rules for document segmentation
type GenericSegmenter struct {
	SegmenterBase
}

// NewGenericSegmenter creates a new generic segmenter
func NewGenericSegmenter(config types.SegmenterConfig) (*GenericSegmenter, error) {
	s := &GenericSegmenter{}
	if err := s.Configure(config); err != nil {
		return nil, err
	}
	
	// Initialize generic rules
	s.rules = &SegmentationRules{
		CategoryPattern:  regexp.MustCompile(`^([0-9]+)\.\s+([A-Z].*)`),
		GuidelinePattern: regexp.MustCompile(`^([0-9]+\.[0-9]+)\s+([A-Z].*)`),
		PartPattern:      regexp.MustCompile(`^([0-9]+\.[0-9]+\.[0-9]+)\s+(.*)`),
		
		TitlePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^title:\s*(.+)`),
			regexp.MustCompile(`(?i)^(.+?)(Standard|Framework|Guidelines?|Requirements?)(\s+Version|\s+v\.|$)`),
		},
		VersionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)version\s*:?\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`),
			regexp.MustCompile(`(?i)v\.?\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`),
		},
		AuthorPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^author:\s*(.+)`),
			regexp.MustCompile(`(?i)^by\s+(.+)`),
		},
		PublicationPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:published|publication\s+date):\s*(.+)`),
			regexp.MustCompile(`(?i)(January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2},?\s+\d{4}`),
		},
		
		ObjectiveKeywords: []string{
			"objective", "purpose", "goal", "intent",
		},
		RecommendationKeywords: []string{
			"recommendation", "guidance", "implementation", "should", "must",
		},
		RequirementKeywords: []string{
			"requirement", "shall", "must", "required",
		},
		
		CategoryHeadingLevel:  1,
		GuidelineHeadingLevel: 2,
		PartHeadingLevel:      3,
	}
	
	return s, nil
}

// Name returns the segmenter name
func (s *GenericSegmenter) Name() string {
	return "generic-v1.0"
}

// Segment converts parsed document into segmented structure
func (s *GenericSegmenter) Segment(doc *types.ParsedDocument) (*types.SegmentedDocument, error) {
	// Extract metadata
	metadata := s.extractMetadata(doc)
	
	// Extract front matter (everything before first category)
	frontMatter := s.extractFrontMatter(doc)
	
	// Extract categories and guidelines
	categories := s.extractCategories(doc)
	
	segmented := &types.SegmentedDocument{
		Metadata: types.SegmentedMetadata{
			SourceVersion: doc.Metadata.Version,
			Segmenter:     s.Name(),
			SegmentedAt:   time.Now(),
			DocumentID:    doc.Metadata.DocumentID,
		},
		DocumentMetadata: metadata,
		FrontMatter:      frontMatter,
		Categories:       categories,
	}
	
	return segmented, nil
}

// extractMetadata extracts document metadata from parsed content
func (s *GenericSegmenter) extractMetadata(doc *types.ParsedDocument) types.DocumentMetadata {
	meta := types.DocumentMetadata{
		ID: doc.Metadata.DocumentID,
	}
	
	// Look through first few pages for metadata
	for i := 0; i < len(doc.Pages) && i < 5; i++ {
		page := doc.Pages[i]
		for _, block := range page.Blocks {
			text := block.Text
			
			// Try to extract title
			if meta.Title == "" {
				for _, pattern := range s.rules.TitlePatterns {
					if matches := pattern.FindStringSubmatch(text); matches != nil && len(matches) > 1 {
						meta.Title = strings.TrimSpace(matches[1])
						break
					}
				}
				
				// If no pattern match, use first heading
				if meta.Title == "" && block.Type == types.BlockTypeHeading && block.Level == 1 {
					meta.Title = text
				}
			}
			
			// Try to extract version
			if meta.Version == "" {
				for _, pattern := range s.rules.VersionPatterns {
					if matches := pattern.FindStringSubmatch(text); matches != nil && len(matches) > 1 {
						meta.Version = matches[1]
						break
					}
				}
			}
			
			// Try to extract author
			if meta.Author == "" {
				for _, pattern := range s.rules.AuthorPatterns {
					if matches := pattern.FindStringSubmatch(text); matches != nil && len(matches) > 1 {
						meta.Author = strings.TrimSpace(matches[1])
						break
					}
				}
			}
			
			// Try to extract publication date
			if meta.PublicationDate == "" {
				for _, pattern := range s.rules.PublicationPatterns {
					if matches := pattern.FindStringSubmatch(text); matches != nil && len(matches) >= 1 {
						meta.PublicationDate = strings.TrimSpace(matches[len(matches)-1])
						break
					}
				}
			}
		}
	}
	
	// Set defaults if not found
	if meta.Title == "" {
		meta.Title = "Untitled Document"
	}
	if meta.Author == "" {
		meta.Author = "Unknown"
	}
	if meta.Description == "" {
		meta.Description = "Automatically extracted from PDF"
	}
	
	return meta
}

// extractFrontMatter extracts introductory text
func (s *GenericSegmenter) extractFrontMatter(doc *types.ParsedDocument) string {
	var frontMatter strings.Builder
	foundFirstCategory := false
	
	for _, page := range doc.Pages {
		for _, block := range page.Blocks {
			// Stop at first category
			if s.rules.CategoryPattern.MatchString(block.Text) {
				foundFirstCategory = true
				break
			}
			
			// Skip headings that look like title/metadata
			if block.Type == types.BlockTypeHeading && block.Level <= 1 {
				continue
			}
			
			// Collect paragraph content
			if block.Type == types.BlockTypeParagraph {
				if frontMatter.Len() > 0 {
					frontMatter.WriteString("\n\n")
				}
				frontMatter.WriteString(block.Text)
			}
		}
		
		if foundFirstCategory {
			break
		}
	}
	
	return strings.TrimSpace(frontMatter.String())
}

// extractCategories extracts categories and their guidelines
func (s *GenericSegmenter) extractCategories(doc *types.ParsedDocument) []types.SegmentCategory {
	var categories []types.SegmentCategory
	var currentCategory *types.SegmentCategory
	var currentGuideline *types.SegmentGuideline
	var currentText strings.Builder
	
	for _, page := range doc.Pages {
		for _, block := range page.Blocks {
			text := block.Text
			
			// Check for category (e.g., "1. Category Name")
			if matches := s.rules.CategoryPattern.FindStringSubmatch(text); matches != nil {
				// Save previous guideline
				if currentGuideline != nil && currentText.Len() > 0 {
					s.finalizeGuideline(currentGuideline, currentText.String())
					currentText.Reset()
				}
				
				// Save previous category
				if currentCategory != nil {
					if currentGuideline != nil {
						currentCategory.Guidelines = append(currentCategory.Guidelines, *currentGuideline)
					}
					categories = append(categories, *currentCategory)
				}
				
				// Start new category
				currentCategory = &types.SegmentCategory{
					ID:    matches[1],
					Title: strings.TrimSpace(matches[2]),
				}
				currentGuideline = nil
				continue
			}
			
			// Check for guideline (e.g., "1.1 Guideline Name")
			if matches := s.rules.GuidelinePattern.FindStringSubmatch(text); matches != nil {
				// Save previous guideline
				if currentGuideline != nil && currentText.Len() > 0 {
					s.finalizeGuideline(currentGuideline, currentText.String())
					currentText.Reset()
				}
				
				if currentCategory != nil && currentGuideline != nil {
					currentCategory.Guidelines = append(currentCategory.Guidelines, *currentGuideline)
				}
				
				// Start new guideline
				currentGuideline = &types.SegmentGuideline{
					ID:    matches[1],
					Title: strings.TrimSpace(matches[2]),
				}
				continue
			}
			
			// Check for part (e.g., "1.1.1 Part Text")
			if matches := s.rules.PartPattern.FindStringSubmatch(text); matches != nil {
				if currentGuideline != nil {
					part := types.SegmentPart{
						ID:   matches[1],
						Text: strings.TrimSpace(matches[2]),
					}
					currentGuideline.Parts = append(currentGuideline.Parts, part)
				}
				continue
			}
			
			// Accumulate content text
			if block.Type == types.BlockTypeParagraph || block.Type == types.BlockTypeList {
				if currentText.Len() > 0 {
					currentText.WriteString("\n")
				}
				currentText.WriteString(text)
			}
		}
	}
	
	// Finalize last guideline and category
	if currentGuideline != nil {
		if currentText.Len() > 0 {
			s.finalizeGuideline(currentGuideline, currentText.String())
		}
		if currentCategory != nil {
			currentCategory.Guidelines = append(currentCategory.Guidelines, *currentGuideline)
		}
	}
	
	if currentCategory != nil {
		categories = append(categories, *currentCategory)
	}
	
	return categories
}

// finalizeGuideline processes accumulated text for a guideline
func (s *GenericSegmenter) finalizeGuideline(guideline *types.SegmentGuideline, text string) {
	// Extract objective if present
	for _, keyword := range s.rules.ObjectiveKeywords {
		pattern := regexp.MustCompile(fmt.Sprintf(`(?i)%s:\s*([^\n]+)`, keyword))
		if matches := pattern.FindStringSubmatch(text); matches != nil {
			guideline.Objective = strings.TrimSpace(matches[1])
			break
		}
	}
	
	// If no explicit objective, use first sentence
	if guideline.Objective == "" {
		sentences := strings.Split(text, ".")
		if len(sentences) > 0 {
			guideline.Objective = strings.TrimSpace(sentences[0])
		}
	}
	
	// Extract recommendations
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		for _, keyword := range s.rules.RecommendationKeywords {
			if strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
				if len(line) > 0 {
					guideline.Recommendations = append(guideline.Recommendations, line)
				}
				break
			}
		}
	}
}

