package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"text/scanner"
	"unicode/utf8"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	tokensIndex map[string][]protocol.Location
)

func references(
	ctx *glsp.Context,
	params *protocol.ReferenceParams,
) ([]protocol.Location, error) {
	uri := params.TextDocument.URI
	pos := params.Position

	var target string

out:
	for tok, locEntries := range tokensIndex {
		for _, loc := range locEntries {
			r := loc.Range
			log.Errorf("requested %+v, comparing with %v at %+v\n", params.Position, tok, loc.Range)
			if loc.URI == uri &&
				pos.Line == r.Start.Line &&
				pos.Character >= r.Start.Character &&
				pos.Character <= r.End.Character {

				target = tok
				break out
			}
		}
	}

	if target == "" {
		return nil, nil
	}

	return tokensIndex[target], nil
}

func listFiles(root string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

func parseSourceTokens() {
	files, _ := listFiles(".")

	if config.Sources != "" {
		// log.Errorf("!parseresult empty source path")
		files = append(files, config.Sources)
	}

	for _, p := range files {
		f, err := os.Open(p)
		if err != nil {
			log.Errorf("!parseresult open error: %v", err)
		}
		defer f.Close()

		var s scanner.Scanner
		s.Init(f)
		s.Filename = p

		uri := protocol.DocumentUri("file://" + filepath.ToSlash(p))

		for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
			text := s.TokenText()

			if tok != scanner.Ident {
				continue
			}

			pos := s.Position

			startLine := uint32(pos.Line - 1)
			startChar := uint32(pos.Column - 1)
			// startLine := uint32(pos.Line)
			// startChar := uint32(pos.Column)

			endChar := startChar + uint32(utf8.RuneCountInString(text))

			loc := protocol.Location{
				URI: uri,
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      startLine,
						Character: startChar,
					},
					End: protocol.Position{
						Line:      startLine,
						Character: endChar,
					},
				},
			}

			tokensIndex[text] = append(tokensIndex[text], loc)
		}
	}
}
