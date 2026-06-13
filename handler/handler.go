package handler

import (
	"fmt"
	"quickfix-lsp/quickfix"
	"quickfix-lsp/symbols"
	"regexp"
	"strings"
	"sync"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const (
	NAME    = "no-lsp"
	VERSION = "0.1"
)

type Handler struct {
	protocol.Handler

	symbols  symbols.Symbols
	quickfix quickfix.Quickfix
}

func New() *Handler {
	h := &Handler{}

	h.Handler = protocol.Handler{
		Initialize: func(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
			return h.initialize(context, params)
		},
		Initialized: func(context *glsp.Context, params *protocol.InitializedParams) error {
			return nil
		},
		Shutdown: func(context *glsp.Context) error {
			return nil
		},
		TextDocumentDidOpen: func(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
			return h.didOpen(context, params)
		},
		TextDocumentDidSave: func(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
			return h.didSave(context, params)
		},
		TextDocumentReferences: func(
			ctx *glsp.Context,
			params *protocol.ReferenceParams,
		) ([]protocol.Location, error) {
			return h.references(ctx, params)
		},
	}

	return h
}

func (h *Handler) references(
	_ *glsp.Context,
	params *protocol.ReferenceParams,
) ([]protocol.Location, error) {
	uri := string(params.TextDocument.URI)
	line := uint32(params.Position.Line)
	char := uint32(params.Position.Character)

	return h.symbols.Locations(uri, line, char)
}

func (h *Handler) diagnosticsUpdate(context *glsp.Context) {
	h.quickfix.Update()
	for k, v := range h.quickfix.Diagnostics {
		go context.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
			Diagnostics: v,
			URI:         k,
		})
	}
}

func (h *Handler) didSave(context *glsp.Context, _ *protocol.DidSaveTextDocumentParams) error {
	h.diagnosticsUpdate(context)
	h.symbols.Reindex()
	return nil
}

func (h *Handler) didOpen(context *glsp.Context, _ *protocol.DidOpenTextDocumentParams) error {
	h.diagnosticsUpdate(context)
	return nil
}

func (h *Handler) initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	capabilities := h.CreateServerCapabilities()

	if params.InitializationOptions == nil {
		return nil, fmt.Errorf("no config provided")
	}

	cfg, err := loadConfig(params.InitializationOptions)
	if err != nil {
		return nil, fmt.Errorf("config loading error: %v", err)
	}

	var wg sync.WaitGroup

	wg.Go(func() {
		regexpSources, err := regexp.Compile(cfg.RegexpSources)
		if err != nil {
			return
		}
		h.symbols = symbols.New(cfg.Sources, regexpSources)

		h.symbols.Reindex()
	})
	wg.Go(func() {
		regexpError, err := regexp.Compile(cfg.RegexpError)
		if err != nil {
			return
		}
		argsRaw := strings.Fields(cfg.Build)
		if len(argsRaw) < 1 {
			return
		}

		build := argsRaw[0]
		buildArgs := argsRaw[1:]
		h.quickfix = quickfix.New(NAME, regexpError, build, buildArgs)
		h.diagnosticsUpdate(context)
	})
	wg.Wait()

	version := VERSION

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    NAME,
			Version: &version,
		},
	}, nil
}
