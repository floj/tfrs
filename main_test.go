package main

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestParseStateList(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		maxDepth int
		want     []string
	}{
		{
			name:     "top level resources only",
			output:   "aws_instance.web\naws_s3_bucket.data\n",
			maxDepth: 0,
			want:     []string{"aws_instance.web", "aws_s3_bucket.data"},
		},
		{
			name:     "module extracted from nested resource, depth 0 hides children",
			output:   "module.vpc.aws_vpc.this\n",
			maxDepth: 0,
			want:     []string{"module.vpc"},
		},
		{
			name:     "module and its resource visible at depth 1",
			output:   "module.vpc.aws_vpc.this\n",
			maxDepth: 1,
			want:     []string{"module.vpc", "module.vpc.aws_vpc.this"},
		},
		{
			name:     "nested submodule chain",
			output:   "module.a.module.b.aws_instance.x\n",
			maxDepth: 2,
			want:     []string{"module.a", "module.a.module.b", "module.a.module.b.aws_instance.x"},
		},
		{
			// NOTE: the dots/2 depth heuristic treats "data.type.name" (3
			// segments) as depth 1, so a *top-level* data source only shows
			// up from --depth 1 onwards. Documents current behaviour.
			name:     "data sources need depth 1",
			output:   "data.aws_ami.ubuntu\n",
			maxDepth: 1,
			want:     []string{"data.aws_ami.ubuntu"},
		},
		{
			name:     "top-level data source hidden at depth 0 (known quirk)",
			output:   "data.aws_ami.ubuntu\n",
			maxDepth: 0,
			want:     []string{},
		},
		{
			name:     "trailing .module segment does not panic",
			output:   "weird.module\n",
			maxDepth: 5,
			want:     []string{"weird.module"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanupList(parseStateList(tt.output, tt.maxDepth))
			want := make([]string, len(tt.want))
			copy(want, tt.want)
			sort.Strings(want)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("parseStateList() = %v, want %v", got, want)
			}
		})
	}
}

func TestCleanupList(t *testing.T) {
	got := cleanupList([]string{"  b ", "", "a", "   ", "b"})
	want := []string{"a", "b", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("cleanupList() = %v, want %v", got, want)
	}
}

func TestNormalize(t *testing.T) {
	got := normalize([]string{" x ", "", "y", "  "})
	want := []string{"x", "y"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("normalize() = %v, want %v", got, want)
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		val string
		def bool
		set bool
		out bool
	}{
		{set: false, def: true, out: true},
		{set: true, val: "true", def: false, out: true},
		{set: true, val: "1", def: false, out: true},
		{set: true, val: "YES", def: false, out: true},
		{set: true, val: "false", def: true, out: false},
		{set: true, val: "0", def: true, out: false},
		{set: true, val: "no", def: true, out: false},
		{set: true, val: "garbage", def: true, out: true},
	}
	const key = "TFRS_TEST_BOOL"
	for _, tt := range tests {
		os.Unsetenv(key)
		if tt.set {
			os.Setenv(key, tt.val)
		}
		if got := getEnvBool(key, tt.def); got != tt.out {
			t.Errorf("getEnvBool(%q=%q, def=%v) = %v, want %v", key, tt.val, tt.def, got, tt.out)
		}
	}
	os.Unsetenv(key)
}

func TestGetEnvInt(t *testing.T) {
	const key = "TFRS_TEST_INT"
	os.Unsetenv(key)
	if got := getEnvInt(key, 7); got != 7 {
		t.Errorf("unset: got %d, want 7", got)
	}
	os.Setenv(key, "42")
	if got := getEnvInt(key, 7); got != 42 {
		t.Errorf("set: got %d, want 42", got)
	}
	os.Setenv(key, "notanumber")
	if got := getEnvInt(key, 7); got != 7 {
		t.Errorf("invalid: got %d, want 7 (default)", got)
	}
	os.Unsetenv(key)
}

func TestGetResourcesFromManifest(t *testing.T) {
	dir := t.TempDir()
	tf := `
resource "aws_instance" "web" {}
resource "aws_s3_bucket" "data" {}
data "aws_ami" "ubuntu" {}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(tf), 0o644); err != nil {
		t.Fatal(err)
	}

	got := getResourcesFromManifest(dir, "", &ModulesManifest{}, 0, 0)
	want := []string{"aws_instance.web", "aws_s3_bucket.data"}
	// data sources are not part of ManagedResources, so they are excluded here.
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("getResourcesFromManifest() = %v, want %v", got, want)
	}
}
