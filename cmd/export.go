// go-pst-export is a command-line interface and library for exporting PST files (using go-pst).
//
// Copyright (C) 2022  Marten Mooij
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
	isOnlyPlaintextBody := flag.Bool("plaintext", false, "only get the plaintext body (not HTML)")

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
			InputFile:           *inputFile,
			OutputDirectory:     *outputDirectory,
			IsOnlyPlaintextBody: *isOnlyPlaintextBody,
		})

		if err != nil {
			pstexport.Logger.Errorf("Failed to export: %s", err)
			return
		}
	}
}
