package slog

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gelozr/go-dash/internal/config"
)

func TestJSONLoggerPreservesNonTimeAttributesNamedTime(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })

	path := filepath.Join(t.TempDir(), "app.log")
	log, err := New(&config.Config{LogFormat: "json", LogOutput: "file", LogPath: path})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := log.Close(); err != nil {
			t.Error(err)
		}
	})

	log.Info("checkpoint reached", slog.Group("checkpoint", "time", "after lunch"))

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Time       string
		Checkpoint struct{ Time string }
	}
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	if got := record.Checkpoint.Time; got != "after lunch" {
		t.Errorf("checkpoint time = %q, want after lunch", got)
	}
	if _, err := time.Parse(time.DateTime, record.Time); err != nil {
		t.Errorf("invalid log timestamp %q: %v", record.Time, err)
	}
}
