package aggregation

import (
	"testing"

	"github.com/maax3v3/macoma/internal/color"
)

func TestReduceColors_Empty(t *testing.T) {
	cm := ReduceColors(nil, 5)
	if len(cm.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(cm.Entries))
	}
	if len(cm.ZoneMap) != 0 {
		t.Errorf("expected 0 zone mappings, got %d", len(cm.ZoneMap))
	}
}

func TestReduceColors_NoReduction(t *testing.T) {
	colors := []color.RGBA{
		{255, 0, 0, 255},
		{0, 255, 0, 255},
		{0, 0, 255, 255},
	}
	cm := ReduceColors(colors, 0) // 0 = unlimited

	if len(cm.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(cm.Entries))
	}
	if len(cm.ZoneMap) != 3 {
		t.Fatalf("expected 3 zone mappings, got %d", len(cm.ZoneMap))
	}

	// Each zone should map to a distinct entry
	seen := make(map[int]bool)
	for _, idx := range cm.ZoneMap {
		seen[idx] = true
	}
	if len(seen) != 3 {
		t.Errorf("expected 3 distinct entry indices, got %d", len(seen))
	}

	// Numbers should be 1-based
	for _, e := range cm.Entries {
		if e.Number < 1 || e.Number > 3 {
			t.Errorf("unexpected number %d", e.Number)
		}
	}
}

func TestReduceColors_DuplicateColors(t *testing.T) {
	red := color.RGBA{255, 0, 0, 255}
	blue := color.RGBA{0, 0, 255, 255}
	colors := []color.RGBA{red, red, blue, red}

	cm := ReduceColors(colors, 0)

	// 2 distinct input colors → 2 entries
	if len(cm.Entries) != 2 {
		t.Fatalf("expected 2 entries for 2 distinct colors, got %d", len(cm.Entries))
	}

	// All red zones should map to the same entry
	if cm.ZoneMap[0] != cm.ZoneMap[1] || cm.ZoneMap[0] != cm.ZoneMap[3] {
		t.Error("duplicate red zones should map to the same entry")
	}
	// Blue zone should map to a different entry
	if cm.ZoneMap[2] == cm.ZoneMap[0] {
		t.Error("blue zone should map to a different entry than red")
	}
}

func TestReduceColors_MergeToMaxColors(t *testing.T) {
	colors := []color.RGBA{
		{255, 0, 0, 255},   // red
		{250, 0, 0, 255},   // near-red
		{0, 0, 255, 255},   // blue
		{0, 0, 250, 255},   // near-blue
		{0, 255, 0, 255},   // green
	}

	cm := ReduceColors(colors, 3)

	if len(cm.Entries) != 3 {
		t.Fatalf("expected 3 entries after reduction, got %d", len(cm.Entries))
	}
	if len(cm.ZoneMap) != 5 {
		t.Fatalf("expected 5 zone mappings, got %d", len(cm.ZoneMap))
	}

	// Near-red and red should merge; near-blue and blue should merge
	if cm.ZoneMap[0] != cm.ZoneMap[1] {
		t.Error("red and near-red should be merged into the same group")
	}
	if cm.ZoneMap[2] != cm.ZoneMap[3] {
		t.Error("blue and near-blue should be merged into the same group")
	}
}

func TestReduceColors_MergeToOne(t *testing.T) {
	colors := []color.RGBA{
		{100, 0, 0, 255},
		{0, 100, 0, 255},
		{0, 0, 100, 255},
	}

	cm := ReduceColors(colors, 1)

	if len(cm.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(cm.Entries))
	}

	// All zones must map to the single entry
	for i, idx := range cm.ZoneMap {
		if idx != 0 {
			t.Errorf("zone %d maps to %d, want 0", i, idx)
		}
	}
}

func TestReduceColors_MaxColorsExceedsDistinct(t *testing.T) {
	colors := []color.RGBA{
		{255, 0, 0, 255},
		{0, 255, 0, 255},
	}

	cm := ReduceColors(colors, 10)

	// Should not merge anything since 2 < 10
	if len(cm.Entries) != 2 {
		t.Errorf("expected 2 entries (no merging needed), got %d", len(cm.Entries))
	}
}

func TestReduceColors_SingleZone(t *testing.T) {
	colors := []color.RGBA{{42, 42, 42, 255}}
	cm := ReduceColors(colors, 5)

	if len(cm.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(cm.Entries))
	}
	if cm.Entries[0].Color != colors[0] {
		t.Errorf("color mismatch: got %+v, want %+v", cm.Entries[0].Color, colors[0])
	}
	if cm.Entries[0].Number != 1 {
		t.Errorf("number: got %d, want 1", cm.Entries[0].Number)
	}
}

func TestReduceColors_NumbersAreOneBased(t *testing.T) {
	colors := []color.RGBA{
		{255, 0, 0, 255},
		{0, 255, 0, 255},
		{0, 0, 255, 255},
	}
	cm := ReduceColors(colors, 0)

	numbers := make(map[int]bool)
	for _, e := range cm.Entries {
		numbers[e.Number] = true
	}
	for i := 1; i <= len(cm.Entries); i++ {
		if !numbers[i] {
			t.Errorf("missing expected number %d", i)
		}
	}
}
