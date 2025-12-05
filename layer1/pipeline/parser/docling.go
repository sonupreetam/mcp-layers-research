package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ossf/gemara/layer1/pipeline/types"
)

// DoclingParser uses docling-serve for PDF parsing
type DoclingParser struct {
	ParserBase
}

// NewDoclingParser creates a new Docling parser
func NewDoclingParser(config types.ParserConfig) (*DoclingParser, error) {
	if config.Endpoint == "" {
		config.Endpoint = "http://localhost:5001/api/v1/convert"
	}
	
	parser := &DoclingParser{}
	if err := parser.Configure(config); err != nil {
		return nil, err
	}
	
	return parser, nil
}

// Name returns the parser name
func (p *DoclingParser) Name() string {
	return "docling"
}

// DoclingRequest represents the request to docling-serve
type DoclingRequest struct {
	File    string                 `json:"file,omitempty"`
	Content string                 `json:"content,omitempty"` // base64 encoded content
	Options map[string]interface{} `json:"options,omitempty"`
}

// DoclingResponse represents the response from docling-serve
type DoclingResponse struct {
	Success bool                   `json:"success"`
	Error   string                 `json:"error,omitempty"`
	Pages   []DoclingPage          `json:"pages"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// DoclingPage represents a page from docling
type DoclingPage struct {
	PageNumber int             `json:"page_number"`
	Width      float64         `json:"width"`
	Height     float64         `json:"height"`
	Elements   []DoclingElement `json:"elements"`
}

// DoclingElement represents an element on a page
type DoclingElement struct {
	Type       string                 `json:"type"`
	Text       string                 `json:"text"`
	BBox       []float64              `json:"bbox,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// Parse extracts content from a PDF file using docling-serve
func (p *DoclingParser) Parse(filePath string) (*types.ParsedDocument, error) {
	// Prepare request
	req := DoclingRequest{
		Options: map[string]interface{}{
			"extract_images": false,
			"extract_tables": true,
		},
	}

	// Create multipart form data
	var requestBody bytes.Buffer
	writer := io.MultiWriter(&requestBody)
	
	// For simplicity, we'll send the file path
	// In production, you'd want to send the file content
	reqData, err := json.Marshal(map[string]interface{}{
		"file_path": filePath,
		"options":   req.Options,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	writer.Write(reqData)

	// Make HTTP request to docling-serve
	resp, err := http.Post(p.config.Endpoint, "application/json", &requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to call docling-serve: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var doclingResp DoclingResponse
	if err := json.NewDecoder(resp.Body).Decode(&doclingResp); err != nil {
		return nil, fmt.Errorf("failed to decode docling response: %w", err)
	}

	if !doclingResp.Success {
		return nil, fmt.Errorf("docling parsing failed: %s", doclingResp.Error)
	}

	// Convert to ParsedDocument
	doc := &types.ParsedDocument{
		Metadata: types.ParsedMetadata{
			SourceFile: filePath,
			Parser:     "docling-v1.0",
			ParsedAt:   time.Now(),
		},
		Pages: make([]types.Page, 0, len(doclingResp.Pages)),
	}

	for _, page := range doclingResp.Pages {
		parsedPage := types.Page{
			PageNumber: page.PageNumber,
			Blocks:     make([]types.Block, 0, len(page.Elements)),
		}

		for _, elem := range page.Elements {
			block := p.convertElement(elem)
			parsedPage.Blocks = append(parsedPage.Blocks, block)
		}

		doc.Pages = append(doc.Pages, parsedPage)
	}

	return doc, nil
}

// convertElement converts a DoclingElement to a Block
func (p *DoclingParser) convertElement(elem DoclingElement) types.Block {
	block := types.Block{
		Text: elem.Text,
	}

	// Convert type
	switch elem.Type {
	case "heading", "title":
		block.Type = types.BlockTypeHeading
		if level, ok := elem.Properties["level"].(float64); ok {
			block.Level = int(level)
		}
	case "paragraph", "text":
		block.Type = types.BlockTypeParagraph
	case "list", "list-item":
		block.Type = types.BlockTypeList
		if marker, ok := elem.Properties["marker"].(string); ok {
			block.ListItem = &types.ListItem{
				Marker: marker,
				Type:   "unordered",
			}
		}
	case "table":
		block.Type = types.BlockTypeTable
	case "code":
		block.Type = types.BlockTypeCode
	default:
		block.Type = types.BlockTypeParagraph
	}

	// Convert bbox
	if len(elem.BBox) == 4 {
		block.BBox = &types.BBox{
			X1: elem.BBox[0],
			Y1: elem.BBox[1],
			X2: elem.BBox[2],
			Y2: elem.BBox[3],
		}
	}

	// Extract font properties
	if fontSize, ok := elem.Properties["font_size"].(float64); ok {
		block.FontSize = fontSize
	}
	if fontWeight, ok := elem.Properties["font_weight"].(string); ok {
		block.FontWeight = fontWeight
	}
	if fontName, ok := elem.Properties["font_name"].(string); ok {
		block.FontName = fontName
	}

	return block
}

