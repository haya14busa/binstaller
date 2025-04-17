package main

import (
	"fmt"
)

// processSource processes the source and returns the generated shell script.
func processSource(source, repo, path, file string) (out []byte, err error) {
	switch source {
	case "godownloader":
		// https://github.com/goreleaser/godownloader
		out, err = processGodownloader(repo, path, file)
	default:
		return nil, fmt.Errorf("unknown source %q", source)
	}
	return
}
