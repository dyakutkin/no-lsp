package main

import (
	"no-lsp/handler"

	"github.com/tliron/commonlog"
	"github.com/tliron/glsp/server"

	_ "github.com/tliron/commonlog/simple"
)

func main() {
	commonlog.Configure(-2, nil)

	h := handler.New()
	srv := server.NewServer(h, handler.NAME, false)

	if err := srv.RunStdio(); err != nil {
		panic(err)
	}
}
