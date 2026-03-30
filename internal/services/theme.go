package services

import "strings"

type themePalette struct {
	PrimaryColor    string
	SecondaryColor  string
	BackgroundColor string
	TextColor       string
}

var themePalettes = map[string]themePalette{
	"classic": {
		PrimaryColor:    "#2563eb",
		SecondaryColor:  "#ffffff",
		BackgroundColor: "#f8fafc",
		TextColor:       "#111827",
	},
	"minimal": {
		PrimaryColor:    "#1f2937",
		SecondaryColor:  "#e5e7eb",
		BackgroundColor: "#ffffff",
		TextColor:       "#111827",
	},
	"party": {
		PrimaryColor:    "#db2777",
		SecondaryColor:  "#fef3c7",
		BackgroundColor: "#1f2937",
		TextColor:       "#f9fafb",
	},
}

func normalizeTheme(theme string) string {
	theme = strings.ToLower(strings.TrimSpace(theme))
	if theme == "" {
		return "classic"
	}
	return theme
}

func paletteForTheme(theme string) themePalette {
	palette, ok := themePalettes[normalizeTheme(theme)]
	if !ok {
		return themePalettes["classic"]
	}
	return palette
}
