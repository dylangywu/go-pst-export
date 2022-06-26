// Package pstexport
// This file is part of go-pst-export (https://github.com/mooijtech/go-pst-export)
// Copyright (C) 2022 Marten Mooij (https://www.mooijtech.com/)
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

func (exportStrategyEML ExportStrategyEML) Export(pstFile *pst.File, message pst.Message, messageIndex int, folder pst.Folder, formatType string, encryptionType string, exportContext ExportContext) error {
	outputDirectory := filepath.Join(exportContext.OutputDirectory, folder.DisplayName)

	if err := os.MkdirAll(outputDirectory, 0755); err != nil {
		return err
	}

	return exportToEML(pstFile, message, messageIndex, formatType, encryptionType, outputDirectory)
}

func exportToEML(pstFile *pst.File, pstMessage pst.Message, messageIndex int, formatType string, encryptionType string, outputDirectory string) error {
	outputFile, err := os.Create(filepath.Join(outputDirectory, strconv.Itoa(messageIndex) + ".eml"))

	if err != nil {
		return err
	}

	header, err := pstMessage.GetHeaders(pstFile, formatType, encryptionType)

	if err != nil {
		return err
	}

	var body string

	pstBodyHTML, err := pstMessage.GetBodyHTML(pstFile, formatType, encryptionType)

	if err == nil {
		body = pstBodyHTML
	} else {
		pstBody, err := pstMessage.GetBody(pstFile, formatType, encryptionType)

		if err == nil {
			body = pstBody
		} else {
			Logger.Error("Failed to get body from message (maybe the body is RTF?).")
		}
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

	pstMessageAttachments, err := pstMessage.GetAttachments(pstFile, formatType, encryptionType)

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

			attachmentInputStream, err := attachment.GetInputStream(pstFile, formatType, encryptionType)

			if err != nil {
				Logger.Warnf("Failed to get attachment input stream, skipping...")
				continue
			}

			attachmentData, err := attachmentInputStream.ReadCompletely()

			if err != nil {
				Logger.Warnf("Failed to read attachment input stream, skipping...")
				continue
			}

			attachmentWriter, err := emlWriter.CreateAttachment(attachmentHeader)

			if err != nil {
				Logger.Warnf("Failed to create attachment, skipping...")
				continue
			}

			_, err = io.Copy(attachmentWriter, bytes.NewReader(attachmentData))

			if err != nil {
				Logger.Warnf("Failed to copy attachment data, skipping...")
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

	inlineHeader.Set("Content-Type", "text/plain")

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