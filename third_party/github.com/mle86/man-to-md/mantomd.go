package mantomd

import _ "embed"

//go:embed man-to-md.pl
var Script string // TODO port to golang
