package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/ossf/gemara/layer1/pipeline/types"
)

// Storage manages versioned intermediate and final outputs
type Storage struct {
	baseDir string
}

// NewStorage creates a new Storage instance
func NewStorage(baseDir string) (*Storage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}
	return &Storage{baseDir: baseDir}, nil
}

// StorageMetadata tracks version and storage information
type StorageMetadata struct {
	DocumentID  string    `json:"document_id" yaml:"document_id"`
	Version     int       `json:"version" yaml:"version"`
	Type        string    `json:"type" yaml:"type"` // "parsed", "segmented", "final"
	StoredAt    time.Time `json:"stored_at" yaml:"stored_at"`
	Size        int64     `json:"size" yaml:"size"`
	Checksum    string    `json:"checksum,omitempty" yaml:"checksum,omitempty"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
}

// SaveParsed saves parsed document with versioning
func (s *Storage) SaveParsed(doc *types.ParsedDocument) error {
	version := s.getNextVersion(doc.Metadata.DocumentID, "parsed")
	doc.Metadata.Version = version

	dir := filepath.Join(s.baseDir, "intermediate", doc.Metadata.DocumentID, fmt.Sprintf("v%d", version))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	// Save parsed document
	filePath := filepath.Join(dir, "parsed.json")
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal parsed document: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write parsed document: %w", err)
	}

	// Save metadata
	meta := StorageMetadata{
		DocumentID: doc.Metadata.DocumentID,
		Version:    version,
		Type:       "parsed",
		StoredAt:   time.Now(),
		Size:       int64(len(data)),
	}
	return s.saveMetadataWithType(dir, meta, "parsed")
}

// LoadParsed loads a parsed document by version (0 = latest)
func (s *Storage) LoadParsed(documentID string, version int) (*types.ParsedDocument, error) {
	if version == 0 {
		version = s.getLatestVersion(documentID, "parsed")
	}

	filePath := filepath.Join(s.baseDir, "intermediate", documentID, fmt.Sprintf("v%d", version), "parsed.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read parsed document: %w", err)
	}

	var doc types.ParsedDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parsed document: %w", err)
	}

	return &doc, nil
}

// SaveSegmented saves segmented document with versioning
func (s *Storage) SaveSegmented(doc *types.SegmentedDocument) error {
	version := s.getNextVersion(doc.Metadata.DocumentID, "segmented")
	doc.Metadata.Version = version

	dir := filepath.Join(s.baseDir, "intermediate", doc.Metadata.DocumentID, fmt.Sprintf("v%d", version))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	// Save segmented document
	filePath := filepath.Join(dir, "segmented.json")
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal segmented document: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write segmented document: %w", err)
	}

	// Save metadata
	meta := StorageMetadata{
		DocumentID: doc.Metadata.DocumentID,
		Version:    version,
		Type:       "segmented",
		StoredAt:   time.Now(),
		Size:       int64(len(data)),
	}
	return s.saveMetadataWithType(dir, meta, "segmented")
}

// LoadSegmented loads a segmented document by version (0 = latest)
func (s *Storage) LoadSegmented(documentID string, version int) (*types.SegmentedDocument, error) {
	if version == 0 {
		version = s.getLatestVersion(documentID, "segmented")
	}

	filePath := filepath.Join(s.baseDir, "intermediate", documentID, fmt.Sprintf("v%d", version), "segmented.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read segmented document: %w", err)
	}

	var doc types.SegmentedDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal segmented document: %w", err)
	}

	return &doc, nil
}

// SaveFinal saves the final Layer-1 document
func (s *Storage) SaveFinal(documentID string, data interface{}, format string) error {
	dir := filepath.Join(s.baseDir, "final")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create final directory: %w", err)
	}

	var fileName string
	var fileData []byte
	var err error

	switch format {
	case "yaml", "yml":
		fileName = fmt.Sprintf("%s.yaml", documentID)
		fileData, err = yaml.Marshal(data)
	case "json":
		fileName = fmt.Sprintf("%s.json", documentID)
		fileData, err = json.MarshalIndent(data, "", "  ")
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal final document: %w", err)
	}

	filePath := filepath.Join(dir, fileName)
	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		return fmt.Errorf("failed to write final document: %w", err)
	}

	return nil
}

// ListVersions lists all versions for a document and type
func (s *Storage) ListVersions(documentID, docType string) ([]StorageMetadata, error) {
	dir := filepath.Join(s.baseDir, "intermediate", documentID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []StorageMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var metas []StorageMetadata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Try to read type-specific metadata first, fallback to generic
		var metaPath string
		if docType != "" {
			metaPath = filepath.Join(dir, entry.Name(), fmt.Sprintf("metadata-%s.json", docType))
		} else {
			metaPath = filepath.Join(dir, entry.Name(), "metadata.json")
		}
		
		data, err := os.ReadFile(metaPath)
		if err != nil {
			// Fallback to generic metadata.json
			if docType != "" {
				metaPath = filepath.Join(dir, entry.Name(), "metadata.json")
				data, err = os.ReadFile(metaPath)
			}
			if err != nil {
				continue
			}
		}

		var meta StorageMetadata
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}

		if docType == "" || meta.Type == docType {
			metas = append(metas, meta)
		}
	}

	// Sort by version descending
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].Version > metas[j].Version
	})

	return metas, nil
}

// getNextVersion determines the next version number
func (s *Storage) getNextVersion(documentID, docType string) int {
	latest := s.getLatestVersion(documentID, docType)
	return latest + 1
}

// getLatestVersion gets the latest version number
func (s *Storage) getLatestVersion(documentID, docType string) int {
	versions, err := s.ListVersions(documentID, docType)
	if err != nil || len(versions) == 0 {
		return 0
	}
	return versions[0].Version
}

// saveMetadataWithType saves metadata for a version with type-specific filename
func (s *Storage) saveMetadataWithType(dir string, meta StorageMetadata, docType string) error {
	metaPath := filepath.Join(dir, fmt.Sprintf("metadata-%s.json", docType))
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metaPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// saveMetadata saves metadata for a version
func (s *Storage) saveMetadata(dir string, meta StorageMetadata) error {
	metaPath := filepath.Join(dir, "metadata.json")
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metaPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// GetBaseDir returns the base directory
func (s *Storage) GetBaseDir() string {
	return s.baseDir
}

