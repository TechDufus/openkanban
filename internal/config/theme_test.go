package config

import (
	"regexp"
	"testing"
)

func TestThemeNames(t *testing.T) {
	names := ThemeNames()

	if len(names) != 20 {
		t.Errorf("ThemeNames() returned %d themes; want 20", len(names))
	}

	for _, name := range names {
		if _, exists := BuiltinThemes[name]; !exists {
			t.Errorf("ThemeNames() includes %q but it's not in BuiltinThemes", name)
		}
	}

	if names[0] != "catppuccin-mocha" {
		t.Errorf("ThemeNames()[0] = %q; want %q", names[0], "catppuccin-mocha")
	}
}

func TestThemeNames_AllBuiltinThemesIncluded(t *testing.T) {
	names := ThemeNames()
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	for name := range BuiltinThemes {
		if !nameSet[name] {
			t.Errorf("BuiltinThemes contains %q but ThemeNames() doesn't include it", name)
		}
	}
}

func TestGetTheme_ReturnsCorrectTheme(t *testing.T) {
	tests := []struct {
		name         string
		expectedName string
	}{
		{"catppuccin-mocha", "Catppuccin Mocha"},
		{"dracula", "Dracula"},
		{"nord", "Nord"},
		{"gruvbox-dark", "Gruvbox Dark"},
		{"tokyo-night", "Tokyo Night"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme := GetTheme(tt.name, nil)
			if theme.Name != tt.expectedName {
				t.Errorf("GetTheme(%q).Name = %q; want %q", tt.name, theme.Name, tt.expectedName)
			}
		})
	}
}

func TestGetTheme_FallbackToDefault(t *testing.T) {
	theme := GetTheme("nonexistent-theme", nil)

	if theme.Name != "Catppuccin Mocha" {
		t.Errorf("GetTheme(\"nonexistent-theme\").Name = %q; want %q", theme.Name, "Catppuccin Mocha")
	}

	expected := BuiltinThemes["catppuccin-mocha"]
	if theme.Colors.Base != expected.Colors.Base {
		t.Errorf("Fallback theme Base = %q; want %q", theme.Colors.Base, expected.Colors.Base)
	}
}

func TestGetTheme_CustomColorOverrides(t *testing.T) {
	customColors := &ThemeColors{
		Primary: "#ff0000",
		Success: "#00ff00",
	}

	theme := GetTheme("catppuccin-mocha", customColors)

	if theme.Colors.Primary != "#ff0000" {
		t.Errorf("Custom Primary = %q; want %q", theme.Colors.Primary, "#ff0000")
	}
	if theme.Colors.Success != "#00ff00" {
		t.Errorf("Custom Success = %q; want %q", theme.Colors.Success, "#00ff00")
	}

	base := BuiltinThemes["catppuccin-mocha"]
	if theme.Colors.Base != base.Colors.Base {
		t.Errorf("Base color should not change; got %q, want %q", theme.Colors.Base, base.Colors.Base)
	}
	if theme.Colors.Error != base.Colors.Error {
		t.Errorf("Error color should not change; got %q, want %q", theme.Colors.Error, base.Colors.Error)
	}
}

func TestGetTheme_EmptyCustomColorsIgnored(t *testing.T) {
	customColors := &ThemeColors{
		Primary: "",
		Success: "#00ff00",
	}

	theme := GetTheme("catppuccin-mocha", customColors)
	base := BuiltinThemes["catppuccin-mocha"]

	if theme.Colors.Primary != base.Colors.Primary {
		t.Errorf("Empty custom color should not override; Primary = %q, want %q", theme.Colors.Primary, base.Colors.Primary)
	}

	if theme.Colors.Success != "#00ff00" {
		t.Errorf("Non-empty custom color should override; Success = %q, want %q", theme.Colors.Success, "#00ff00")
	}
}

func TestGetTheme_AllCustomColorsCanBeOverridden(t *testing.T) {
	customColors := &ThemeColors{
		Base:      "#000001",
		Surface:   "#000002",
		Overlay:   "#000003",
		Text:      "#000004",
		Subtext:   "#000005",
		Muted:     "#000006",
		Primary:   "#000007",
		Secondary: "#000008",
		Success:   "#000009",
		Warning:   "#00000a",
		Error:     "#00000b",
		Info:      "#00000c",
	}

	theme := GetTheme("catppuccin-mocha", customColors)

	if theme.Colors.Base != "#000001" {
		t.Errorf("Base override failed")
	}
	if theme.Colors.Surface != "#000002" {
		t.Errorf("Surface override failed")
	}
	if theme.Colors.Overlay != "#000003" {
		t.Errorf("Overlay override failed")
	}
	if theme.Colors.Text != "#000004" {
		t.Errorf("Text override failed")
	}
	if theme.Colors.Subtext != "#000005" {
		t.Errorf("Subtext override failed")
	}
	if theme.Colors.Muted != "#000006" {
		t.Errorf("Muted override failed")
	}
	if theme.Colors.Primary != "#000007" {
		t.Errorf("Primary override failed")
	}
	if theme.Colors.Secondary != "#000008" {
		t.Errorf("Secondary override failed")
	}
	if theme.Colors.Success != "#000009" {
		t.Errorf("Success override failed")
	}
	if theme.Colors.Warning != "#00000a" {
		t.Errorf("Warning override failed")
	}
	if theme.Colors.Error != "#00000b" {
		t.Errorf("Error override failed")
	}
	if theme.Colors.Info != "#00000c" {
		t.Errorf("Info override failed")
	}
}

func TestIsValidTheme(t *testing.T) {
	validThemes := []string{
		"catppuccin-mocha",
		"dracula",
		"nord",
		"gruvbox-dark",
		"tokyo-night",
	}

	for _, name := range validThemes {
		if !IsValidTheme(name) {
			t.Errorf("IsValidTheme(%q) = false; want true", name)
		}
	}

	invalidThemes := []string{
		"nonexistent",
		"",
		"CATPPUCCIN-MOCHA",
		"catppuccin",
	}

	for _, name := range invalidThemes {
		if IsValidTheme(name) {
			t.Errorf("IsValidTheme(%q) = true; want false", name)
		}
	}
}

func TestBuiltinThemes_AllHaveRequiredColors(t *testing.T) {
	hexColorRegex := regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

	for name, theme := range BuiltinThemes {
		t.Run(name, func(t *testing.T) {
			if theme.Name == "" {
				t.Errorf("theme %q has empty Name", name)
			}

			colors := map[string]string{
				"Base":      theme.Colors.Base,
				"Surface":   theme.Colors.Surface,
				"Overlay":   theme.Colors.Overlay,
				"Text":      theme.Colors.Text,
				"Subtext":   theme.Colors.Subtext,
				"Muted":     theme.Colors.Muted,
				"Primary":   theme.Colors.Primary,
				"Secondary": theme.Colors.Secondary,
				"Success":   theme.Colors.Success,
				"Warning":   theme.Colors.Warning,
				"Error":     theme.Colors.Error,
				"Info":      theme.Colors.Info,
			}

			for colorName, colorValue := range colors {
				if colorValue == "" {
					t.Errorf("theme %q has empty %s color", name, colorName)
				} else if !hexColorRegex.MatchString(colorValue) {
					t.Errorf("theme %q has invalid %s color: %q (expected #RRGGBB format)", name, colorName, colorValue)
				}
			}
		})
	}
}

func TestBuiltinThemes_Count(t *testing.T) {
	if len(BuiltinThemes) != 20 {
		t.Errorf("BuiltinThemes has %d themes; want 20", len(BuiltinThemes))
	}
}

func TestBuiltinThemes_ExpectedThemesExist(t *testing.T) {
	expectedThemes := []string{
		"catppuccin-mocha",
		"catppuccin-macchiato",
		"catppuccin-frappe",
		"catppuccin-latte",
		"tokyo-night",
		"tokyo-night-storm",
		"tokyo-night-light",
		"gruvbox-dark",
		"gruvbox-light",
		"nord",
		"dracula",
		"one-dark",
		"solarized-dark",
		"solarized-light",
		"rose-pine",
		"rose-pine-moon",
		"rose-pine-dawn",
		"kanagawa",
		"everforest-dark",
		"everforest-light",
	}

	for _, name := range expectedThemes {
		if _, exists := BuiltinThemes[name]; !exists {
			t.Errorf("expected theme %q not found in BuiltinThemes", name)
		}
	}
}
