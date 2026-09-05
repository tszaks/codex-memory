package cmd

import (
	"bytes"
	"encoding/json"
	"github.com/tszaks/pallium/internal/routing"
	"path/filepath"
	"testing"
)

func TestRouteInitPreservesExistingPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "routing.json")
	var out bytes.Buffer
	app := NewApp(&out, &out)
	if err := app.Run([]string{"route", "models", "init", "--config", path, "--json"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"route", "models", "init", "--config", path}); err == nil {
		t.Fatal("overwrote policy")
	}
	c, err := routing.Load(path)
	if err != nil || c.Mode != "shadow" {
		t.Fatalf("%+v %v", c, err)
	}
	out.Reset()
	if err := app.Run([]string{"route", "models", "catalog", "--config", path, "--json"}); err != nil {
		t.Fatal(err)
	}
	var decoded routing.Config
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Candidates) != 6 {
		t.Fatal(decoded)
	}
}

func TestRouteRejectsDisallowedLabWithoutLaunching(t *testing.T) {
	path := filepath.Join(t.TempDir(), "routing.json")
	var out bytes.Buffer
	app := NewApp(&out, &out)
	if err := app.Run([]string{"route", "models", "init", "--config", path}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"route", "models", "explain", "--config", path, "--provider", "claude"}); err == nil {
		t.Fatal("crossed lab policy")
	}
}
