package man

import (
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
)

// CarapaceStyleConfig is the carapace style. (TODO: wip - based on tokyo-night and cattpuccin-mocha)
var CarapaceStyleConfig = ansi.StyleConfig{
	Document: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:           stringPtr("#a9b1d6"),
			BackgroundColor: stringPtr("#181825"),
		},
		Indent: uintPtr(0),
		Margin: uintPtr(0),
	},
	BlockQuote: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{},
		Indent:         uintPtr(1),
		IndentToken:    stringPtr("│ "),
	},
	List: ansi.StyleList{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr("#a9b1d6"),
			},
		},
		LevelIndent: 2,
	},
	Heading: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			BlockSuffix: "\n",
			Color:       stringPtr("#bb9af7"),
			Bold:        boolPtr(true),
		},
	},
	H1: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "# ",
			Bold:   boolPtr(true),
		},
	},
	H2: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "## ",
		},
	},
	H3: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "### ",
		},
	},
	H4: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "#### ",
		},
	},
	H5: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "##### ",
		},
	},
	H6: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Prefix: "###### ",
		},
	},
	Strikethrough: ansi.StylePrimitive{
		CrossedOut: boolPtr(true),
	},
	Emph: ansi.StylePrimitive{
		Italic: boolPtr(true),
	},
	Strong: ansi.StylePrimitive{
		Bold: boolPtr(true),
	},
	HorizontalRule: ansi.StylePrimitive{
		Color:  stringPtr("#565f89"),
		Format: "\n--------\n",
	},
	Item: ansi.StylePrimitive{
		BlockPrefix: "• ",
	},
	Enumeration: ansi.StylePrimitive{
		BlockPrefix: ". ",
		Color:       stringPtr("#7aa2f7"),
	},
	Task: ansi.StyleTask{
		StylePrimitive: ansi.StylePrimitive{},
		Ticked:         "[✓] ",
		Unticked:       "[ ] ",
	},
	Link: ansi.StylePrimitive{
		Color:           stringPtr("#7aa2f7"),
		Underline:       boolPtr(true),
		BackgroundColor: stringPtr("#181825"),
	},
	LinkText: ansi.StylePrimitive{
		Color:           stringPtr("#2ac3de"),
		BackgroundColor: stringPtr("#181825"),
	},
	Image: ansi.StylePrimitive{
		Color:     stringPtr("#7aa2f7"),
		Underline: boolPtr(true),
	},
	ImageText: ansi.StylePrimitive{
		Color:  stringPtr("#2ac3de"),
		Format: "Image: {{.text}} →",
	},
	Code: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:           stringPtr("#89B4FA"),
			BackgroundColor: stringPtr("#181825"),
		},
	},
	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           stringPtr("#ff9e64"),
				BackgroundColor: stringPtr("#181825"),
			},
			Margin: uintPtr(2),
		},
		Theme: "cattpuccin-mocha",
	},
	Table: ansi.StyleTable{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{},
		},
	},
	Text: ansi.StylePrimitive{
		Color:           stringPtr("#cdd6f4"),
		BackgroundColor: stringPtr("#181825"),
	},
	DefinitionDescription: ansi.StylePrimitive{
		BlockPrefix: "\n🠶 ",
	},
}

func init() {
	styles.DefaultStyles["carapace"] = &CarapaceStyleConfig
}

func boolPtr(b bool) *bool       { return &b }
func stringPtr(s string) *string { return &s }
func uintPtr(u uint) *uint       { return &u }
