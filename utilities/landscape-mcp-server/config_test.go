package main

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.Name != "CNCF" {
		t.Errorf("expected Name=CNCF, got %s", cfg.Name)
	}
	if cfg.Description != "Cloud Native Computing Foundation" {
		t.Errorf("expected Description='Cloud Native Computing Foundation', got %s", cfg.Description)
	}
}
