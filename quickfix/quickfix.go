package quickfix

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	// TODO: make quickfix package completely glsp-independent.
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Quickfix struct {
	Diagnostics map[string][]protocol.Diagnostic

	source      string
	build       string
	buildArgs   []string
	regexpError *regexp.Regexp
}

func New(source string, regexpError *regexp.Regexp, build string, buildArgs []string) Quickfix {
	return Quickfix{
		source:      source,
		build:       build,
		buildArgs:   buildArgs,
		regexpError: regexpError,
		Diagnostics: make(map[string][]protocol.Diagnostic),
	}
}

func (q *Quickfix) Update() {
	cmd := exec.Command(q.build, q.buildArgs...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run()

	parsedDiag := q.parseErrors(stderr.String())

	for k := range q.Diagnostics {
		q.Diagnostics[k] = []protocol.Diagnostic{}
	}

	for _, d := range parsedDiag {
		uri := fmt.Sprintf("file:///%v", d.Path)

		var ok bool

		if _, ok = q.Diagnostics[uri]; !ok {
			q.Diagnostics[uri] = []protocol.Diagnostic{}
		}
		q.Diagnostics[uri] = append(q.Diagnostics[uri], toDiagnostic(d))
	}
}

func (q *Quickfix) parseErrors(output string) []parsedDiagnostic {
	var result []parsedDiagnostic

	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)

		match := q.regexpError.FindStringSubmatch(line)

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

		result = append(result, parsedDiagnostic{
			Source:   q.source,
			Path:     filepath.Clean(match[1]),
			Line:     lineNum,
			Column:   colNum,
			Severity: protocol.DiagnosticSeverityError,
			Message:  match[4],
		})
	}

	return result
}

type parsedDiagnostic struct {
	Source   string
	Path     string
	Line     int
	Column   int
	Severity protocol.DiagnosticSeverity
	Message  string
}

func toDiagnostic(d parsedDiagnostic) protocol.Diagnostic {
	severity := protocol.DiagnosticSeverityError

	return protocol.Diagnostic{
		Message:  d.Message,
		Source:   &d.Source,
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
