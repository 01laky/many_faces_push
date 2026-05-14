package grpccreds

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLoadServerCredentials_plaintextWhenBothUnset(t *testing.T) {
	t.Parallel()
	c, err := LoadServerCredentials("", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c != nil {
		t.Fatalf("expected nil credentials for plaintext")
	}
}

func TestLoadServerCredentials_errorsWhenOnlyOnePathSet(t *testing.T) {
	t.Parallel()
	_, err := LoadServerCredentials("/tmp/a.crt", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = LoadServerCredentials("", "/tmp/a.key", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadServerCredentials_errorsOnCorruptKeyAfterValidGen(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not on PATH")
	}
	dir := t.TempDir()
	cert := filepath.Join(dir, "server.crt")
	key := filepath.Join(dir, "server.key")
	err := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes",
		"-keyout", key, "-out", cert, "-days", "1", "-subj", "/CN=localhost",
		"-addext", "subjectAltName=DNS:localhost").Run()
	if err != nil {
		t.Fatalf("openssl: %v", err)
	}
	if err := exec.Command("sh", "-c", "echo not-a-key > "+key).Run(); err != nil { //nolint:gosec
		t.Fatal(err)
	}
	_, err = LoadServerCredentials(cert, key, "")
	if err == nil {
		t.Fatal("expected error for corrupted key PEM")
	}
}

func TestLoadServerCredentials_mtlsRequiresValidClientCA(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not on PATH")
	}
	dir := t.TempDir()
	cert := filepath.Join(dir, "server.crt")
	key := filepath.Join(dir, "server.key")
	err := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes",
		"-keyout", key, "-out", cert, "-days", "1", "-subj", "/CN=localhost",
		"-addext", "subjectAltName=DNS:localhost").Run()
	if err != nil {
		t.Fatalf("openssl: %v", err)
	}

	ca := filepath.Join(dir, "ca.pem")
	if err := exec.Command("sh", "-c", "echo '# not pem' > "+ca).Run(); err != nil { //nolint:gosec
		t.Fatal(err)
	}

	_, err = LoadServerCredentials(cert, key, ca)
	if err == nil {
		t.Fatal("expected error when client CA PEM has no certificates")
	}
}
