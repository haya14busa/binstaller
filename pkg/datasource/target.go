package datasource

import "slices"

func isValidTarget(goos, goarch string) bool {
	return slices.Contains(validTargets, goos+goarch)
}

// lists from https://go.dev/doc/install/source#environment
var validTargets = []string{
	"aixppc64",
	"android386",
	"androidamd64",
	"androidarm",
	"androidarm64",
	"darwinamd64",
	"darwinarm64",
	"dragonflyamd64",
	"freebsd386",
	"freebsdamd64",
	"freebsdarm",
	"freebsdarm64",
	"illumosamd64",
	"iosarm64",
	"jswasm",
	"wasip1wasm",
	"linux386",
	"linuxamd64",
	"linuxarm",
	"linuxarm64",
	"linuxppc64",
	"linuxppc64le",
	"linuxmips",
	"linuxmipsle",
	"linuxmips64",
	"linuxmips64le",
	"linuxs390x",
	"linuxriscv64",
	"linuxloong64",
	"netbsd386",
	"netbsdamd64",
	"netbsdarm",
	"netbsdarm64",
	"openbsd386",
	"openbsdamd64",
	"openbsdarm",
	"openbsdarm64",
	"plan9386",
	"plan9amd64",
	"plan9arm",
	"solarisamd64",
	"solarissparc",
	"solarissparc64",
	"windowsarm",
	"windowsarm64",
	"windows386",
	"windowsamd64",
}
