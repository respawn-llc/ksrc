package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestPagerCustomCommand(t *testing.T) {
	cmd := pagerCommand("go version")
	cmd.Stdin = strings.NewReader("source text\n")
	out, err := cmd.CombinedOutput()
	if err != nil || !bytes.Contains(out, []byte("go version")) {
		t.Fatalf("pager override failed: %v\n%s", err, out)
	}
}
