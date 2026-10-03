package generator_test

import (
	"strings"
	"testing"

	"github.com/ccqpein/lisp-rpc-golang/generator"
)

func TestDefMsgDocSupport(t *testing.T) {
	spec := `(def-msg user "User model documentation" :id 'number :name 'string)`
	dm, err := generator.ParseDefMsg(spec)
	if err != nil {
		t.Fatalf("failed to parse def-msg: %v", err)
	}
	if dm.Doc != "User model documentation" {
		t.Fatalf("unexpected doc: %s", dm.Doc)
	}

	code, err := dm.GenCode()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}
	if !strings.Contains(code, "// User model documentation\ntype User struct") {
		t.Fatalf("missing struct comment in generated code:\n%s", code)
	}
}

func TestDefRPCDocSupport(t *testing.T) {
	spec := `(def-rpc fetch-user
    "Fetch user documentation."
    '(:id 'number)
  'user)`
	dr, err := generator.ParseDefRPC(spec)
	if err != nil {
		t.Fatalf("failed to parse def-rpc: %v", err)
	}
	if dr.Doc != "Fetch user documentation." {
		t.Fatalf("unexpected doc: %s", dr.Doc)
	}

	code, err := dr.GenCode()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}
	if !strings.Contains(code, "// Fetch user documentation.\ntype FetchUser struct") {
		t.Fatalf("missing rpc struct comment in generated code:\n%s", code)
	}
}
