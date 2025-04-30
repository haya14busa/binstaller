package datasource

import (
	"strings"

	"github.com/apex/log"
)

var extensions = []string{
	"tar.br",
	"tar.bz2",
	"tar.gz",
	"tar.lz4",
	"tar.sz",
	"tar.xz",
	"tbr",
	"tbz",
	"tbz2",
	"tgz",
	"tlz4",
	"tsz",
	"txz",
	"tar.zst",
	"zip",
	"gz",
	"bz2",
	"lz4",
	"sz",
	"xz",
	"zst",
	"dmg",
	"pkg",
	"rar",
	"tar",
}

// formatToExtension converts a goreleaser archive format to a file extension.
func formatToExtension(format string) string {
	for _, e := range extensions {
		if e == format {
			return "." + e
		}
	}
	// Handle other special format
	switch format {
	case "binary", "raw", "":
		return "" // No extension for binary format
	default:
		log.Warnf("unknown archive format '%s'", format)
		return "." + strings.TrimLeft(format, ".")
	}
}

func hasExtensions(s string) bool {
	for _, e := range extensions {
		if strings.HasSuffix(s, "."+e) {
			return true
		}
	}
	return false
}

func extractExtension(s string) string {
	for _, e := range extensions {
		ext := "." + e
		if strings.HasSuffix(s, ext) {
			return ext
		}
	}
	return ""
}

func trimExtension(s string) string {
	for _, e := range extensions {
		ext := "." + e
		if strings.HasSuffix(s, ext) {
			return strings.TrimSuffix(s, ext)
		}
	}
	return s
}
