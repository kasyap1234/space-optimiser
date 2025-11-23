package main

import (
	"testing"
)

func TestPack(t *testing.T) {
	items := []InputItem{
		{ID: "item-b", W: 20, H: 20, D: 20, Quantity: 1},
		{ID: "item-a", W: 10, H: 10, D: 10, Quantity: 2},
		{ID: "item-c", W: 5, H: 5, D: 5, Quantity: 5},
	}

	boxes := []InputBox{
		{ID: "box-small", W: 15, H: 15, D: 15},
		{ID: "box-large", W: 30, H: 30, D: 30},
	}

	packedBoxes, unpackedItems := Pack(items, boxes)

	if len(unpackedItems) > 0 {
		t.Errorf("Expected all items to be packed, but got %d unpacked items", len(unpackedItems))
	}

	if len(packedBoxes) == 0 {
		t.Errorf("Expected at least one packed box, but got 0")
	}

	// Verification logic
	// item-b (20x20x20) fits only in box-large (30x30x30)
	// item-a (10x10x10) x 2 fits in box-small (15x15x15) or box-large
	// item-c (5x5x5) x 5 fits in box-small or box-large

	// With the new strategy, it should try to pack everything into the best box.
	// Total volume:
	// item-b: 8000
	// item-a: 1000 * 2 = 2000
	// item-c: 125 * 5 = 625
	// Total: 10625

	// box-small vol: 3375
	// box-large vol: 27000

	// Everything fits in one box-large (27000 > 10625).
	// Ideally, it should pack everything into one box-large if possible,
	// OR maybe it splits if it thinks that's better?
	// But our logic tries to pack *remaining* items into *best* box.
	// box-large can take ALL items. box-small cannot take item-b.

	// So first iteration:
	// Remaining: [item-b, item-a, item-a, item-c...]
	// Try box-small: fails (item-b too big). PackedVol = 0 (or partial if we allowed partial, but we check if it fits the batch? No, we pack as much as possible).
	// Wait, packIntoBox packs as much as possible.
	// box-small: packs item-a's and item-c's. item-b fails.
	// box-large: packs item-b, item-a's, item-c's. All fit.

	// box-large packs MORE volume (everything). So it should be chosen.

	if len(packedBoxes) != 1 {
		t.Errorf("Expected 1 box to contain everything, got %d", len(packedBoxes))
	}

	if packedBoxes[0].BoxID != "box-large" {
		t.Errorf("Expected box-large, got %s", packedBoxes[0].BoxID)
	}
}

func TestPackSplit(t *testing.T) {
	// Test where items MUST be split across boxes
	items := []InputItem{
		{ID: "item-big-1", W: 20, H: 20, D: 20, Quantity: 1},
		{ID: "item-big-2", W: 20, H: 20, D: 20, Quantity: 1},
	}

	boxes := []InputBox{
		{ID: "box-medium", W: 25, H: 25, D: 25}, // Can only hold one 20x20x20 item
	}

	packedBoxes, unpackedItems := Pack(items, boxes)

	if len(unpackedItems) > 0 {
		t.Errorf("Expected all items to be packed, but got %d unpacked items", len(unpackedItems))
	}

	if len(packedBoxes) != 2 {
		t.Errorf("Expected 2 boxes, got %d", len(packedBoxes))
	}
}
