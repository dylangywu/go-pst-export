// This file is part of go-pst-export (https://github.com/mooijtech/go-pst-export)
// Copyright (C) 2022 Marten Mooij (https://www.mooijtech.com/)
package main

import (
	"flag"
	pstexport "github.com/mooijtech/go-pst-export/pkg"
)

func main() {
	listStrategies := flag.Bool("strategies", false, "lists all available export strategies")
	inputFile := flag.String("input", "data/enron.pst", "input PST file to use")
	outputDirectory := flag.String("output", "data", "sets the output directory")
	exportStrategy := flag.String("strategy", "eml", "sets the export strategy")

	flag.Parse()

	if *listStrategies {
		pstexport.Logger.Info("Export strategies:")

		for _, strategy := range pstexport.GetAllExportStrategies() {
			pstexport.Logger.Info("- " + strategy.Name())
		}
	} else {
		strategy, err := pstexport.GetExportStrategyByName(*exportStrategy)

		if err != nil {
			pstexport.Logger.Error("Failed to find export strategy.")
			return
		}

		err = pstexport.ExecuteExportStrategy(strategy, pstexport.ExportContext{
			InputFile: *inputFile,
			OutputDirectory: *outputDirectory,
		})

		if err != nil {
			pstexport.Logger.Errorf("Failed to export: %s", err)
			return
		}
	}
}
