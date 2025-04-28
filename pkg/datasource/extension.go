package datasource

import "github.com/apex/log"

// formatToExtension converts a goreleaser archive format to a file extension.
func formatToExtension(format string) string {
	switch format {
	case "tar.gz":
		return ".tar.gz"
	case "tgz":
		return ".tgz" // Alias for tar.gz
	case "tar.xz":
		return ".tar.xz"
	case "tar":
		return ".tar"
	case "zip":
		return ".zip"
	case "gz":
		return ".gz" // Less common for archives, but possible
	case "binary", "raw", "":
		return "" // No extension for binary format
	default:
		log.Warnf("unknown archive format '%s', assuming no extension", format)
		return ""
	}
}
