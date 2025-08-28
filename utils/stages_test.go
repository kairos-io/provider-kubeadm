package utils

import (
	"testing"

	. "github.com/onsi/gomega"
)

// TestGetFileStage tests the GetFileStage function
func TestGetFileStage(t *testing.T) {
	tests := []struct {
		name            string
		stageName       string
		filePath        string
		content         string
		expectedName    string
		expectedPath    string
		expectedContent string
	}{
		{
			name:            "standard_file_stage",
			stageName:       "Test Stage",
			filePath:        "/test/path/file.txt",
			content:         "test content",
			expectedName:    "Test Stage",
			expectedPath:    "/test/path/file.txt",
			expectedContent: "test content",
		},
		{
			name:            "empty_content",
			stageName:       "Empty Stage",
			filePath:        "/empty/file.txt",
			content:         "",
			expectedName:    "Empty Stage",
			expectedPath:    "/empty/file.txt",
			expectedContent: "",
		},
		{
			name:            "complex_content",
			stageName:       "Complex Stage",
			filePath:        "/complex/config.yaml",
			content:         "apiVersion: v1\nkind: Config\nmetadata:\n  name: test",
			expectedName:    "Complex Stage",
			expectedPath:    "/complex/config.yaml",
			expectedContent: "apiVersion: v1\nkind: Config\nmetadata:\n  name: test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := GetFileStage(tt.stageName, tt.filePath, tt.content)

			// Validate stage structure
			g.Expect(result.Name).To(Equal(tt.expectedName))
			g.Expect(result.Files).To(HaveLen(1))
			g.Expect(result.Files[0].Path).To(Equal(tt.expectedPath))
			g.Expect(result.Files[0].Content).To(Equal(tt.expectedContent))
			g.Expect(result.Files[0].Permissions).To(Equal(uint32(0640)))
		})
	}
}
