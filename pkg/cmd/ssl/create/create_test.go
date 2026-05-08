package create

import (
	"os"
	"testing"
)

func TestMaybeReadFileReadsBareRelativePath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	if err := os.WriteFile("cert.pem", []byte("file-cert"), 0o644); err != nil {
		t.Fatalf("write cert: %v", err)
	}

	got, err := maybeReadFile("cert.pem")
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
