package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestPagerWindowsDefaultReadsSource(t *testing.T) {
	for _, pager := range []string{"", `"%SystemRoot%\System32\findstr.exe" /c:"WindowsSource"`} {
		t.Run(pager, func(t *testing.T) {
			cmd := pagerCommand(pager)
			cmd.Stdin = strings.NewReader("public class WindowsSource\r\n")
			out, err := cmd.CombinedOutput()
			if err != nil || !bytes.Contains(out, []byte("public class WindowsSource")) {
				t.Fatalf("native pager failed: %v\n%s", err, out)
			}
		})
	}
}
