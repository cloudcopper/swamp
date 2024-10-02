package swamp

import "embed"

//go:embed templates/**
//go:embed static/**
var appFS embed.FS
