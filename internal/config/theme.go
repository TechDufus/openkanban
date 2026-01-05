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

	// Semantic accent colors
	Primary   string `json:"primary"`   // Main accent (focus, selection, headers, backlog)
	Secondary string `json:"secondary"` // Secondary accent (links, highlights)
	Success   string `json:"success"`   // Positive states (done, confirmations)
	Warning   string `json:"warning"`   // Caution states (in-progress, warnings)
	Error     string `json:"error"`     // Errors, destructive actions
	Info      string `json:"info"`      // Informational elements
}

// BuiltinThemes contains all pre-defined themes
var BuiltinThemes = map[string]Theme{
	// Catppuccin variants
	"catppuccin-mocha": {
		Name: "Catppuccin Mocha",
		Colors: ThemeColors{
			Base:      "#1e1e2e",
			Surface:   "#313244",
			Overlay:   "#45475a",
			Text:      "#cdd6f4",
			Subtext:   "#bac2de",
			Muted:     "#6c7086",
			Primary:   "#89b4fa",
			Secondary: "#cba6f7",
			Success:   "#a6e3a1",
			Warning:   "#f9e2af",
			Error:     "#f38ba8",
			Info:      "#94e2d5",
		},
	},
	"catppuccin-macchiato": {
		Name: "Catppuccin Macchiato",
		Colors: ThemeColors{
			Base:      "#24273a",
			Surface:   "#363a4f",
			Overlay:   "#494d64",
			Text:      "#cad3f5",
			Subtext:   "#b8c0e0",
			Muted:     "#6e738d",
			Primary:   "#8aadf4",
			Secondary: "#c6a0f6",
			Success:   "#a6da95",
			Warning:   "#eed49f",
			Error:     "#ed8796",
			Info:      "#8bd5ca",
		},
	},
	"catppuccin-frappe": {
		Name: "Catppuccin Frappe",
		Colors: ThemeColors{
			Base:      "#303446",
			Surface:   "#414559",
			Overlay:   "#51576d",
			Text:      "#c6d0f5",
			Subtext:   "#b5bfe2",
			Muted:     "#737994",
			Primary:   "#8caaee",
			Secondary: "#ca9ee6",
			Success:   "#a6d189",
			Warning:   "#e5c890",
			Error:     "#e78284",
			Info:      "#81c8be",
		},
	},
	"catppuccin-latte": {
		Name: "Catppuccin Latte",
		Colors: ThemeColors{
			Base:      "#eff1f5",
			Surface:   "#ccd0da",
			Overlay:   "#bcc0cc",
			Text:      "#4c4f69",
			Subtext:   "#5c5f77",
			Muted:     "#9ca0b0",
			Primary:   "#1e66f5",
			Secondary: "#8839ef",
			Success:   "#40a02b",
			Warning:   "#df8e1d",
			Error:     "#d20f39",
			Info:      "#179299",
		},
	},

	// Tokyo Night variants
	"tokyo-night": {
		Name: "Tokyo Night",
		Colors: ThemeColors{
			Base:      "#1a1b26",
			Surface:   "#16161e",
			Overlay:   "#292e42",
			Text:      "#c0caf5",
			Subtext:   "#a9b1d6",
			Muted:     "#565f89",
			Primary:   "#7aa2f7",
			Secondary: "#bb9af7",
			Success:   "#9ece6a",
			Warning:   "#e0af68",
			Error:     "#f7768e",
			Info:      "#7dcfff",
		},
	},
	"tokyo-night-storm": {
		Name: "Tokyo Night Storm",
		Colors: ThemeColors{
			Base:      "#24283b",
			Surface:   "#1f2335",
			Overlay:   "#292e42",
			Text:      "#c0caf5",
			Subtext:   "#a9b1d6",
			Muted:     "#565f89",
			Primary:   "#7aa2f7",
			Secondary: "#bb9af7",
			Success:   "#9ece6a",
			Warning:   "#e0af68",
			Error:     "#f7768e",
			Info:      "#7dcfff",
		},
	},
	"tokyo-night-light": {
		Name: "Tokyo Night Day",
		Colors: ThemeColors{
			Base:      "#e1e2e7",
			Surface:   "#d5d6db",
			Overlay:   "#c4c5ca",
			Text:      "#3760bf",
			Subtext:   "#6172b0",
			Muted:     "#848cb5",
			Primary:   "#2e7de9",
			Secondary: "#9854f1",
			Success:   "#587539",
			Warning:   "#8c6c3e",
			Error:     "#f52a65",
			Info:      "#007197",
		},
	},

	// Gruvbox variants
	"gruvbox-dark": {
		Name: "Gruvbox Dark",
		Colors: ThemeColors{
			Base:      "#282828",
			Surface:   "#3c3836",
			Overlay:   "#504945",
			Text:      "#ebdbb2",
			Subtext:   "#d5c4a1",
			Muted:     "#928374",
			Primary:   "#83a598",
			Secondary: "#d3869b",
			Success:   "#b8bb26",
			Warning:   "#fabd2f",
			Error:     "#fb4934",
			Info:      "#8ec07c",
		},
	},
	"gruvbox-light": {
		Name: "Gruvbox Light",
		Colors: ThemeColors{
			Base:      "#fbf1c7",
			Surface:   "#ebdbb2",
			Overlay:   "#d5c4a1",
			Text:      "#3c3836",
			Subtext:   "#504945",
			Muted:     "#928374",
			Primary:   "#076678",
			Secondary: "#8f3f71",
			Success:   "#79740e",
			Warning:   "#b57614",
			Error:     "#9d0006",
			Info:      "#427b58",
		},
	},

	// Nord
	"nord": {
		Name: "Nord",
		Colors: ThemeColors{
			Base:      "#2e3440",
			Surface:   "#3b4252",
			Overlay:   "#434c5e",
			Text:      "#eceff4",
			Subtext:   "#e5e9f0",
			Muted:     "#4c566a",
			Primary:   "#5e81ac",
			Secondary: "#b48ead",
			Success:   "#a3be8c",
			Warning:   "#ebcb8b",
			Error:     "#bf616a",
			Info:      "#88c0d0",
		},
	},

	// Dracula
	"dracula": {
		Name: "Dracula",
		Colors: ThemeColors{
			Base:      "#282a36",
			Surface:   "#44475a",
			Overlay:   "#6272a4",
			Text:      "#f8f8f2",
			Subtext:   "#e9e9e4",
			Muted:     "#6272a4",
			Primary:   "#bd93f9",
			Secondary: "#ff79c6",
			Success:   "#50fa7b",
			Warning:   "#f1fa8c",
			Error:     "#ff5555",
			Info:      "#8be9fd",
		},
	},

	// One Dark
	"one-dark": {
		Name: "One Dark",
		Colors: ThemeColors{
			Base:      "#282c34",
			Surface:   "#21252b",
			Overlay:   "#2c313a",
			Text:      "#abb2bf",
			Subtext:   "#828997",
			Muted:     "#5c6370",
			Primary:   "#61afef",
			Secondary: "#c678dd",
			Success:   "#98c379",
			Warning:   "#e5c07b",
			Error:     "#e06c75",
			Info:      "#56b6c2",
		},
	},

	// Solarized variants
	"solarized-dark": {
		Name: "Solarized Dark",
		Colors: ThemeColors{
			Base:      "#002b36",
			Surface:   "#073642",
			Overlay:   "#586e75",
			Text:      "#839496",
			Subtext:   "#93a1a1",
			Muted:     "#657b83",
			Primary:   "#268bd2",
			Secondary: "#6c71c4",
			Success:   "#859900",
			Warning:   "#b58900",
			Error:     "#dc322f",
			Info:      "#2aa198",
		},
	},
	"solarized-light": {
		Name: "Solarized Light",
		Colors: ThemeColors{
			Base:      "#fdf6e3",
			Surface:   "#eee8d5",
			Overlay:   "#93a1a1",
			Text:      "#657b83",
			Subtext:   "#586e75",
			Muted:     "#93a1a1",
			Primary:   "#268bd2",
			Secondary: "#6c71c4",
			Success:   "#859900",
			Warning:   "#b58900",
			Error:     "#dc322f",
			Info:      "#2aa198",
		},
	},

	// Rosé Pine variants
	"rose-pine": {
		Name: "Rose Pine",
		Colors: ThemeColors{
			Base:      "#191724",
			Surface:   "#1f1d2e",
			Overlay:   "#26233a",
			Text:      "#e0def4",
			Subtext:   "#908caa",
			Muted:     "#6e6a86",
			Primary:   "#31748f",
			Secondary: "#c4a7e7",
			Success:   "#9ccfd8",
			Warning:   "#f6c177",
			Error:     "#eb6f92",
			Info:      "#9ccfd8",
		},
	},
	"rose-pine-moon": {
		Name: "Rose Pine Moon",
		Colors: ThemeColors{
			Base:      "#232136",
			Surface:   "#2a273f",
			Overlay:   "#393552",
			Text:      "#e0def4",
			Subtext:   "#908caa",
			Muted:     "#6e6a86",
			Primary:   "#3e8fb0",
			Secondary: "#c4a7e7",
			Success:   "#9ccfd8",
			Warning:   "#f6c177",
			Error:     "#eb6f92",
			Info:      "#9ccfd8",
		},
	},
	"rose-pine-dawn": {
		Name: "Rose Pine Dawn",
		Colors: ThemeColors{
			Base:      "#faf4ed",
			Surface:   "#fffaf3",
			Overlay:   "#f2e9e1",
			Text:      "#575279",
			Subtext:   "#797593",
			Muted:     "#9893a5",
			Primary:   "#286983",
			Secondary: "#907aa9",
			Success:   "#56949f",
			Warning:   "#ea9d34",
			Error:     "#b4637a",
			Info:      "#56949f",
		},
	},

	// Kanagawa
	"kanagawa": {
		Name: "Kanagawa",
		Colors: ThemeColors{
			Base:      "#1f1f28",
			Surface:   "#2a2a37",
			Overlay:   "#363646",
			Text:      "#dcd7ba",
			Subtext:   "#c8c093",
			Muted:     "#727169",
			Primary:   "#7e9cd8",
			Secondary: "#957fb8",
			Success:   "#98bb6c",
			Warning:   "#e6c384",
			Error:     "#e46876",
			Info:      "#7aa89f",
		},
	},

	// Everforest
	"everforest-dark": {
		Name: "Everforest Dark",
		Colors: ThemeColors{
			Base:      "#2d353b",
			Surface:   "#343f44",
			Overlay:   "#3d484d",
			Text:      "#d3c6aa",
			Subtext:   "#9da9a0",
			Muted:     "#859289",
			Primary:   "#7fbbb3",
			Secondary: "#d699b6",
			Success:   "#a7c080",
			Warning:   "#dbbc7f",
			Error:     "#e67e80",
			Info:      "#83c092",
		},
	},
	"everforest-light": {
		Name: "Everforest Light",
		Colors: ThemeColors{
			Base:      "#fdf6e3",
			Surface:   "#f4f0d9",
			Overlay:   "#efebd4",
			Text:      "#5c6a72",
			Subtext:   "#829181",
			Muted:     "#939f91",
			Primary:   "#3a94c5",
			Secondary: "#df69ba",
			Success:   "#8da101",
			Warning:   "#dfa000",
			Error:     "#f85552",
			Info:      "#35a77c",
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
		if customColors.Primary != "" {
			theme.Colors.Primary = customColors.Primary
		}
		if customColors.Secondary != "" {
			theme.Colors.Secondary = customColors.Secondary
		}
		if customColors.Success != "" {
			theme.Colors.Success = customColors.Success
		}
		if customColors.Warning != "" {
			theme.Colors.Warning = customColors.Warning
		}
		if customColors.Error != "" {
			theme.Colors.Error = customColors.Error
		}
		if customColors.Info != "" {
			theme.Colors.Info = customColors.Info
		}
	}

	return theme
}

// IsValidTheme checks if a theme name is valid
func IsValidTheme(name string) bool {
	_, exists := BuiltinThemes[name]
	return exists
}
