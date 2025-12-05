package segmenter

import (
	"regexp"
	"strings"

	"github.com/ossf/gemara/layer1/pipeline/types"
)

// PCIDSSSegmenter applies PCI-DSS specific rules
type PCIDSSSegmenter struct {
	GenericSegmenter
}

// NewPCIDSSSegmenter creates a new PCI-DSS segmenter
func NewPCIDSSSegmenter(config types.SegmenterConfig) (*PCIDSSSegmenter, error) {
	s := &PCIDSSSegmenter{}
	if err := s.Configure(config); err != nil {
		return nil, err
	}
	
	// Initialize PCI-DSS specific rules
	s.rules = &SegmentationRules{
		// PCI-DSS uses patterns like:
		// "Requirement 1: Install and maintain a firewall..."
		// "1.1 Establish firewall and router configuration standards"
		// "1.1.1 A formal process for approving..."
		CategoryPattern:  regexp.MustCompile(`^(?:Requirement\s+)?([0-9]+)[:.]?\s+([A-Z].*)`),
		GuidelinePattern: regexp.MustCompile(`^([0-9]+\.[0-9]+)\s+([A-Z].*)`),
		PartPattern:      regexp.MustCompile(`^([0-9]+\.[0-9]+\.[0-9]+)\s+(.*)`),
		
		TitlePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)payment\s+card\s+industry.*security\s+standard`),
			regexp.MustCompile(`(?i)PCI\s+DSS`),
		},
		VersionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)version\s+([0-9]+\.[0-9]+(?:\.[0-9]+)?)`),
			regexp.MustCompile(`(?i)v\.?\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`),
		},
		AuthorPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)PCI\s+Security\s+Standards\s+Council`),
		},
		
		ObjectiveKeywords: []string{
			"objective", "intent", "purpose",
		},
		RecommendationKeywords: []string{
			"guidance", "examples", "testing procedures",
		},
		RequirementKeywords: []string{
			"requirement", "must", "shall",
		},
		
		CategoryHeadingLevel:  1,
		GuidelineHeadingLevel: 2,
		PartHeadingLevel:      3,
	}
	
	return s, nil
}

// Name returns the segmenter name
func (s *PCIDSSSegmenter) Name() string {
	return "pci-dss-v1.0"
}

// Segment overrides generic segmentation with PCI-DSS specific logic
func (s *PCIDSSSegmenter) Segment(doc *types.ParsedDocument) (*types.SegmentedDocument, error) {
	// Use parent's segmentation
	segmented, err := s.GenericSegmenter.Segment(doc)
	if err != nil {
		return nil, err
	}
	
	// Post-process to add PCI-DSS specific metadata
	segmented.Metadata.Segmenter = s.Name()
	segmented.DocumentMetadata.DocumentType = "Standard"
	segmented.DocumentMetadata.IndustrySectors = []string{"financial-services", "payment-processing"}
	
	// Clean up category descriptions
	for i := range segmented.Categories {
		s.enrichCategory(&segmented.Categories[i])
	}
	
	return segmented, nil
}

// enrichCategory adds PCI-DSS specific category information
func (s *PCIDSSSegmenter) enrichCategory(category *types.SegmentCategory) {
	// Map known PCI-DSS requirements to descriptions
	requirementDescriptions := map[string]string{
		"1":  "Build and Maintain a Secure Network",
		"2":  "Build and Maintain a Secure Network",
		"3":  "Protect Cardholder Data",
		"4":  "Protect Cardholder Data",
		"5":  "Maintain a Vulnerability Management Program",
		"6":  "Maintain a Vulnerability Management Program",
		"7":  "Implement Strong Access Control Measures",
		"8":  "Implement Strong Access Control Measures",
		"9":  "Implement Strong Access Control Measures",
		"10": "Regularly Monitor and Test Networks",
		"11": "Regularly Monitor and Test Networks",
		"12": "Maintain an Information Security Policy",
	}
	
	if desc, ok := requirementDescriptions[category.ID]; ok {
		if category.Description == "" {
			category.Description = desc
		}
	}
	
	// Format category ID as "REQ-X"
	if !strings.HasPrefix(category.ID, "REQ-") {
		category.ID = "REQ-" + category.ID
	}
	
	// Process guidelines
	for i := range category.Guidelines {
		s.enrichGuideline(&category.Guidelines[i], category.ID)
	}
}

// enrichGuideline adds PCI-DSS specific guideline information
func (s *PCIDSSSegmenter) enrichGuideline(guideline *types.SegmentGuideline, categoryID string) {
	// Format guideline ID as "PCI-DSS-X.Y"
	if !strings.HasPrefix(guideline.ID, "PCI-DSS-") {
		guideline.ID = "PCI-DSS-" + guideline.ID
	}
	
	// Process parts
	for i := range guideline.Parts {
		s.enrichPart(&guideline.Parts[i], guideline.ID)
	}
}

// enrichPart adds PCI-DSS specific part information
func (s *PCIDSSSegmenter) enrichPart(part *types.SegmentPart, guidelineID string) {
	// Format part ID as "PCI-DSS-X.Y.Z"
	if !strings.HasPrefix(part.ID, "PCI-DSS-") {
		part.ID = "PCI-DSS-" + part.ID
	}
}

// NIST80053Segmenter applies NIST 800-53 specific rules
type NIST80053Segmenter struct {
	GenericSegmenter
}

// NewNIST80053Segmenter creates a new NIST 800-53 segmenter
func NewNIST80053Segmenter(config types.SegmenterConfig) (*NIST80053Segmenter, error) {
	s := &NIST80053Segmenter{}
	if err := s.Configure(config); err != nil {
		return nil, err
	}
	
	// Initialize NIST 800-53 specific rules
	s.rules = &SegmentationRules{
		// NIST 800-53 uses patterns like:
		// "AC - ACCESS CONTROL"
		// "AC-1 Policy and Procedures"
		// "AC-2(1) Automated System Account Management"
		CategoryPattern:  regexp.MustCompile(`^([A-Z]{2,3})\s*[-–]\s*([A-Z\s]+)`),
		GuidelinePattern: regexp.MustCompile(`^([A-Z]{2,3}-[0-9]+)\s+([A-Z].*)`),
		PartPattern:      regexp.MustCompile(`^([A-Z]{2,3}-[0-9]+)\(([0-9]+)\)\s+(.*)`),
		
		TitlePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)NIST.*800-53`),
			regexp.MustCompile(`(?i)Security\s+and\s+Privacy\s+Controls`),
		},
		VersionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)revision\s+([0-9]+)`),
			regexp.MustCompile(`(?i)rev\.?\s*([0-9]+)`),
		},
		AuthorPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)National\s+Institute\s+of\s+Standards\s+and\s+Technology`),
			regexp.MustCompile(`(?i)NIST`),
		},
		
		ObjectiveKeywords: []string{
			"control", "objective", "purpose",
		},
		RecommendationKeywords: []string{
			"guidance", "supplemental guidance", "discussion",
		},
		RequirementKeywords: []string{
			"control", "requirement",
		},
		
		CategoryHeadingLevel:  1,
		GuidelineHeadingLevel: 2,
		PartHeadingLevel:      3,
	}
	
	return s, nil
}

// Name returns the segmenter name
func (s *NIST80053Segmenter) Name() string {
	return "nist-800-53-v1.0"
}

