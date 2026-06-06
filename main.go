package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"

	_ "github.com/tliron/commonlog/simple"
)

const NAME string = "no-lsp"

var (
	version string = "0.1"
	handler protocol.Handler

	config        Config
	workspaceRoot string

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

type Config struct {
	BuildCmd    string `json:"build"`
	RegexpError string `json:"re_error"`
	RegexpWarn  string `json:"re_warn"`
}

func parseOdinErrors(output string) []ParsedDiagnostic {
	var result []ParsedDiagnostic

	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)

		match := regexpError.FindStringSubmatch(line)
		if match == nil {
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

func update(context *glsp.Context) {
	cmd := exec.Command(buildCmd, buildCmdArgs...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Run()

	parsedDiag := parseOdinErrors(stderr.String())

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

func didSave(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	update(context)
	return nil

}

func didOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	update(context)
	return nil
}

func main() {
	commonlog.Configure(-2, nil)
	filesDiagnosed = make(map[string][]protocol.Diagnostic)

	handler = protocol.Handler{
		Initialize:          initialize,
		Shutdown:            shutdown,
		SetTrace:            setTrace,
		TextDocumentDidOpen: didOpen,
		TextDocumentDidSave: didSave,
	}

	server := server.NewServer(&handler, NAME, false)

	server.RunStdio()
}

func initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	capabilities := handler.CreateServerCapabilities()

	if len(params.WorkspaceFolders) > 0 {
		workspaceRoot = params.WorkspaceFolders[0].Name
	}

	if params.InitializationOptions != nil {
		b, err := json.Marshal(params.InitializationOptions)
		if err == nil {
			if err := json.Unmarshal(b, &config); err != nil {
				panic(fmt.Sprintf("failed to parse config: %v", err))
			}
		}

		// TODO: proper whitespace instead of relying on a hardcoded one.
		argsRaw := strings.Split(config.BuildCmd, " ")
		if len(argsRaw) >= 1 {
			buildCmd = argsRaw[0]
			buildCmdArgs = argsRaw[1:]
			regexpError = *regexp.MustCompile(config.RegexpError)
			regexpWarn = *regexp.MustCompile(config.RegexpWarn)
		}
	}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    NAME,
			Version: &version,
		},
	}, nil
}

func shutdown(context *glsp.Context) error {
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}
