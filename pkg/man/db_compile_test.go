package man

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func mustDescribe(t *testing.T, dbPath, uidStr string) string {
	t.Helper()
	uid, err := url.Parse(uidStr)
	if err != nil {
		t.Fatalf("parse uid %q: %v", uidStr, err)
	}
	doc, ok, err := InspectDescribe(dbPath, uid)
	if err != nil {
		t.Fatalf("inspect %q: %v", uidStr, err)
	}
	if !ok {
		t.Fatalf("uid %q not found in %s", uidStr, dbPath)
	}
	return doc
}

func TestCompileDB_Import(t *testing.T) {
	dir := t.TempDir()

	// source db: jj scheme host
	jjDir := filepath.Join(dir, "src", "jj", "revset-function")
	if err := os.MkdirAll(jjDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(jjDir, "revset-function.yaml"), []byte("parents: |\n  Parents description\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(dir, "jj.db")
	if err := CompileManDir(filepath.Join(dir, "src"), src); err != nil {
		t.Fatalf("compile src: %v", err)
	}

	// destination db: different content
	fooDir := filepath.Join(dir, "dst", "foo", "bar")
	if err := os.MkdirAll(fooDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fooDir, "bar.yaml"), []byte("x: |\n  Foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "carapace.db")
	if err := CompileManDir(filepath.Join(dir, "dst"), dst); err != nil {
		t.Fatalf("compile dst: %v", err)
	}

	if err := CompileDB(src, dst, WithAmend(true)); err != nil {
		t.Fatalf("import db: %v", err)
	}

	if got := mustDescribe(t, dst, "jj://revset-function/parents"); got == "" {
		t.Fatal("expected imported jj doc")
	}
	if got := mustDescribe(t, dst, "foo://bar/x"); got == "" {
		t.Fatal("expected existing host to be preserved")
	}
}

func TestCompileDB_ImportGzipAndReplaceHost(t *testing.T) {
	dir := t.TempDir()

	writeStatic := func(root, scheme, host, key, value string) string {
		hdir := filepath.Join(root, scheme, host)
		if err := os.MkdirAll(hdir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(hdir, host+".yaml"), []byte(key+": |\n  "+value+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return root
	}

	src1 := filepath.Join(dir, "src1.db.gz")
	if err := CompileManDir(writeStatic(filepath.Join(dir, "s1"), "jj", "revset-function", "parents", "old"), src1); err != nil {
		t.Fatalf("compile src1: %v", err)
	}
	src2 := filepath.Join(dir, "src2.db.gz")
	if err := CompileManDir(writeStatic(filepath.Join(dir, "s2"), "jj", "revset-function", "parents", "new"), src2); err != nil {
		t.Fatalf("compile src2: %v", err)
	}

	dst := filepath.Join(dir, "carapace.db.gz")
	if err := CompileDB(src1, dst); err != nil {
		t.Fatalf("import src1: %v", err)
	}
	if err := CompileDB(src2, dst, WithAmend(true)); err != nil {
		t.Fatalf("import src2: %v", err)
	}

	doc := mustDescribe(t, dst, "jj://revset-function/parents")
	if doc != "new\n" && doc != "new" {
		t.Fatalf("expected replaced host content, got %q", doc)
	}
}
