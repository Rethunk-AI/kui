package web

import "embed"

// Dist embeds the built frontend from web/dist.
// Run `cd web && corepack yarn run build` before `go build`.
//
//go:embed dist/*
var Dist embed.FS
