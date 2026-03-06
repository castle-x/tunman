package core

import (
	"os"
	"strings"
	"testing"
)

func TestStorageReadLogs(t *testing.T) {
	tests := []struct {
		name    string
		content string
		lines   int
		want    string
	}{
		{
			name:    "missing file returns empty",
			content: "",
			lines:   100,
			want:    "",
		},
		{
			name:    "returns full content when lines disabled",
			content: "one\ntwo\n",
			lines:   0,
			want:    "one\ntwo\n",
		},
		{
			name:    "tails requested line count",
			content: "one\ntwo\nthree\nfour\n",
			lines:   2,
			want:    "three\nfour\n",
		},
		{
			name:    "handles windows newlines",
			content: strings.Join([]string{"one", "two", "three"}, "\r\n") + "\r\n",
			lines:   2,
			want:    "two\nthree\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &Storage{BaseDir: t.TempDir()}
			path := storage.LogPath("demo")

			if tt.content != "" {
				if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}
			}

			got, err := storage.ReadLogs("demo", tt.lines)
			if err != nil {
				t.Fatalf("ReadLogs() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ReadLogs() = %q, want %q", got, tt.want)
			}
		})
	}
}
