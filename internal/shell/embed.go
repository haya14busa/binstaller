package shell

import _ "embed"

// mainScriptTemplate is the main body of the installer script.
// It performs runtime detection and resolution.
//
//go:embed template.sh
var mainScriptTemplate string

//go:embed hash_sha256.sh
var hashSHA256 string

//go:embed hash_sha1.sh
var hashSHA1 string

// shellFunctions contains the library of POSIX shell functions.
// Adapted from https://github.com/client9/shlib
//
//go:embed shlib.sh
var shellFunctions string
