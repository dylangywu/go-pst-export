// Package pstexport
// This file is part of go-pst-export (https://github.com/mooijtech/go-pst-export)
// Copyright (C) 2022 Marten Mooij (https://www.mooijtech.com/)
package pstexport

// ExportContext defines the context used when using an export strategy.
type ExportContext struct {
	InputFile string
	OutputDirectory string
	IsOnlyPlaintextBody bool
}
