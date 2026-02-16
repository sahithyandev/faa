package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple lowercase",
			input:    "myproject",
			expected: "myproject",
		},
		{
			name:     "uppercase to lowercase",
			input:    "MyProject",
			expected: "myproject",
		},
		{
			name:     "with spaces",
			input:    "my project",
			expected: "my-project",
		},
		{
			name:     "with underscores",
			input:    "my_project",
			expected: "my-project",
		},
		{
			name:     "with dots",
			input:    "my.project.name",
			expected: "my-project-name",
		},
		{
			name:     "with multiple special chars",
			input:    "my___project...name",
			expected: "my-project-name",
		},
		{
			name:     "with leading dash",
			input:    "-myproject",
			expected: "myproject",
		},
		{
			name:     "with trailing dash",
			input:    "myproject-",
			expected: "myproject",
		},
		{
			name:     "with leading and trailing dashes",
			input:    "--myproject--",
			expected: "myproject",
		},
		{
			name:     "scoped package",
			input:    "@myorg/myproject",
			expected: "myorg-myproject",
		},
		{
			name:     "complex scoped package",
			input:    "@My-Org/My.Project_Name",
			expected: "my-org-my-project-name",
		},
		{
			name:     "with numbers",
			input:    "project123",
			expected: "project123",
		},
		{
			name:     "only special chars",
			input:    "@@@___...",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFindProjectRoot(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()

	// Create nested directories
	// tmpDir/
	//   project1/
	//     package.json (name: "test-project")
	//     src/
	//       nested/
	//   project2/
	//     package.json (name: "@myorg/another-project")

	project1Dir := filepath.Join(tmpDir, "project1")
	project1SrcDir := filepath.Join(project1Dir, "src")
	project1NestedDir := filepath.Join(project1SrcDir, "nested")

	project2Dir := filepath.Join(tmpDir, "project2")

	// Create directories
	if err := os.MkdirAll(project1NestedDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	if err := os.MkdirAll(project2Dir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Create package.json files
	pkg1 := PackageJSON{Name: "test-project"}
	pkg1Data, _ := json.Marshal(pkg1)
	if err := os.WriteFile(filepath.Join(project1Dir, "package.json"), pkg1Data, 0644); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}

	pkg2 := PackageJSON{Name: "@myorg/another-project"}
	pkg2Data, _ := json.Marshal(pkg2)
	if err := os.WriteFile(filepath.Join(project2Dir, "package.json"), pkg2Data, 0644); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}

	tests := []struct {
		name         string
		startDir     string
		expectedRoot string
		expectedName string
		expectError  bool
	}{
		{
			name:         "from project root",
			startDir:     project1Dir,
			expectedRoot: project1Dir,
			expectedName: "test-project",
			expectError:  false,
		},
		{
			name:         "from nested directory",
			startDir:     project1NestedDir,
			expectedRoot: project1Dir,
			expectedName: "test-project",
			expectError:  false,
		},
		{
			name:         "from src directory",
			startDir:     project1SrcDir,
			expectedRoot: project1Dir,
			expectedName: "test-project",
			expectError:  false,
		},
		{
			name:         "scoped package",
			startDir:     project2Dir,
			expectedRoot: project2Dir,
			expectedName: "myorg-another-project",
			expectError:  false,
		},
		{
			name:        "no package.json",
			startDir:    tmpDir,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := FindProjectRoot(tt.startDir)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if project.Root != tt.expectedRoot {
				t.Errorf("Root = %q, want %q", project.Root, tt.expectedRoot)
			}

			if project.Name != tt.expectedName {
				t.Errorf("Name = %q, want %q", project.Name, tt.expectedName)
			}
		})
	}
}

func TestProjectHost(t *testing.T) {
	tests := []struct {
		name         string
		projectName  string
		expectedHost string
	}{
		{
			name:         "simple name",
			projectName:  "myproject",
			expectedHost: "myproject",
		},
		{
			name:         "normalized name",
			projectName:  "my-project",
			expectedHost: "my-project",
		},
		{
			name:         "with numbers",
			projectName:  "project-123",
			expectedHost: "project-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := &Project{
				Root: "/some/path",
				Name: tt.projectName,
			}

			host := project.Host()
			if host != tt.expectedHost {
				t.Errorf("Host() = %q, want %q", host, tt.expectedHost)
			}
		})
	}
}

func TestFindProjectRootWithInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a package.json with invalid JSON
	invalidJSON := []byte(`{"name": "test`) // incomplete JSON
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), invalidJSON, 0644); err != nil {
		t.Fatalf("failed to create invalid package.json: %v", err)
	}

	_, err := FindProjectRoot(tmpDir)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
