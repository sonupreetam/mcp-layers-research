package validator

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ossf/gemara/layer1"
	"gopkg.in/yaml.v3"
)

// ValidationError represents a schema validation error
type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Value   any    `json:"value,omitempty"`
}

func (e ValidationError) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("%s: %s (got: %v)", e.Path, e.Message, e.Value)
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// ValidationResult contains all validation errors
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

func (r *ValidationResult) Error() string {
	if r.Valid {
		return ""
	}
	var msgs []string
	for _, e := range r.Errors {
		msgs = append(msgs, e.Error())
	}
	return fmt.Sprintf("validation failed with %d errors:\n  - %s", len(r.Errors), strings.Join(msgs, "\n  - "))
}

func (r *ValidationResult) AddError(path, message string, value any) {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{
		Path:    path,
		Message: message,
		Value:   value,
	})
}

// ValidDocumentTypes are the allowed document types per CUE schema
var ValidDocumentTypes = map[layer1.DocumentType]bool{
	"Standard":      true,
	"Regulation":    true,
	"Best Practice": true,
	"Framework":     true,
}

// Validator provides Layer-1 schema validation
type Validator struct {
	strict bool // If true, treat warnings as errors
}

// Option is a functional option for configuring the validator
type Option func(*Validator)

// WithStrictMode enables or disables strict validation
func WithStrictMode(strict bool) Option {
	return func(v *Validator) {
		v.strict = strict
	}
}

// NewValidator creates a new schema validator with optional configuration
func NewValidator(opts ...Option) *Validator {
	v := &Validator{strict: false}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// Validate performs full schema validation on a GuidanceDocument
func (v *Validator) Validate(doc *layer1.GuidanceDocument) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if doc == nil {
		result.AddError("", "document is nil", nil)
		return result
	}

	// Validate metadata (required)
	v.validateMetadata(&doc.Metadata, result)

	// Validate categories
	v.validateCategories(doc.Categories, result)

	// Validate imported guidelines mappings
	for i, mapping := range doc.ImportedGuidelines {
		v.validateMapping(&mapping, fmt.Sprintf("imported-guidelines[%d]", i), result)
	}

	// Validate imported principles mappings
	for i, mapping := range doc.ImportedPrinciples {
		v.validateMapping(&mapping, fmt.Sprintf("imported-principles[%d]", i), result)
	}

	return result
}

// validateMetadata validates the Metadata structure
func (v *Validator) validateMetadata(meta *layer1.Metadata, result *ValidationResult) {
	// Required fields per CUE schema
	if meta.Id == "" {
		result.AddError("metadata.id", "required field is empty", nil)
	}

	if meta.Title == "" {
		result.AddError("metadata.title", "required field is empty", nil)
	}

	if meta.Description == "" {
		result.AddError("metadata.description", "required field is empty", nil)
	}

	if meta.Author == "" {
		result.AddError("metadata.author", "required field is empty", nil)
	}

	// DocumentType validation - must be one of the allowed enum values
	if meta.DocumentType != "" {
		if !ValidDocumentTypes[meta.DocumentType] {
			result.AddError("metadata.document-type", 
				"must be one of: Standard, Regulation, Best Practice, Framework",
				meta.DocumentType)
		}
	} else if v.strict {
		// In strict mode, document-type is required
		result.AddError("metadata.document-type",
			"required field is empty (strict mode)",
			nil)
	}

	// Validate nested applicability if present
	if meta.Applicability != nil {
		v.validateApplicability(meta.Applicability, result)
	}

	// Validate mapping references if present
	for i, ref := range meta.MappingReferences {
		v.validateMappingReference(&ref, fmt.Sprintf("metadata.mapping-references[%d]", i), result)
	}
}

// validateApplicability validates the Applicability structure
func (v *Validator) validateApplicability(app *layer1.Applicability, result *ValidationResult) {
	// Applicability fields are all optional, but if present should have content
	// This is a soft validation - we don't fail on empty arrays
}

// validateMappingReference validates a MappingReference structure
func (v *Validator) validateMappingReference(ref *layer1.MappingReference, path string, result *ValidationResult) {
	if ref.Id == "" {
		result.AddError(path+".id", "required field is empty", nil)
	}
	if ref.Title == "" {
		result.AddError(path+".title", "required field is empty", nil)
	}
	if ref.Version == "" {
		result.AddError(path+".version", "required field is empty", nil)
	}
}

// validateCategories validates all categories
func (v *Validator) validateCategories(categories []layer1.Category, result *ValidationResult) {
	if len(categories) == 0 {
		result.AddError("categories", "document must have at least one category", nil)
		return
	}

	seenCategoryIDs := make(map[string]bool)
	for i, cat := range categories {
		path := fmt.Sprintf("categories[%d]", i)

		// Check for duplicate category IDs
		if cat.Id != "" {
			if seenCategoryIDs[cat.Id] {
				result.AddError(path+".id", "duplicate category ID", cat.Id)
			}
			seenCategoryIDs[cat.Id] = true
		}

		v.validateCategory(&cat, path, result)
	}
}

// validateCategory validates a single Category
func (v *Validator) validateCategory(cat *layer1.Category, path string, result *ValidationResult) {
	if cat.Id == "" {
		result.AddError(path+".id", "required field is empty", nil)
	}
	if cat.Title == "" {
		result.AddError(path+".title", "required field is empty", nil)
	}
	if cat.Description == "" {
		result.AddError(path+".description", "required field is empty", nil)
	}

	// Validate guidelines
	seenGuidelineIDs := make(map[string]bool)
	for i, guide := range cat.Guidelines {
		guidePath := fmt.Sprintf("%s.guidelines[%d]", path, i)

		// Check for duplicate guideline IDs within category
		if guide.Id != "" {
			if seenGuidelineIDs[guide.Id] {
				result.AddError(guidePath+".id", "duplicate guideline ID within category", guide.Id)
			}
			seenGuidelineIDs[guide.Id] = true
		}

		v.validateGuideline(&guide, guidePath, result)
	}
}

// validateGuideline validates a single Guideline
func (v *Validator) validateGuideline(guide *layer1.Guideline, path string, result *ValidationResult) {
	if guide.Id == "" {
		result.AddError(path+".id", "required field is empty", nil)
	}
	if guide.Title == "" {
		result.AddError(path+".title", "required field is empty", nil)
	}

	// Validate rationale if present
	if guide.Rationale != nil {
		v.validateRationale(guide.Rationale, path+".rationale", result)
	}

	// Validate guideline parts
	seenPartIDs := make(map[string]bool)
	for i, part := range guide.GuidelineParts {
		partPath := fmt.Sprintf("%s.guideline-parts[%d]", path, i)

		if part.Id != "" {
			if seenPartIDs[part.Id] {
				result.AddError(partPath+".id", "duplicate part ID within guideline", part.Id)
			}
			seenPartIDs[part.Id] = true
		}

		v.validatePart(&part, partPath, result)
	}

	// Validate guideline mappings
	for i, mapping := range guide.GuidelineMappings {
		v.validateMapping(&mapping, fmt.Sprintf("%s.guideline-mappings[%d]", path, i), result)
	}

	// Validate principle mappings
	for i, mapping := range guide.PrincipleMappings {
		v.validateMapping(&mapping, fmt.Sprintf("%s.principle-mappings[%d]", path, i), result)
	}
}

// validateRationale validates a Rationale structure
func (v *Validator) validateRationale(rat *layer1.Rationale, path string, result *ValidationResult) {
	// Per CUE schema, risks and outcomes are required arrays within Rationale
	for i, risk := range rat.Risks {
		riskPath := fmt.Sprintf("%s.risks[%d]", path, i)
		if risk.Title == "" {
			result.AddError(riskPath+".title", "required field is empty", nil)
		}
		if risk.Description == "" {
			result.AddError(riskPath+".description", "required field is empty", nil)
		}
	}

	for i, outcome := range rat.Outcomes {
		outcomePath := fmt.Sprintf("%s.outcomes[%d]", path, i)
		if outcome.Title == "" {
			result.AddError(outcomePath+".title", "required field is empty", nil)
		}
		if outcome.Description == "" {
			result.AddError(outcomePath+".description", "required field is empty", nil)
		}
	}
}

// validatePart validates a guideline Part
func (v *Validator) validatePart(part *layer1.Part, path string, result *ValidationResult) {
	if part.Id == "" {
		result.AddError(path+".id", "required field is empty", nil)
	}
	if part.Text == "" {
		result.AddError(path+".text", "required field is empty", nil)
	}
}

// validateMapping validates a Mapping structure
func (v *Validator) validateMapping(mapping *layer1.Mapping, path string, result *ValidationResult) {
	if mapping.ReferenceId == "" {
		result.AddError(path+".reference-id", "required field is empty", nil)
	}

	for i, entry := range mapping.Entries {
		entryPath := fmt.Sprintf("%s.entries[%d]", path, i)
		if entry.ReferenceId == "" {
			result.AddError(entryPath+".reference-id", "required field is empty", nil)
		}
		// Strength should be validated - typically 0-100 or similar range
		if entry.Strength < 0 || entry.Strength > 100 {
			result.AddError(entryPath+".strength", "should be between 0 and 100", entry.Strength)
		}
	}
}

// ValidateLayer1 is a convenience function that returns an error if validation fails
func ValidateLayer1(doc *layer1.GuidanceDocument) error {
	v := NewValidator(WithStrictMode(true))
	result := v.Validate(doc)
	if !result.Valid {
		return result
	}
	return nil
}

// ValidateLayer1Lenient validates but allows some optional fields to be empty
func ValidateLayer1Lenient(doc *layer1.GuidanceDocument) error {
	v := NewValidator()
	result := v.Validate(doc)
	if !result.Valid {
		return result
	}
	return nil
}

// QuickValidate provides a simple pass/fail validation
func QuickValidate(doc *layer1.GuidanceDocument) error {
	v := NewValidator()
	result := v.Validate(doc)
	if !result.Valid {
		return result
	}
	return nil
}

// ValidateJSON validates a JSON byte slice as a Layer-1 document
func (v *Validator) ValidateJSON(data []byte) (*ValidationResult, error) {
	var doc layer1.GuidanceDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		result := &ValidationResult{Valid: false}
		result.AddError("", fmt.Sprintf("invalid JSON: %v", err), nil)
		return result, nil
	}
	return v.Validate(&doc), nil
}

// ValidateYAML validates a YAML byte slice as a Layer-1 document
func (v *Validator) ValidateYAML(data []byte) (*ValidationResult, error) {
	var doc layer1.GuidanceDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		result := &ValidationResult{Valid: false}
		result.AddError("", fmt.Sprintf("invalid YAML: %v", err), nil)
		return result, nil
	}
	return v.Validate(&doc), nil
}

// ValidateFile validates a Layer-1 document from a file path
func ValidateFile(path string, strict bool) (*ValidationResult, error) {
	var v *Validator
	if strict {
		v = NewValidator(WithStrictMode(true))
	} else {
		v = NewValidator()
	}
	
	// Read file and determine format
	data, err := readFileBytes(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	// Try JSON first, then YAML
	if isJSON(data) {
		return v.ValidateJSON(data)
	}
	return v.ValidateYAML(data)
}

// readFileBytes reads file content (extracted for testability)
func readFileBytes(path string) ([]byte, error) {
	// Import os package would be needed
	return nil, fmt.Errorf("file reading not implemented - use ValidateJSON or ValidateYAML directly")
}

// isJSON checks if data looks like JSON
func isJSON(data []byte) bool {
	return len(data) > 0 && (data[0] == '{' || data[0] == '[')
}
