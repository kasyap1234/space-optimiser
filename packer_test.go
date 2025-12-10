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

	// Verify no overlaps in placements
	for _, box := range packedBoxes {
		if !verifyNoOverlaps(box.Contents) {
			t.Errorf("Detected overlapping items in box %s", box.BoxID)
		}
	}

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

	// Verify no overlaps
	for _, box := range packedBoxes {
		if !verifyNoOverlaps(box.Contents) {
			t.Errorf("Detected overlapping items in box %s", box.BoxID)
		}
	}
}

func TestNoOverlap(t *testing.T) {
	// Test that items are placed without overlapping
	items := []InputItem{
		{ID: "cube", W: 10, H: 10, D: 10, Quantity: 8},
	}

	boxes := []InputBox{
		{ID: "box", W: 20, H: 20, D: 20}, // Should fit exactly 8 cubes
	}

	packedBoxes, unpackedItems := Pack(items, boxes)

	if len(unpackedItems) > 0 {
		t.Errorf("Expected all 8 cubes to be packed, but got %d unpacked", len(unpackedItems))
	}

	if len(packedBoxes) != 1 {
		t.Errorf("Expected 1 box, got %d", len(packedBoxes))
	}

	// Verify each box has 8 items
	if len(packedBoxes[0].Contents) != 8 {
		t.Errorf("Expected 8 items in box, got %d", len(packedBoxes[0].Contents))
	}

	// Verify no overlaps
	if !verifyNoOverlaps(packedBoxes[0].Contents) {
		t.Error("Detected overlapping items!")
	}
}

func TestRotation(t *testing.T) {
	// Test that rotation is used to fit items
	items := []InputItem{
		{ID: "long-item", W: 50, H: 5, D: 5, Quantity: 1},
	}

	boxes := []InputBox{
		{ID: "tall-box", W: 10, H: 60, D: 10}, // Item needs rotation to fit
	}

	packedBoxes, unpackedItems := Pack(items, boxes)

	if len(unpackedItems) > 0 {
		t.Errorf("Expected long item to be packed (rotated), but it wasn't")
	}

	if len(packedBoxes) != 1 {
		t.Errorf("Expected 1 box, got %d", len(packedBoxes))
	}

	// The item should be rotated to fit
	if len(packedBoxes[0].Contents) != 1 {
		t.Errorf("Expected 1 item, got %d", len(packedBoxes[0].Contents))
	}

	item := packedBoxes[0].Contents[0]
	// Check that item fits in box dimensions after rotation
	if item.W > 10 || item.H > 60 || item.D > 10 {
		t.Errorf("Item dimensions %dx%dx%d don't fit in 10x60x10 box", item.W, item.H, item.D)
	}
}

func TestItemsWithinBounds(t *testing.T) {
	// Test that all items stay within box bounds
	items := []InputItem{
		{ID: "item-1", W: 10, H: 10, D: 10, Quantity: 5},
		{ID: "item-2", W: 8, H: 8, D: 8, Quantity: 3},
	}

	boxes := []InputBox{
		{ID: "box", W: 30, H: 30, D: 30},
	}

	packedBoxes, _ := Pack(items, boxes)

	box := boxes[0]
	for _, pb := range packedBoxes {
		for _, item := range pb.Contents {
			if item.X < 0 || item.Y < 0 || item.Z < 0 {
				t.Errorf("Item %s has negative position: (%d,%d,%d)", item.ItemID, item.X, item.Y, item.Z)
			}
			if item.X+item.W > box.W || item.Y+item.H > box.H || item.Z+item.D > box.D {
				t.Errorf("Item %s extends outside box bounds: pos(%d,%d,%d) size(%d,%d,%d)",
					item.ItemID, item.X, item.Y, item.Z, item.W, item.H, item.D)
			}
		}
	}
}

func TestTightPacking(t *testing.T) {
	// Test tight packing scenario
	items := []InputItem{
		{ID: "item", W: 10, H: 10, D: 10, Quantity: 3},
	}

	boxes := []InputBox{
		{ID: "box", W: 20, H: 10, D: 20}, // Volume 4000, can fit 3 items (3000) but spatially only 4 positions
	}

	packedBoxes, unpackedItems := Pack(items, boxes)

	// Should pack all 3 items in various positions
	totalPacked := 0
	for _, pb := range packedBoxes {
		totalPacked += len(pb.Contents)
		if !verifyNoOverlaps(pb.Contents) {
			t.Error("Detected overlapping items in tight packing")
		}
	}

	if len(unpackedItems) > 0 {
		t.Errorf("Expected all 3 items packed, got %d unpacked", len(unpackedItems))
	}
}

func TestBottomLeftBackPreference(t *testing.T) {
	// Test that items are packed with bottom-left-back preference
	items := []InputItem{
		{ID: "item", W: 5, H: 5, D: 5, Quantity: 1},
	}

	boxes := []InputBox{
		{ID: "box", W: 20, H: 20, D: 20},
	}

	packedBoxes, _ := Pack(items, boxes)

	if len(packedBoxes) == 0 || len(packedBoxes[0].Contents) == 0 {
		t.Fatal("Expected item to be packed")
	}

	item := packedBoxes[0].Contents[0]
	// Item should be at origin (0,0,0) due to bottom-left-back preference
	if item.X != 0 || item.Y != 0 || item.Z != 0 {
		t.Errorf("Expected item at origin (0,0,0), got (%d,%d,%d)", item.X, item.Y, item.Z)
	}
}

// Helper function to verify no items overlap
func verifyNoOverlaps(placements []Placement) bool {
	for i := 0; i < len(placements); i++ {
		for j := i + 1; j < len(placements); j++ {
			p1 := placements[i]
			p2 := placements[j]

			// Check overlap on all three axes
			overlapX := p1.X < p2.X+p2.W && p1.X+p1.W > p2.X
			overlapY := p1.Y < p2.Y+p2.H && p1.Y+p1.H > p2.Y
			overlapZ := p1.Z < p2.Z+p2.D && p1.Z+p1.D > p2.Z

			if overlapX && overlapY && overlapZ {
				return false
			}
		}
	}
	return true
}
