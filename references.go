package main

import (
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"text/scanner"
	"time"
	"unicode/utf8"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type lineKey struct {
	uri  string
	line uint32
}

type tokensLineIndexEntry struct {
	token string
	r     protocol.Range
}

var (
	tokensIndex     map[string][]protocol.Location
	tokensLineIndex map[lineKey][]tokensLineIndexEntry
)

func init() {
	tokensIndex = make(map[string][]protocol.Location)
	tokensLineIndex = make(map[lineKey][]tokensLineIndexEntry)
}

func toURI(p string) protocol.DocumentUri {
	abs, _ := filepath.Abs(p)
	abs = filepath.ToSlash(abs)

	// fix Windows drive paths
	if len(abs) >= 2 && abs[1] == ':' {
		abs = "/" + abs
	}

	u := url.URL{
		Scheme: "file",
		Path:   abs,
	}

	return protocol.DocumentUri(u.String())
}

func references(
	ctx *glsp.Context,
	params *protocol.ReferenceParams,
) ([]protocol.Location, error) {
	uri := string(params.TextDocument.URI)
	line := uint32(params.Position.Line)
	char := uint32(params.Position.Character)

	key := lineKey{
		uri:  uri,
		line: line,
	}

	lineEntries, ok := tokensLineIndex[key]
	if !ok {
		return nil, nil
	}

	for _, entry := range lineEntries {
		if char >= entry.r.Start.Character && char < entry.r.End.Character {
			return tokensIndex[entry.token], nil
		}
	}

	return nil, nil
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

func isBinaryFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer f.Close()

	buf := make([]byte, 4096)
	n, _ := f.Read(buf)
	buf = buf[:n]

	// NUL byte heuristic (standard in ripgrep, git, etc.)
	return slices.Contains(buf, 0)
}

func reindexSymbols() {
	start := time.Now()

	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	files, _ := listFiles(cwd)
	log.Errorf("files: %v", files)

	for _, src := range config.Sources {
		log.Errorf("sources is empty")
		if sFiles, err := listFiles(src); err == nil {
			files = append(files, sFiles...)
		}
	}

	for _, p := range files {
		if isBinaryFile(p) {
			continue
		}
		func() {
			f, err := os.Open(p)
			if err != nil {
				log.Errorf("open error: %v", err)
				return
			}
			defer f.Close()

			var s scanner.Scanner
			s.Init(f)
			s.Filename = p

			uri := string(toURI(p))

			for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
				if tok != scanner.Ident {
					continue
				}

				text := s.TokenText()
				pos := s.Position

				startLine := uint32(pos.Line - 1)
				startChar := uint32(pos.Column - 1)
				endChar := startChar + uint32(utf8.RuneCountInString(text))

				rng := protocol.Range{
					Start: protocol.Position{
						Line:      startLine,
						Character: startChar,
					},
					End: protocol.Position{
						Line:      startLine,
						Character: endChar,
					},
				}

				key := lineKey{
					uri:  uri,
					line: startLine,
				}

				tokensLineIndex[key] = append(tokensLineIndex[key], tokensLineIndexEntry{
					token: text,
					r:     rng,
				})

				loc := protocol.Location{
					URI:   protocol.DocumentUri(uri),
					Range: rng,
				}

				tokensIndex[text] = append(tokensIndex[text], loc)
			}
		}()
	}

	elapsed := time.Since(start)
	log.Errorf("reindex took %s", elapsed)
}
