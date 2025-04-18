package main

import (
	"fmt"
)

// AttestationOptions contains options for attestation verification
type AttestationOptions struct {
	EnableGHAttestation      bool
	RequireAttestation       bool
	GHAttestationVerifyFlags string
	SkipSourceInfo           bool
}

// processSource processes the source and returns the generated shell script.
func processSource(source, repo, path, file string, attestationOpts AttestationOptions, commitHash string) (out []byte, err error) {
	switch source {
	case "godownloader":
		// https://github.com/goreleaser/godownloader
		out, err = processGodownloader(repo, path, file, attestationOpts, commitHash)
	default:
		return nil, fmt.Errorf("unknown source %q", source)
	}
	return
}
