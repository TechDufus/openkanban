package config

// Theme represents a color theme for the UI
type Theme struct {
	Name   string      `json:"name"`
	Colors ThemeColors `json:"colors"`
}

// ThemeColors contains all color values for a theme
type ThemeColors struct {
	// Background colors
	Base    string `json:"base"`    // Main background
	Surface string `json:"surface"` // Elevated surfaces (cards, panels)
	Overlay string `json:"overlay"` // Highest elevation (modals, dropdowns)

	// Text colors
	Text    string `json:"text"`    // Primary text
	Subtext string `json:"subtext"` // Secondary text
	Muted   string `json:"muted"`   // Disabled/placeholder text

	// Accent colors
	Blue   string `json:"blue"`   // Primary accent, info
	Green  string `json:"green"`  // Success, confirmations
	Yellow string `json:"yellow"` // Warnings, in-progress
	Red    string `json:"red"`    // Errors, destructive
	Mauve  string `json:"mauve"`  // Purple accent, special
	Teal   string `json:"teal"`   // Cyan accent, links
}

// BuiltinThemes contains all pre-defined themes
var BuiltinThemes = map[string]Theme{
	// Catppuccin variants
	"catppuccin-mocha": {
		Name: "Catppuccin Mocha",
		Colors: ThemeColors{
			Base:    "#1e1e2e",
			Surface: "#313244",
			Overlay: "#45475a",
			Text:    "#cdd6f4",
			Subtext: "#a6adc8",
			Muted:   "#6c7086",
			Blue:    "#89b4fa",
			Green:   "#a6e3a1",
			Yellow:  "#f9e2af",
			Red:     "#f38ba8",
			Mauve:   "#cba6f7",
			Teal:    "#94e2d5",
		},
	},
	"catppuccin-macchiato": {
		Name: "Catppuccin Macchiato",
		Colors: ThemeColors{
			Base:    "#24273a",
			Surface: "#363a4f",
			Overlay: "#494d64",
			Text:    "#cad3f5",
			Subtext: "#a5adcb",
			Muted:   "#6e738d",
			Blue:    "#8aadf4",
			Green:   "#a6da95",
			Yellow:  "#eed49f",
			Red:     "#ed8796",
			Mauve:   "#c6a0f6",
			Teal:    "#8bd5ca",
		},
	},
	"catppuccin-frappe": {
		Name: "Catppuccin Frappe",
		Colors: ThemeColors{
			Base:    "#303446",
			Surface: "#414559",
			Overlay: "#51576d",
			Text:    "#c6d0f5",
			Subtext: "#a5adce",
			Muted:   "#737994",
			Blue:    "#8caaee",
			Green:   "#a6d189",
			Yellow:  "#e5c890",
			Red:     "#e78284",
			Mauve:   "#ca9ee6",
			Teal:    "#81c8be",
		},
	},
	"catppuccin-latte": {
		Name: "Catppuccin Latte",
		Colors: ThemeColors{
			Base:    "#eff1f5",
			Surface: "#e6e9ef",
			Overlay: "#ccd0da",
			Text:    "#4c4f69",
			Subtext: "#5c5f77",
			Muted:   "#9ca0b0",
			Blue:    "#1e66f5",
			Green:   "#40a02b",
			Yellow:  "#df8e1d",
			Red:     "#d20f39",
			Mauve:   "#8839ef",
			Teal:    "#179299",
		},
	},

	// Tokyo Night variants
	"tokyo-night": {
		Name: "Tokyo Night",
		Colors: ThemeColors{
			Base:    "#1a1b26",
			Surface: "#24283b",
			Overlay: "#414868",
			Text:    "#c0caf5",
			Subtext: "#a9b1d6",
			Muted:   "#565f89",
			Blue:    "#7aa2f7",
			Green:   "#9ece6a",
			Yellow:  "#e0af68",
			Red:     "#f7768e",
			Mauve:   "#bb9af7",
			Teal:    "#7dcfff",
		},
	},
	"tokyo-night-storm": {
		Name: "Tokyo Night Storm",
		Colors: ThemeColors{
			Base:    "#24283b",
			Surface: "#1f2335",
			Overlay: "#414868",
			Text:    "#c0caf5",
			Subtext: "#a9b1d6",
			Muted:   "#565f89",
			Blue:    "#7aa2f7",
			Green:   "#9ece6a",
			Yellow:  "#e0af68",
			Red:     "#f7768e",
			Mauve:   "#bb9af7",
			Teal:    "#7dcfff",
		},
	},
	"tokyo-night-light": {
		Name: "Tokyo Night Light",
		Colors: ThemeColors{
			Base:    "#d5d6db",
			Surface: "#cbccd1",
			Overlay: "#9699a3",
			Text:    "#343b58",
			Subtext: "#4c505e",
			Muted:   "#9699a3",
			Blue:    "#34548a",
			Green:   "#485e30",
			Yellow:  "#8f5e15",
			Red:     "#8c4351",
			Mauve:   "#5a4a78",
			Teal:    "#166775",
		},
	},

	// Gruvbox variants
	"gruvbox-dark": {
		Name: "Gruvbox Dark",
		Colors: ThemeColors{
			Base:    "#282828",
			Surface: "#3c3836",
			Overlay: "#504945",
			Text:    "#ebdbb2",
			Subtext: "#d5c4a1",
			Muted:   "#928374",
			Blue:    "#83a598",
			Green:   "#b8bb26",
			Yellow:  "#fabd2f",
			Red:     "#fb4934",
			Mauve:   "#d3869b",
			Teal:    "#8ec07c",
		},
	},
	"gruvbox-light": {
		Name: "Gruvbox Light",
		Colors: ThemeColors{
			Base:    "#fbf1c7",
			Surface: "#ebdbb2",
			Overlay: "#d5c4a1",
			Text:    "#3c3836",
			Subtext: "#504945",
			Muted:   "#928374",
			Blue:    "#458588",
			Green:   "#79740e",
			Yellow:  "#b57614",
			Red:     "#9d0006",
			Mauve:   "#8f3f71",
			Teal:    "#427b58",
		},
	},

	// Nord
	"nord": {
		Name: "Nord",
		Colors: ThemeColors{
			Base:    "#2e3440",
			Surface: "#3b4252",
			Overlay: "#434c5e",
			Text:    "#eceff4",
			Subtext: "#e5e9f0",
			Muted:   "#4c566a",
			Blue:    "#81a1c1",
			Green:   "#a3be8c",
			Yellow:  "#ebcb8b",
			Red:     "#bf616a",
			Mauve:   "#b48ead",
			Teal:    "#88c0d0",
		},
	},

	// Dracula
	"dracula": {
		Name: "Dracula",
		Colors: ThemeColors{
			Base:    "#282a36",
			Surface: "#44475a",
			Overlay: "#6272a4",
			Text:    "#f8f8f2",
			Subtext: "#e9e9e4",
			Muted:   "#6272a4",
			Blue:    "#8be9fd",
			Green:   "#50fa7b",
			Yellow:  "#f1fa8c",
			Red:     "#ff5555",
			Mauve:   "#bd93f9",
			Teal:    "#8be9fd",
		},
	},

	// One Dark
	"one-dark": {
		Name: "One Dark",
		Colors: ThemeColors{
			Base:    "#282c34",
			Surface: "#21252b",
			Overlay: "#3e4451",
			Text:    "#abb2bf",
			Subtext: "#9da5b4",
			Muted:   "#5c6370",
			Blue:    "#61afef",
			Green:   "#98c379",
			Yellow:  "#e5c07b",
			Red:     "#e06c75",
			Mauve:   "#c678dd",
			Teal:    "#56b6c2",
		},
	},

	// Solarized variants
	"solarized-dark": {
		Name: "Solarized Dark",
		Colors: ThemeColors{
			Base:    "#002b36",
			Surface: "#073642",
			Overlay: "#586e75",
			Text:    "#839496",
			Subtext: "#93a1a1",
			Muted:   "#657b83",
			Blue:    "#268bd2",
			Green:   "#859900",
			Yellow:  "#b58900",
			Red:     "#dc322f",
			Mauve:   "#6c71c4",
			Teal:    "#2aa198",
		},
	},
	"solarized-light": {
		Name: "Solarized Light",
		Colors: ThemeColors{
			Base:    "#fdf6e3",
			Surface: "#eee8d5",
			Overlay: "#93a1a1",
			Text:    "#657b83",
			Subtext: "#586e75",
			Muted:   "#93a1a1",
			Blue:    "#268bd2",
			Green:   "#859900",
			Yellow:  "#b58900",
			Red:     "#dc322f",
			Mauve:   "#6c71c4",
			Teal:    "#2aa198",
		},
	},

	// Rosé Pine variants
	"rose-pine": {
		Name: "Rose Pine",
		Colors: ThemeColors{
			Base:    "#191724",
			Surface: "#1f1d2e",
			Overlay: "#26233a",
			Text:    "#e0def4",
			Subtext: "#908caa",
			Muted:   "#6e6a86",
			Blue:    "#31748f",
			Green:   "#9ccfd8",
			Yellow:  "#f6c177",
			Red:     "#eb6f92",
			Mauve:   "#c4a7e7",
			Teal:    "#9ccfd8",
		},
	},
	"rose-pine-moon": {
		Name: "Rose Pine Moon",
		Colors: ThemeColors{
			Base:    "#232136",
			Surface: "#2a273f",
			Overlay: "#393552",
			Text:    "#e0def4",
			Subtext: "#908caa",
			Muted:   "#6e6a86",
			Blue:    "#3e8fb0",
			Green:   "#9ccfd8",
			Yellow:  "#f6c177",
			Red:     "#eb6f92",
			Mauve:   "#c4a7e7",
			Teal:    "#9ccfd8",
		},
	},
	"rose-pine-dawn": {
		Name: "Rose Pine Dawn",
		Colors: ThemeColors{
			Base:    "#faf4ed",
			Surface: "#fffaf3",
			Overlay: "#f2e9e1",
			Text:    "#575279",
			Subtext: "#797593",
			Muted:   "#9893a5",
			Blue:    "#286983",
			Green:   "#56949f",
			Yellow:  "#ea9d34",
			Red:     "#b4637a",
			Mauve:   "#907aa9",
			Teal:    "#56949f",
		},
	},

	// Kanagawa
	"kanagawa": {
		Name: "Kanagawa",
		Colors: ThemeColors{
			Base:    "#1f1f28",
			Surface: "#2a2a37",
			Overlay: "#363646",
			Text:    "#dcd7ba",
			Subtext: "#c8c093",
			Muted:   "#727169",
			Blue:    "#7e9cd8",
			Green:   "#98bb6c",
			Yellow:  "#e6c384",
			Red:     "#c34043",
			Mauve:   "#957fb8",
			Teal:    "#7fb4ca",
		},
	},

	// Everforest
	"everforest-dark": {
		Name: "Everforest Dark",
		Colors: ThemeColors{
			Base:    "#2d353b",
			Surface: "#343f44",
			Overlay: "#3d484d",
			Text:    "#d3c6aa",
			Subtext: "#9da9a0",
			Muted:   "#7a8478",
			Blue:    "#7fbbb3",
			Green:   "#a7c080",
			Yellow:  "#dbbc7f",
			Red:     "#e67e80",
			Mauve:   "#d699b6",
			Teal:    "#83c092",
		},
	},
	"everforest-light": {
		Name: "Everforest Light",
		Colors: ThemeColors{
			Base:    "#fdf6e3",
			Surface: "#f4f0d9",
			Overlay: "#e6e2cc",
			Text:    "#5c6a72",
			Subtext: "#708089",
			Muted:   "#939f91",
			Blue:    "#3a94c5",
			Green:   "#8da101",
			Yellow:  "#dfa000",
			Red:     "#f85552",
			Mauve:   "#df69ba",
			Teal:    "#35a77c",
		},
	},
}

// ThemeNames returns a sorted list of all available theme names
func ThemeNames() []string {
	return []string{
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
}

// GetTheme returns a theme by name, with optional custom color overrides
func GetTheme(name string, customColors *ThemeColors) Theme {
	theme, exists := BuiltinThemes[name]
	if !exists {
		// Fall back to catppuccin-mocha
		theme = BuiltinThemes["catppuccin-mocha"]
	}

	// Apply custom color overrides if provided
	if customColors != nil {
		if customColors.Base != "" {
			theme.Colors.Base = customColors.Base
		}
		if customColors.Surface != "" {
			theme.Colors.Surface = customColors.Surface
		}
		if customColors.Overlay != "" {
			theme.Colors.Overlay = customColors.Overlay
		}
		if customColors.Text != "" {
			theme.Colors.Text = customColors.Text
		}
		if customColors.Subtext != "" {
			theme.Colors.Subtext = customColors.Subtext
		}
		if customColors.Muted != "" {
			theme.Colors.Muted = customColors.Muted
		}
		if customColors.Blue != "" {
			theme.Colors.Blue = customColors.Blue
		}
		if customColors.Green != "" {
			theme.Colors.Green = customColors.Green
		}
		if customColors.Yellow != "" {
			theme.Colors.Yellow = customColors.Yellow
		}
		if customColors.Red != "" {
			theme.Colors.Red = customColors.Red
		}
		if customColors.Mauve != "" {
			theme.Colors.Mauve = customColors.Mauve
		}
		if customColors.Teal != "" {
			theme.Colors.Teal = customColors.Teal
		}
	}

	return theme
}

// IsValidTheme checks if a theme name is valid
func IsValidTheme(name string) bool {
	_, exists := BuiltinThemes[name]
	return exists
}
