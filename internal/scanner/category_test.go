package scanner

import (
	"testing"
)

func TestDefaultCategories(t *testing.T) {
	cats := DefaultCategories()
	if len(cats) == 0 {
		t.Fatal("expected at least one category")
	}

	expectedNames := []string{
		"homebrew", "docker", "xcode", "node", "python",
		"rust", "go", "ruby", "java", "system-caches",
		"system-logs", "trash", "downloads", "app-leftovers",
		"mail", "composer", "cocoapods",
	}

	if len(cats) != len(expectedNames) {
		t.Errorf("expected %d categories, got %d", len(expectedNames), len(cats))
	}

	for i, expected := range expectedNames {
		if i >= len(cats) {
			break
		}
		if cats[i].Name != expected {
			t.Errorf("category %d: expected name %q, got %q", i, expected, cats[i].Name)
		}
	}
}

func TestCategoryNames(t *testing.T) {
	names := CategoryNames()
	if len(names) == 0 {
		t.Fatal("expected at least one category name")
	}

	// Verify first and last.
	if names[0] != "homebrew" {
		t.Errorf("expected first name 'homebrew', got %q", names[0])
	}
	if names[len(names)-1] != "cocoapods" {
		t.Errorf("expected last name 'cocoapods', got %q", names[len(names)-1])
	}
}

func TestFindCategory(t *testing.T) {
	t.Run("existing category", func(t *testing.T) {
		cat, err := FindCategory("homebrew")
		if err != nil {
			t.Fatal(err)
		}
		if cat.Name != "homebrew" {
			t.Errorf("expected name 'homebrew', got %q", cat.Name)
		}
		if cat.DisplayName != "Homebrew" {
			t.Errorf("expected display name 'Homebrew', got %q", cat.DisplayName)
		}
	})

	t.Run("nonexistent category", func(t *testing.T) {
		_, err := FindCategory("nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent category")
		}
	})
}

func TestRiskLevelString(t *testing.T) {
	tests := []struct {
		level    RiskLevel
		expected string
	}{
		{RiskSafe, "safe"},
		{RiskModerate, "moderate"},
		{RiskCaution, "caution"},
		{RiskLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("RiskLevel(%d).String() = %q, want %q", tt.level, got, tt.expected)
			}
		})
	}
}

func TestCategoryPaths(t *testing.T) {
	homeDir := "/Users/testuser"

	tests := []struct {
		name          string
		category      string
		expectPaths   bool
		minPathCount  int
	}{
		{"homebrew has paths", "homebrew", true, 1},
		{"xcode has multiple paths", "xcode", true, 4},
		{"node has paths", "node", true, 3},
		{"python has paths", "python", true, 3},
		{"go has paths", "go", true, 1},
		{"trash has paths", "trash", true, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat, err := FindCategory(tt.category)
			if err != nil {
				t.Fatal(err)
			}
			paths := cat.Paths(homeDir)
			if tt.expectPaths && len(paths) < tt.minPathCount {
				t.Errorf("expected at least %d paths, got %d", tt.minPathCount, len(paths))
			}
		})
	}
}

func TestCategoryFields(t *testing.T) {
	for _, cat := range DefaultCategories() {
		t.Run(cat.Name, func(t *testing.T) {
			if cat.Name == "" {
				t.Error("category name is empty")
			}
			if cat.DisplayName == "" {
				t.Error("category display name is empty")
			}
			if cat.Description == "" {
				t.Error("category description is empty")
			}
			if cat.Paths == nil {
				t.Error("category Paths function is nil")
			}
		})
	}
}

func TestCategoryRiskLevels(t *testing.T) {
	// Verify specific risk levels for important categories.
	safeCategories := []string{"homebrew", "xcode", "node", "python", "rust", "go", "ruby", "system-caches", "system-logs", "trash", "composer", "cocoapods"}
	moderateCategories := []string{"docker", "java", "mail"}
	cautionCategories := []string{"downloads", "app-leftovers"}

	for _, name := range safeCategories {
		cat, err := FindCategory(name)
		if err != nil {
			t.Fatal(err)
		}
		if cat.Risk != RiskSafe {
			t.Errorf("category %q: expected risk Safe, got %s", name, cat.Risk)
		}
	}

	for _, name := range moderateCategories {
		cat, err := FindCategory(name)
		if err != nil {
			t.Fatal(err)
		}
		if cat.Risk != RiskModerate {
			t.Errorf("category %q: expected risk Moderate, got %s", name, cat.Risk)
		}
	}

	for _, name := range cautionCategories {
		cat, err := FindCategory(name)
		if err != nil {
			t.Fatal(err)
		}
		if cat.Risk != RiskCaution {
			t.Errorf("category %q: expected risk Caution, got %s", name, cat.Risk)
		}
	}
}
