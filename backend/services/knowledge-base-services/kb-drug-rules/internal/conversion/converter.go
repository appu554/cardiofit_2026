// Package conversion provides format conversion utilities for drug rules
// Supports TOML <-> JSON conversion with validation
package conversion

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/BurntSushi/toml"
)

// FormatConverter handles conversion between TOML and JSON formats
type FormatConverter struct {
	strictMode bool
}

// NewFormatConverter creates a new format converter
func NewFormatConverter() *FormatConverter {
	return &FormatConverter{
		strictMode: true,
	}
}

// ConversionResult contains the result of a format conversion
type ConversionResult struct {
	Success      bool              `json:"success"`
	Output       string            `json:"output,omitempty"`
	OutputFormat string            `json:"output_format,omitempty"`
	Errors       []string          `json:"errors,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// TOMLToJSON converts TOML content to JSON
func (fc *FormatConverter) TOMLToJSON(tomlContent string) (*ConversionResult, error) {
	result := &ConversionResult{
		OutputFormat: "json",
		Metadata:     make(map[string]string),
	}

	// Parse TOML
	var data interface{}
	if _, err := toml.Decode(tomlContent, &data); err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("TOML parse error: %v", err))
		return result, err
	}

	// Convert to JSON
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("JSON encoding error: %v", err))
		return result, err
	}

	result.Success = true
	result.Output = string(jsonBytes)
	result.Metadata["input_format"] = "toml"
	result.Metadata["output_format"] = "json"

	return result, nil
}

// JSONToTOML converts JSON content to TOML
func (fc *FormatConverter) JSONToTOML(jsonContent string) (*ConversionResult, error) {
	result := &ConversionResult{
		OutputFormat: "toml",
		Metadata:     make(map[string]string),
	}

	// Parse JSON
	var data interface{}
	if err := json.Unmarshal([]byte(jsonContent), &data); err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("JSON parse error: %v", err))
		return result, err
	}

	// Convert to TOML
	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(data); err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("TOML encoding error: %v", err))
		return result, err
	}

	result.Success = true
	result.Output = buf.String()
	result.Metadata["input_format"] = "json"
	result.Metadata["output_format"] = "toml"

	return result, nil
}

// Convert performs format conversion based on source and target formats
func (fc *FormatConverter) Convert(content, sourceFormat, targetFormat string) (*ConversionResult, error) {
	if sourceFormat == targetFormat {
		return &ConversionResult{
			Success:      true,
			Output:       content,
			OutputFormat: targetFormat,
			Warnings:     []string{"Source and target formats are the same, no conversion performed"},
		}, nil
	}

	switch {
	case sourceFormat == "toml" && targetFormat == "json":
		return fc.TOMLToJSON(content)
	case sourceFormat == "json" && targetFormat == "toml":
		return fc.JSONToTOML(content)
	default:
		return &ConversionResult{
			Success: false,
			Errors:  []string{fmt.Sprintf("Unsupported conversion: %s to %s", sourceFormat, targetFormat)},
		}, fmt.Errorf("unsupported conversion: %s to %s", sourceFormat, targetFormat)
	}
}

// ValidateJSON checks if content is valid JSON
func (fc *FormatConverter) ValidateJSON(content string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(content), &js) == nil
}

// ValidateTOML checks if content is valid TOML
func (fc *FormatConverter) ValidateTOML(content string) bool {
	var data interface{}
	_, err := toml.Decode(content, &data)
	return err == nil
}

// DetectFormat attempts to detect whether content is TOML or JSON
func (fc *FormatConverter) DetectFormat(content string) string {
	// Try JSON first (stricter format)
	if fc.ValidateJSON(content) {
		return "json"
	}
	// Try TOML
	if fc.ValidateTOML(content) {
		return "toml"
	}
	return "unknown"
}
