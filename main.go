package main

import (
	"encoding/json"
	"fmt"
	"regexp"
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
	log     = commonlog.GetLogger(NAME)

	config        Config
	workspaceRoot string
)

type Config struct {
	BuildCmd    string `json:"build"`
	RegexpError string `json:"re_error"`
	RegexpWarn  string `json:"re_warn"`
	Sources     string `json:"sources"`
}

func main() {
	commonlog.Configure(-2, nil)
	filesDiagnosed = make(map[string][]protocol.Diagnostic)

	handler = protocol.Handler{
		Initialize:             initialize,
		Initialized:            initialized,
		Shutdown:               shutdown,
		SetTrace:               setTrace,
		TextDocumentDidOpen:    didOpen,
		TextDocumentDidSave:    didSave,
		TextDocumentReferences: references,
	}

	server := server.NewServer(&handler, NAME, false)

	server.RunStdio()
}

func initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	tokensIndex = make(map[string][]protocol.Location)

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

	parseSourceTokens()

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    NAME,
			Version: &version,
		},
	}, nil
}

func initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	return nil
}

func shutdown(context *glsp.Context) error {
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}
