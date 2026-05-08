package create

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMaybeReadFileReadsBareRelativePath(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "cert.pem")
	if err := os.WriteFile(path, []byte("file-cert"), 0o644); err != nil {
		t.Fatalf("write cert: %v", err)
	}

	got, err := maybeReadFile(path)
	if err != nil {
		t.Fatalf("maybeReadFile failed: %v", err)
	}
	if got != "file-cert" {
		t.Fatalf("expected file contents, got %q", got)
	}
}

func TestMaybeReadFileKeepsPEMLiteral(t *testing.T) {
	input := "-----BEGIN CERTIFICATE-----\ninline\n-----END CERTIFICATE-----"
	got, err := maybeReadFile(input)
	if err != nil {
		t.Fatalf("maybeReadFile failed: %v", err)
	}
	if got != input {
		t.Fatalf("expected inline PEM to stay unchanged")
	}
}
