package main

//used
// embbeds REACT APP, middleware

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"time"
)

var addr = flag.String("addr", ":8080", "specify the port of the server")

// middleware
type Logger struct {
	handler http.Handler
}

func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	l.handler.ServeHTTP(w, r)
	log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
}

func NewLogger(handlerToWrap http.Handler) *Logger {
	return &Logger{handlerToWrap}
}

//go:embed frontend/dist/*
var reactApp embed.FS

func main() {
	flag.Parse()
	mux := http.NewServeMux()
	dist, err := fs.Sub(reactApp,"frontend/dist")

	if err != nil {
		panic(err)
	}
	frontend:=http.FileServer(http.FS(dist))
	mux.Handle("GET /", frontend)
	wrappedmux := NewLogger(mux)
	http.ListenAndServe(*addr, wrappedmux)
}
