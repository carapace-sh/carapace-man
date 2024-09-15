module github.com/carapace-sh/carapace-man

go 1.23.0

require (
	github.com/carapace-sh/carapace v1.2.0
	github.com/carapace-sh/carapace-spec v1.0.5
	github.com/charmbracelet/glamour v0.8.0
	github.com/charmbracelet/lipgloss v0.13.0
	github.com/charmbracelet/x/termios v0.1.0
	github.com/jmorganca/ollama v0.1.30-rc4
	github.com/mattn/go-isatty v0.0.20
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.27.0
	golang.org/x/term v0.24.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/alecthomas/chroma/v2 v2.14.0 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/carapace-sh/carapace-shlex v1.0.1 // indirect
	github.com/charmbracelet/x/ansi v0.3.1 // indirect
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/invopop/jsonschema v0.12.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.15.3-0.20240618155329-98d742f6907a // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.11.0 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	github.com/yuin/goldmark v1.7.4 // indirect
	github.com/yuin/goldmark-emoji v1.0.3 // indirect
	golang.org/x/net v0.29.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace github.com/spf13/pflag => github.com/carapace-sh/carapace-pflag v1.0.0

replace github.com/carapace-sh/carapace-spec => ../carapace-spec/
