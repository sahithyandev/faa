package hosts

import (
	"bytes"
	"runtime"
	"testing"
)

func TestIsSupported(t *testing.T) {
	supported := IsSupported()
	expected := runtime.GOOS == "linux"
	
	if supported != expected {
		t.Errorf("IsSupported() = %v, want %v (on %s)", supported, expected, runtime.GOOS)
	}
}

func TestHasEntry(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		hostname string
		want     bool
	}{
		{
			name: "entry exists",
			content: `127.0.0.1 localhost
# faa-managed-start
127.0.0.1 my-app.local
# faa-managed-end
`,
			hostname: "my-app.local",
			want:     true,
		},
		{
			name: "entry does not exist",
			content: `127.0.0.1 localhost
# faa-managed-start
127.0.0.1 other-app.local
# faa-managed-end
`,
			hostname: "my-app.local",
			want:     false,
		},
		{
			name: "no faa section",
			content: `127.0.0.1 localhost
`,
			hostname: "my-app.local",
			want:     false,
		},
		{
			name: "commented entry should not match",
			content: `127.0.0.1 localhost
# faa-managed-start
# 127.0.0.1 my-app.local
# faa-managed-end
`,
			hostname: "my-app.local",
			want:     false,
		},
		{
			name: "partial match should not match (no false positives)",
			content: `127.0.0.1 localhost
# faa-managed-start
127.0.0.1 my-app.local
# faa-managed-end
`,
			hostname: "app.local",
			want:     false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasEntry([]byte(tt.content), tt.hostname)
			if got != tt.want {
				t.Errorf("hasEntry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddEntryToContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		hostname string
		want     string
	}{
		{
			name:     "add to existing faa section",
			content: `127.0.0.1 localhost
# faa-managed-start
127.0.0.1 other-app.local
# faa-managed-end
`,
			hostname: "my-app.local",
			want: `127.0.0.1 localhost
# faa-managed-start
127.0.0.1 other-app.local
127.0.0.1 my-app.local
# faa-managed-end
`,
		},
		{
			name:     "create faa section if not exists",
			content:  "127.0.0.1 localhost\n",
			hostname: "my-app.local",
			want: `127.0.0.1 localhost

# faa-managed-start
127.0.0.1 my-app.local
# faa-managed-end
`,
		},
		{
			name:     "add to empty file",
			content:  "",
			hostname: "my-app.local",
			want: `
# faa-managed-start
127.0.0.1 my-app.local
# faa-managed-end
`,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addEntryToContent([]byte(tt.content), tt.hostname)
			if string(got) != tt.want {
				t.Errorf("addEntryToContent() =\n%q\nwant:\n%q", string(got), tt.want)
			}
		})
	}
}

func TestRemoveEntryFromContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		hostname string
		want     string
	}{
		{
			name: "remove existing entry",
			content: `127.0.0.1 localhost
# faa-managed-start
127.0.0.1 my-app.local
127.0.0.1 other-app.local
# faa-managed-end
`,
			hostname: "my-app.local",
			want: `127.0.0.1 localhost
# faa-managed-start
127.0.0.1 other-app.local
# faa-managed-end
`,
		},
		{
			name: "remove non-existing entry",
			content: `127.0.0.1 localhost
# faa-managed-start
127.0.0.1 other-app.local
# faa-managed-end
`,
			hostname: "my-app.local",
			want: `127.0.0.1 localhost
# faa-managed-start
127.0.0.1 other-app.local
# faa-managed-end
`,
		},
		{
			name: "partial match should not remove (no false positives)",
			content: `127.0.0.1 localhost
# faa-managed-start
127.0.0.1 my-app.local
127.0.0.1 other-app.local
# faa-managed-end
`,
			hostname: "app.local",
			want: `127.0.0.1 localhost
# faa-managed-start
127.0.0.1 my-app.local
127.0.0.1 other-app.local
# faa-managed-end
`,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeEntryFromContent([]byte(tt.content), tt.hostname)
			if string(got) != tt.want {
				t.Errorf("removeEntryFromContent() =\n%q\nwant:\n%q", string(got), tt.want)
			}
		})
	}
}

func TestSyncEntriesInContent(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		hostnames []string
		want      string
	}{
		{
			name: "sync with new entries",
			content: `127.0.0.1 localhost
# faa-managed-start
127.0.0.1 old-app.local
# faa-managed-end
`,
			hostnames: []string{"app1.local", "app2.local"},
			want: `127.0.0.1 localhost
# faa-managed-start
127.0.0.1 app1.local
127.0.0.1 app2.local
# faa-managed-end
`,
		},
		{
			name:      "create section with entries",
			content:   "127.0.0.1 localhost\n",
			hostnames: []string{"app1.local"},
			want: `127.0.0.1 localhost

# faa-managed-start
127.0.0.1 app1.local
# faa-managed-end
`,
		},
		{
			name: "remove all entries when list is empty",
			content: `127.0.0.1 localhost
# faa-managed-start
127.0.0.1 app1.local
# faa-managed-end
`,
			hostnames: []string{},
			want:      "127.0.0.1 localhost\n",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := syncEntriesInContent([]byte(tt.content), tt.hostnames)
			if !bytes.Equal(got, []byte(tt.want)) {
				t.Errorf("syncEntriesInContent() =\n%q\nwant:\n%q", string(got), tt.want)
			}
		})
	}
}
