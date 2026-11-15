package cli

import (
	"path/filepath"
	"testing"

	"github.com/respawn-llc/ksrc/internal/fileidcache"
)

func TestSourceReading(t *testing.T) {
	t.Setenv("KSRC_FILEID_CACHE_DIR", t.TempDir())
	t.Setenv("PAGER", "ksrc-pager-must-not-run")
	jar := filepath.Join(t.TempDir(), "source.jar")
	const fileID = "test:source:1.0!/Sample.kt"
	if err := writeTestJar(jar, "Sample.kt", "first\n\nthird\n"); err != nil {
		t.Fatal(err)
	}
	if err := fileidcache.Register(fileID, jar); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"cat", "open"} {
		for _, tc := range []struct {
			name    string
			flags   []string
			want    string
			wantErr bool
		}{
			{name: "raw", want: "first\n\nthird\n"},
			{name: "range", flags: []string{"--lines", "2,3"}, want: "\nthird\n"},
			{name: "numbered", flags: []string{"--line-numbers"}, want: "1: first\n2: \n3: third\n"},
			{name: "numbered range", flags: []string{"--lines", "2,3", "-n"}, want: "2: \n3: third\n"},
			{name: "invalid range", flags: []string{"--lines", "3,2"}, wantErr: true},
		} {
			t.Run(command+"/"+tc.name, func(t *testing.T) {
				args := append([]string{command, fileID}, tc.flags...)
				out, err := runCommand(NewApp(), args)
				if (err != nil) != tc.wantErr {
					t.Fatalf("error = %v", err)
				}
				if out != tc.want {
					t.Fatalf("output = %q; want %q", out, tc.want)
				}
			})
		}
	}
}
