package service

import (
	"testing"

	"github.com/xyztony999/devswitch/internal/models"
	"github.com/xyztony999/devswitch/internal/store"
)

// saveTestState / loadTestState：单测内直接读写 state（沙箱 paths 已被替换）。
func saveTestState(t *testing.T, s *models.State) {
	t.Helper()
	if err := store.Save(*s); err != nil {
		t.Fatal(err)
	}
}

func loadTestState(t *testing.T) models.State {
	t.Helper()
	return store.Load()
}
