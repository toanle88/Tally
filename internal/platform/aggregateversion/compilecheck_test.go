package aggregateversion

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAggregateVersionRejectsRawInt64AssignmentsAtCompileTime(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("go test subprocess is unavailable on js")
	}

	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err = filepath.Abs(filepath.Join(root, "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	temp := t.TempDir()
	write := func(name, contents string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(temp, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module github.com/toanle88/Tally/internal-aggregate-version-compilecheck\n\ngo 1.26.3\n\nrequire github.com/toanle88/Tally v0.0.0\n\nreplace github.com/toanle88/Tally => "+root+"\n")
	write("compilecheck_test.go", `package compilecheck

import (
	"testing"

	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

func TestInvalidAggregateVersionAssignment(t *testing.T) {
	var version aggregateversion.AggregateVersion
	acceptInt64(version)
}

func acceptInt64(int64) {}
`)

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = temp
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("invalid AggregateVersion assignment compiled successfully")
	}
	text := string(output)
	if !strings.Contains(text, "cannot use") && !strings.Contains(text, "cannot assign") {
		t.Fatalf("compile failed for an unexpected reason: %s", text)
	}
}
