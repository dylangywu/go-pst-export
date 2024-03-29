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

package pstexport

import (
	"errors"
	pst "github.com/mooijtech/go-pst/v4/pkg"
)

// ExportStrategy defines the interface all export strategies implement.
type ExportStrategy interface {
	Name() string
	Export(pstFile *pst.File, pstMessage pst.Message, pstMessageIndex int, pstFolder pst.Folder, pstFormatType string, pstEncryptionType string, exportContext ExportContext) error
}

// GetAllExportStrategies returns all export strategies.
func GetAllExportStrategies() []ExportStrategy {
	return []ExportStrategy{ExportStrategyEML{}}
}

// GetExportStrategyByName returns the export strategy by the specified name.
func GetExportStrategyByName(name string) (ExportStrategy, error) {
	for _, strategy := range GetAllExportStrategies() {
		if strategy.Name() == name {
			return strategy, nil
		}
	}

	return nil, errors.New("failed to find export strategy by name")
}

// ExecuteExportStrategy executes the export strategy.
// Processes the PST file and calls Export on the export strategy for every message.
func ExecuteExportStrategy(exportStrategy ExportStrategy, exportContext ExportContext) error {
	Logger.Infof("Executing export strategy: %s", exportStrategy.Name())
	Logger.Info("Processing PST file...")

	pstFile, err := pst.NewFromFile(exportContext.InputFile)

	if err != nil {
		return err
	}

	isValidSignature, err := pstFile.IsValidSignature()

	if err != nil {
		return err
	}

	if !isValidSignature {
		return errors.New("invalid input file signature")
	}

	formatType, err := pstFile.GetFormatType()

	if err != nil {
		return err
	}

	encryptionType, err := pstFile.GetEncryptionType(formatType)

	if err != nil {
		return err
	}

	Logger.Info("Initializing b-trees...")

	err = pstFile.InitializeBTrees(formatType)

	if err != nil {
		return err
	}

	rootFolder, err := pstFile.GetRootFolder(formatType, encryptionType)

	if err != nil {
		return err
	}

	err = processSubFolders(&pstFile, rootFolder, formatType, encryptionType, exportStrategy, exportContext)

	if err != nil {
		return err
	}

	return pstFile.Close()
}

// processSubFolders processes the folders and all messages.
func processSubFolders(pstFile *pst.File, folder pst.Folder, formatType string, encryptionType string, exportStrategy ExportStrategy, exportContext ExportContext) error {
	subFolders, err := pstFile.GetSubFolders(folder, formatType, encryptionType)

	if err != nil {
		return err
	}

	for _, subFolder := range subFolders {
		Logger.Infof("Processing sub-folder: %s\n", subFolder.DisplayName)

		messages, err := pstFile.GetMessages(subFolder, formatType, encryptionType)

		if err != nil {
			return err
		}

		if len(messages) > 0 {
			Logger.Infof("Processing %d messages...", len(messages))

			for messageIndex, message := range messages {
				err := exportStrategy.Export(pstFile, message, messageIndex, subFolder, formatType, encryptionType, exportContext)

				if err != nil {
					Logger.Errorf("Failed to export message (skipping): %s", err)
				}
			}
		}

		err = processSubFolders(pstFile, subFolder, formatType, encryptionType, exportStrategy, exportContext)

		if err != nil {
			return err
		}
	}

	return nil
}