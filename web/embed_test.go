package web

import (
	"io/fs"
	"testing"
)

func TestDashboardAssetsAreEmbedded(t *testing.T) {
	dashboard, err := fs.Sub(Dashboard, "dashboard")
	if err != nil {
		t.Fatalf("create dashboard filesystem: %v", err)
	}

	for _, path := range []string{"index.html", "app.js", "styles.css"} {
		if _, err := fs.ReadFile(dashboard, path); err != nil {
			t.Fatalf("read embedded dashboard asset %q: %v", path, err)
		}
	}
}
