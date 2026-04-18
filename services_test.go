package main

import (
	"testing"
)

func TestChunkText(t *testing.T) {
	text := "Hello world. This is a test. One more sentence."
	chunks := chunkText(text, 30)
	if len(chunks) == 0 {
		t.Error("expected chunks but got none")
	}
	for _, c := range chunks {
		if len(c) > 30 {
			t.Errorf("chunk exceeds max size: %d chars", len(c))
		}
	}
}

func TestWithOneChunk(t *testing.T) {
	text := "Hello."
	chunks := chunkText(text, 2000)
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(chunks))
	}
}
func TestWithZeroText(t *testing.T) {
	chunks := chunkText("", 2000)
	if len(chunks) != 0 {
		t.Errorf("expected empty chunk, got %d", len(chunks))
	}
}
