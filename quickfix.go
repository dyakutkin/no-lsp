package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	filesDiagnosed map[string][]protocol.Diagnostic

	buildCmd     string
	buildCmdArgs []string
	regexpError  regexp.Regexp
	regexpWarn   regexp.Regexp
)

type ParsedDiagnostic struct {
	Path     string
	Line     int
	Column   int
	Severity protocol.DiagnosticSeverity
	Message  string
}

func toDiagnostic(d ParsedDiagnostic) protocol.Diagnostic {
	source := NAME
	severity := protocol.DiagnosticSeverityError

	return protocol.Diagnostic{
		Message:  d.Message,
		Source:   &source,
		Severity: &severity,
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(d.Line - 1),
				Character: uint32(d.Column - 1),
			},
			End: protocol.Position{
				Line:      uint32(d.Line - 1),
				Character: uint32(d.Column - 1),
			}},
	}
}

func updateDiagnostics(context *glsp.Context) {
	cmd := exec.Command(buildCmd, buildCmdArgs...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Run()

	parsedDiag := parseErrors(stderr.String())

	for k := range filesDiagnosed {
		filesDiagnosed[k] = []protocol.Diagnostic{}
	}

	for _, d := range parsedDiag {
		uri := fmt.Sprintf("file:///%v", d.Path)

		var ok bool

		if _, ok = filesDiagnosed[uri]; !ok {
			filesDiagnosed[uri] = []protocol.Diagnostic{}
		}
		filesDiagnosed[uri] = append(filesDiagnosed[uri], toDiagnostic(d))
	}

	for k, v := range filesDiagnosed {
		go context.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
			Diagnostics: v,
			URI:         k,
		})
	}
}

func didOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	updateDiagnostics(context)
	return nil
}

func parseErrors(output string) []ParsedDiagnostic {
	var result []ParsedDiagnostic

	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)

		match := regexpError.FindStringSubmatch(line)
		if match == nil || len(match) != 5 {
			continue
		}

		lineNum, err := strconv.Atoi(match[2])
		if err != nil {
			continue
		}

		colNum, err := strconv.Atoi(match[3])
		if err != nil {
			continue
		}

		result = append(result, ParsedDiagnostic{
			Path:     filepath.Clean(match[1]),
			Line:     lineNum,
			Column:   colNum,
			Severity: protocol.DiagnosticSeverityError,
			Message:  match[4],
		})
	}

	return result
}
