package main

import "testing"

func TestByteCount(t *testing.T) {
	meg := ByteCountIEC(1024 * 1024)
	if meg != "1.0 MiB" {
		t.Errorf("Expected 1.0 MiB, got %s", meg)
	}
	meg = ByteCountIEC(0)
	if meg != "0 B" {
		t.Errorf("Expected 0 B, got %s", meg)
	}
}
