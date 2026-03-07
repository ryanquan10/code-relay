package main

import "embed"

//go:embed all:internal/frontend/dist
var FrontendFS embed.FS
