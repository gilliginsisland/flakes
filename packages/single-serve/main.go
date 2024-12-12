package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"single-serve/stdio"
)

type SingleFileHandler struct {
	path string
}

func (hndlr *SingleFileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Open the file
	file, err := os.Open(hndlr.path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error opening file: %s", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Get the file information
	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "Error getting file info", http.StatusInternalServerError)
		return
	}

	// Set the content type based on the file extension (optional)
	if len(os.Args) == 3 {
		w.Header().Set("Content-Type", os.Args[2])
	}

	// Serve the file contents
	http.ServeContent(w, r, fileInfo.Name(), fileInfo.ModTime(), file)
}

func main() {
	if n := len(os.Args); n < 2 || n > 3 {
		log.Fatalf("usage: %s <file> [mime-type]", os.Args[0])
	}

	srv := &http.Server{
		Handler: &SingleFileHandler{path: os.Args[1]},
	}
	srv.Serve(stdio.Listener())
	srv.Shutdown(context.Background())
}
