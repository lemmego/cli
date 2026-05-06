package cli

import "embed"

//go:embed _scaffold/** _scaffold/base/.air.toml _scaffold/base/.gitignore
var scaffoldFS embed.FS
