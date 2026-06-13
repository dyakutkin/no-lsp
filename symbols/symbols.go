package symbols

import (
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"text/scanner"
	"unicode/utf8"

	// TODO: make symbols package completely glsp-independent.
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const MAX_LOCATIONS_ENTRIES = 256

type Symbols struct {
	dirs            []string
	regexpSources   *regexp.Regexp
	tokensIndex     map[string][]protocol.Location
	tokensLineIndex map[lineKey][]tokensLineIndexEntry
}

type lineKey struct {
	uri  string
	line uint32
}

type tokensLineIndexEntry struct {
	token string
	r     protocol.Range
}

func New(dirs []string, regexp *regexp.Regexp) Symbols {
	return Symbols{
		dirs:            dirs,
		regexpSources:   regexp,
		tokensIndex:     make(map[string][]protocol.Location),
		tokensLineIndex: make(map[lineKey][]tokensLineIndexEntry),
	}
}

func (r *Symbols) Locations(
	uri string, line uint32, char uint32,
) ([]protocol.Location, error) {
	key := lineKey{
		uri:  uri,
		line: line,
	}

	lineEntries, ok := r.tokensLineIndex[key]
	if !ok {
		return nil, nil
	}

	for _, entry := range lineEntries {
		if char >= entry.r.Start.Character && char < entry.r.End.Character {
			loc := r.tokensIndex[entry.token]
			if len(loc) > MAX_LOCATIONS_ENTRIES {
				return r.tokensIndex[entry.token][:MAX_LOCATIONS_ENTRIES], nil
			}
			return r.tokensIndex[entry.token], nil
		}
	}

	return nil, nil
}

func (r *Symbols) Reindex() {
	// TODO: consider not reindexing everything every time (e.g. sources probably can only be indexed once at initialization stage).
	clear(r.tokensIndex)
	clear(r.tokensLineIndex)

	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	files, _ := listFiles(cwd)

	for _, src := range r.dirs {
		if sFiles, err := listFiles(src); err == nil {
			files = append(files, sFiles...)
		}
	}

	for _, p := range files {
		if r.regexpSources != nil && !r.regexpSources.MatchString(path.Base(p)) {
			continue
		}
		func() {
			f, err := os.Open(p)
			if err != nil {
				return
			}
			defer f.Close()

			var s scanner.Scanner
			s.Init(f)
			s.Filename = p

			uri := string(pathToURI(p))

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

				r.tokensLineIndex[key] = append(r.tokensLineIndex[key], tokensLineIndexEntry{
					token: text,
					r:     rng,
				})

				loc := protocol.Location{
					URI:   protocol.DocumentUri(uri),
					Range: rng,
				}

				r.tokensIndex[text] = append(r.tokensIndex[text], loc)
			}
		}()
	}
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

func pathToURI(p string) protocol.DocumentUri {
	abs, _ := filepath.Abs(p)
	abs = filepath.ToSlash(abs)

	// Windows path handling.
	if len(abs) >= 2 && abs[1] == ':' {
		abs = "/" + abs
	}

	u := url.URL{
		Scheme: "file",
		Path:   abs,
	}

	return protocol.DocumentUri(u.String())
}
