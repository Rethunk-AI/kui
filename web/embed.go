package web

import "embed"

// Dist embeds the built frontend from web/dist.
// Run `cd web && npm run build` before `go build`.
//
//go:embed dist/*
var Dist embed.FS
