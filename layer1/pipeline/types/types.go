package types

import "time"

// ParsedDocument represents the raw output from PDF parsing
type ParsedDocument struct {
	Metadata ParsedMetadata `json:"metadata" yaml:"metadata"`
	Pages    []Page         `json:"pages" yaml:"pages"`
}

// ParsedMetadata contains information about the parsing process
type ParsedMetadata struct {
	SourceFile string    `json:"source_file" yaml:"source_file"`
	Parser     string    `json:"parser" yaml:"parser"`
	ParsedAt   time.Time `json:"parsed_at" yaml:"parsed_at"`
	Version    int       `json:"version" yaml:"version"`
	DocumentID string    `json:"document_id" yaml:"document_id"`
}

// Page represents a single page from the PDF
type Page struct {
	PageNumber int     `json:"page_number" yaml:"page_number"`
	Blocks     []Block `json:"blocks" yaml:"blocks"`
}

// Block represents a text block with formatting information
type Block struct {
	Type       BlockType  `json:"type" yaml:"type"`
	Level      int        `json:"level,omitempty" yaml:"level,omitempty"` // For headings
	Text       string     `json:"text" yaml:"text"`
	BBox       *BBox      `json:"bbox,omitempty" yaml:"bbox,omitempty"`
	FontSize   float64    `json:"font_size,omitempty" yaml:"font_size,omitempty"`
	FontWeight string     `json:"font_weight,omitempty" yaml:"font_weight,omitempty"`
	FontName   string     `json:"font_name,omitempty" yaml:"font_name,omitempty"`
	ListItem   *ListItem  `json:"list_item,omitempty" yaml:"list_item,omitempty"`
	TableData  *TableData `json:"table_data,omitempty" yaml:"table_data,omitempty"`
}

// BlockType represents the type of content block
type BlockType string

const (
	BlockTypeHeading   BlockType = "heading"
	BlockTypeParagraph BlockType = "paragraph"
	BlockTypeList      BlockType = "list"
	BlockTypeTable     BlockType = "table"
	BlockTypeCode      BlockType = "code"
	BlockTypeFootnote  BlockType = "footnote"
	BlockTypeCaption   BlockType = "caption"
)

// BBox represents a bounding box
type BBox struct {
	X1 float64 `json:"x1" yaml:"x1"`
	Y1 float64 `json:"y1" yaml:"y1"`
	X2 float64 `json:"x2" yaml:"x2"`
	Y2 float64 `json:"y2" yaml:"y2"`
}

// ListItem contains list-specific information
type ListItem struct {
	Level  int    `json:"level" yaml:"level"`
	Marker string `json:"marker" yaml:"marker"` // e.g., "1.", "a.", "•"
	Type   string `json:"type" yaml:"type"`     // "ordered" or "unordered"
}

// TableData contains table-specific information
type TableData struct {
	Rows [][]string `json:"rows" yaml:"rows"`
}

// SegmentedDocument represents the document after rule-based segmentation
type SegmentedDocument struct {
	Metadata         SegmentedMetadata `json:"metadata" yaml:"metadata"`
	DocumentMetadata DocumentMetadata  `json:"document_metadata" yaml:"document_metadata"`
	FrontMatter      string            `json:"front_matter,omitempty" yaml:"front_matter,omitempty"`
	Categories       []SegmentCategory `json:"categories" yaml:"categories"`
}

// SegmentedMetadata contains information about the segmentation process
type SegmentedMetadata struct {
	SourceVersion int       `json:"source_version" yaml:"source_version"`
	Segmenter     string    `json:"segmenter" yaml:"segmenter"`
	SegmentedAt   time.Time `json:"segmented_at" yaml:"segmented_at"`
	Version       int       `json:"version" yaml:"version"`
	DocumentID    string    `json:"document_id" yaml:"document_id"`
}

// DocumentMetadata contains extracted document metadata
type DocumentMetadata struct {
	ID              string   `json:"id" yaml:"id"`
	Title           string   `json:"title" yaml:"title"`
	Description     string   `json:"description" yaml:"description"`
	Author          string   `json:"author" yaml:"author"`
	Version         string   `json:"version" yaml:"version"`
	PublicationDate string   `json:"publication_date,omitempty" yaml:"publication_date,omitempty"`
	DocumentType    string   `json:"document_type,omitempty" yaml:"document_type,omitempty"`
	Jurisdictions   []string `json:"jurisdictions,omitempty" yaml:"jurisdictions,omitempty"`
	IndustrySectors []string `json:"industry_sectors,omitempty" yaml:"industry_sectors,omitempty"`
}

// SegmentCategory represents a category with its guidelines
type SegmentCategory struct {
	ID          string             `json:"id" yaml:"id"`
	Title       string             `json:"title" yaml:"title"`
	Description string             `json:"description" yaml:"description"`
	Guidelines  []SegmentGuideline `json:"guidelines,omitempty" yaml:"guidelines,omitempty"`
}

// SegmentGuideline represents a guideline with its parts
type SegmentGuideline struct {
	ID              string        `json:"id" yaml:"id"`
	Title           string        `json:"title" yaml:"title"`
	Objective       string        `json:"objective,omitempty" yaml:"objective,omitempty"`
	Recommendations []string      `json:"recommendations,omitempty" yaml:"recommendations,omitempty"`
	Parts           []SegmentPart `json:"parts,omitempty" yaml:"parts,omitempty"`
}

// SegmentPart represents a part of a guideline
type SegmentPart struct {
	ID              string   `json:"id" yaml:"id"`
	Title           string   `json:"title,omitempty" yaml:"title,omitempty"`
	Text            string   `json:"text" yaml:"text"`
	Recommendations []string `json:"recommendations,omitempty" yaml:"recommendations,omitempty"`
}

// ParserConfig contains configuration for the PDF parser
type ParserConfig struct {
	Provider      string            `json:"provider" yaml:"provider"` // "docling", "pymupdf", etc.
	Endpoint      string            `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Options       map[string]string `json:"options,omitempty" yaml:"options,omitempty"`
	TempDir       string            `json:"temp_dir" yaml:"temp_dir"`
	KeepTempFiles bool              `json:"keep_temp_files" yaml:"keep_temp_files"`
}

// SegmenterConfig contains configuration for the segmenter
type SegmenterConfig struct {
	RulesFile    string            `json:"rules_file" yaml:"rules_file"`
	DocumentType string            `json:"document_type" yaml:"document_type"` // "pci-dss", "nist-800-53", etc.
	Options      map[string]string `json:"options,omitempty" yaml:"options,omitempty"`
}

// LLMConfig contains configuration for LLM enhancement
type LLMConfig struct {
	Provider   string            `json:"provider" yaml:"provider"` // "openai", "anthropic", etc.
	Model      string            `json:"model" yaml:"model"`
	APIKey     string            `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	Endpoint   string            `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	MaxTokens  int               `json:"max_tokens" yaml:"max_tokens"`
	Options    map[string]string `json:"options,omitempty" yaml:"options,omitempty"`
	Temperature float64          `json:"temperature" yaml:"temperature"`
}

// EnhancementResult contains the result of LLM enhancement
type EnhancementResult struct {
	OriginalData interface{}       `json:"original_data" yaml:"original_data"`
	EnhancedData interface{}       `json:"enhanced_data" yaml:"enhanced_data"`
	Changes      []EnhancementChange `json:"changes" yaml:"changes"`
	Confidence   float64           `json:"confidence" yaml:"confidence"`
	Provider     string            `json:"provider" yaml:"provider"`
	Model        string            `json:"model" yaml:"model"`
	Timestamp    time.Time         `json:"timestamp" yaml:"timestamp"`
}

// EnhancementChange describes a change made by LLM enhancement
type EnhancementChange struct {
	Path        string `json:"path" yaml:"path"`         // JSON path to the changed field
	Type        string `json:"type" yaml:"type"`         // "add", "modify", "remove"
	OldValue    string `json:"old_value,omitempty" yaml:"old_value,omitempty"`
	NewValue    string `json:"new_value" yaml:"new_value"`
	Reason      string `json:"reason" yaml:"reason"`
	Confidence  float64 `json:"confidence" yaml:"confidence"`
}

