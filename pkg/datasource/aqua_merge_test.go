package datasource

import (
	"testing"

	"github.com/aquaproj/aqua/v2/pkg/config/registry"
)

func TestMergeVersionOverride_Asset(t *testing.T) {
	pkg := registry.PackageInfo{Asset: "default-asset"}
	vo := registry.VersionOverride{Asset: "override-asset"}
	got := mergeVersionOverride(pkg, vo)
	if got.Asset != "override-asset" {
		t.Errorf("Asset: got %q, want %q", got.Asset, "override-asset")
	}
}

func TestMergeVersionOverride_Format(t *testing.T) {
	pkg := registry.PackageInfo{Format: "tar.gz"}
	vo := registry.VersionOverride{Format: "zip"}
	got := mergeVersionOverride(pkg, vo)
	if got.Format != "zip" {
		t.Errorf("Format: got %q, want %q", got.Format, "zip")
	}
}

func TestMergeVersionOverride_Files(t *testing.T) {
	pkg := registry.PackageInfo{Files: []*registry.File{{Name: "default"}}}
	vo := registry.VersionOverride{Files: []*registry.File{{Name: "override"}}}
	got := mergeVersionOverride(pkg, vo)
	want := []*registry.File{{Name: "override"}}
	if len(got.Files) != len(want) || got.Files[0].Name != want[0].Name {
		t.Errorf("Files: got %+v, want %+v", got.Files, want)
	}
}

func TestMergeVersionOverride_SupportedEnvs(t *testing.T) {
	pkg := registry.PackageInfo{SupportedEnvs: []string{"linux/amd64"}}
	vo := registry.VersionOverride{SupportedEnvs: []string{"darwin/amd64"}}
	got := mergeVersionOverride(pkg, vo)
	want := []string{"darwin/amd64"}
	if len(got.SupportedEnvs) != len(want) || got.SupportedEnvs[0] != want[0] {
		t.Errorf("SupportedEnvs: got %+v, want %+v", got.SupportedEnvs, want)
	}
}

func TestMergeVersionOverride_Checksum(t *testing.T) {
	checksum := &registry.Checksum{Asset: "override-checksum.txt"}
	pkg := registry.PackageInfo{}
	vo := registry.VersionOverride{Checksum: checksum}
	got := mergeVersionOverride(pkg, vo)
	if got.Checksum != checksum {
		t.Errorf("Checksum: got %+v, want %+v", got.Checksum, checksum)
	}
}

func TestMergeVersionOverride_FormatOverrides(t *testing.T) {
	fo := registry.FormatOverrides{{GOOS: "windows", Format: "zip"}}
	pkg := registry.PackageInfo{}
	vo := registry.VersionOverride{FormatOverrides: fo}
	got := mergeVersionOverride(pkg, vo)
	if len(got.FormatOverrides) != len(fo) || got.FormatOverrides[0].GOOS != fo[0].GOOS || got.FormatOverrides[0].Format != fo[0].Format {
		t.Errorf("FormatOverrides: got %+v, want %+v", got.FormatOverrides, fo)
	}
}

func TestMergeVersionOverride_Overrides(t *testing.T) {
	ov := registry.Overrides{{GOOS: "linux", Format: "tar.gz"}}
	pkg := registry.PackageInfo{}
	vo := registry.VersionOverride{Overrides: ov}
	got := mergeVersionOverride(pkg, vo)
	if len(got.Overrides) != len(ov) || got.Overrides[0].GOOS != ov[0].GOOS || got.Overrides[0].Format != ov[0].Format {
		t.Errorf("Overrides: got %+v, want %+v", got.Overrides, ov)
	}
}
