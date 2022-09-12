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
	"bufio"
	"bytes"
	"errors"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	pst "github.com/mooijtech/go-pst/v4/pkg"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

// ExportStrategyEML implements exporting to EML.
type ExportStrategyEML struct {
	ExportStrategy
}

func (exportStrategyEML ExportStrategyEML) Name() string {
	return "eml"
}

func (exportStrategyEML ExportStrategyEML) Export(pstFile *pst.File, pstMessage pst.Message, pstMessageIndex int, pstFolder pst.Folder, pstFormatType string, pstEncryptionType string, exportContext ExportContext) error {
	outputDirectory := filepath.Join(exportContext.OutputDirectory, pstFolder.DisplayName)

	if err := os.MkdirAll(outputDirectory, 0755); err != nil {
		return err
	}

	outputFile, err := os.Create(filepath.Join(outputDirectory, strconv.Itoa(pstMessageIndex) + ".eml"))

	if err != nil {
		return err
	}

	header, err := pstMessage.GetHeaders(pstFile, pstFormatType, pstEncryptionType)

	if err != nil {
		return err
	}

	var body string
	isHTMLBody := false

	if exportContext.IsOnlyPlaintextBody {
		pstBody, err := pstMessage.GetBody(pstFile, pstFormatType, pstEncryptionType)

		if err == nil {
			body = pstBody
		} else {
			Logger.Errorf("Failed to get plaintext body from message: %s", err)
		}
	} else {
		pstBodyHTML, err := pstMessage.GetBodyHTML(pstFile, pstFormatType, pstEncryptionType)

		if err == nil {
			body = pstBodyHTML
			isHTMLBody = true
		} else {
			pstBody, err := pstMessage.GetBody(pstFile, pstFormatType, pstEncryptionType)

			if err == nil {
				body = pstBody
			} else {
				Logger.Errorf("Failed to get body from message: %s", err)
			}
		}
	}

	if len(body) == 0 {
		Logger.Errorf("Empty message body for message: #%d", pstMessageIndex)
	}

	emlMessage, err := message.Read(strings.NewReader(header))

	if err != nil {
		emlMessage, err = fixHeaderEncodingIssues(header, err, 10)

		if err != nil {
			return err
		}
	}

	// Bypass go-message unhandled charset.
	// createWriter (WriteTo) only allows UTF-8 and US-ASCII.
	// See https://github.com/emersion/go-message/issues/127
	emlMessage.Header.Set("Content-Type", "utf-8")

	emlWriter, err := mail.CreateWriter(outputFile, mail.NewReader(emlMessage).Header)

	if err != nil {
		return err
	}

	pstMessageAttachments, err := pstMessage.GetAttachments(pstFile, pstFormatType, pstEncryptionType)

	if err != nil {
		Logger.Warnf("Failed to get message attachments: %s", err)
	} else {
		for _, attachment := range pstMessageAttachments {
			var attachmentHeader mail.AttachmentHeader

			if attachmentName, err := attachment.GetLongFilename(); err == nil {
				attachmentHeader.SetFilename(attachmentName)
			} else {
				if attachmentName, err := attachment.GetFilename(); err == nil {
					attachmentHeader.SetFilename(attachmentName)
				} else {
					Logger.Warnf("Failed to get attachment name, skipping...")
					continue
				}
			}

			attachmentInputStream, err := attachment.GetInputStream(pstFile, pstFormatType, pstEncryptionType)

			if err != nil {
				Logger.Warnf("Failed to get attachment input stream, skipping: %s", err)
				continue
			}

			attachmentData, err := attachmentInputStream.ReadCompletely()

			if err != nil {
				Logger.Warnf("Failed to read attachment input stream, skipping: %s", err)
				continue
			}

			attachmentWriter, err := emlWriter.CreateAttachment(attachmentHeader)

			if err != nil {
				Logger.Warnf("Failed to create attachment, skipping: %s", err)
				continue
			}

			_, err = io.Copy(attachmentWriter, bytes.NewReader(attachmentData))

			if err != nil {
				Logger.Warnf("Failed to copy attachment data, skipping: %s", err)
				continue
			}

			if err := attachmentWriter.Close(); err != nil {
				Logger.Warnf("Failed to close attachment writer: %s", err)
			}
		}
	}

	inlineWriter, err := emlWriter.CreateInline()

	if err != nil {
		return err
	}

	var inlineHeader mail.InlineHeader

	if isHTMLBody {
		inlineHeader.Set("Content-Type", "text/html")
	} else {
		inlineHeader.Set("Content-Type", "text/plain")
	}

	bodyWriter, err := inlineWriter.CreatePart(inlineHeader)

	if err != nil {
		return err
	}

	_, err = bodyWriter.Write([]byte(body))

	if err != nil {
		return err
	}

	return outputFile.Close()
}

func fixHeaderEncodingIssues(header string, emlError error, maxRetries int) (*message.Entity, error) {
	if maxRetries <= 0 {
		return nil, errors.New("failed to fix encoding issues")
	}

	var headerBuilder strings.Builder

	if strings.Contains(emlError.Error(), "malformed MIME header key") {
		// The header encoding is incorrect from go-pst, remove ASCII NUL and replacement characters.
		for _, r := range []rune(header) {
			if int(r) != 0 && r != unicode.ReplacementChar {
				headerBuilder.WriteRune(r)
			}
		}

		// Try again, it may be valid now.
		emlMessage, err := message.Read(strings.NewReader(headerBuilder.String()))

		if err == nil {
			// Solved it.
			return emlMessage, nil
		} else if err.Error() == emlError.Error() {
			// Didn't solve it.
			return nil, err
		} else {
			// Different error, call ourselves again to try fix the malformed lines.
			emlMessage, err := fixHeaderEncodingIssues(headerBuilder.String(), err, maxRetries)

			if err != nil {
				return nil, err
			}

			return emlMessage, nil
		}
	} else if strings.Contains(emlError.Error(), "malformed MIME header line") {
		// Let's just remove this line and log it, maybe we can recover.
		malformedLine := strings.Replace(emlError.Error(), "message: malformed MIME header line: ", "", -1)
		malformedLine = strings.Replace(malformedLine, "\n", "", -1)
		malformedLine = strings.Replace(malformedLine, "\r", "", -1)
		scanner := bufio.NewScanner(strings.NewReader(header))

		for scanner.Scan() {
			if scanner.Text() != malformedLine {
				headerBuilder.WriteString(scanner.Text() + "\n")
			} else {
				Logger.Warnf("Removing malformed MINE header line: %s", malformedLine)
			}
		}

		// Try again, it may be valid now.
		emlMessage, err := message.Read(strings.NewReader(headerBuilder.String()))

		if err == nil {
			// Solved it.
			return emlMessage, nil
		} else if err.Error() == emlError.Error() {
			// Same error, let's remove the next malformed line.
			emlMessage, err := fixHeaderEncodingIssues(headerBuilder.String(), err, maxRetries - 1)

			if err != nil {
				return nil, err
			}

			return emlMessage, nil
		}
	}

	// Can't handle this error.
	return nil, emlError
}