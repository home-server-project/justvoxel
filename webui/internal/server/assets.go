package server

import (
	"context"
	"embed"
)

//go:embed templates/*.html static/* locales/*.json
var assets embed.FS

var _ = context.Background
