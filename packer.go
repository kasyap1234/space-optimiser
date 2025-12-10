package main

import (
	"math"
	"sort"
)

// --- Data Structures ---

type InputItem struct {
	ID       string `json:"id"`
	W        int    `json:"w"`
	H        int    `json:"h"`
	D        int    `json:"d"`
	Quantity int    `json:"quantity"`
}

type InputBox struct {
	ID string `json:"id"`
	W  int    `json:"w"`
	H  int    `json:"h"`
	D  int    `json:"d"`
}

type PackedBox struct {
	BoxID    string      `json:"box_id"`
	Contents []Placement `json:"contents"`
}

type Placement struct {
	ItemID string `json:"item_id"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Z      int    `json:"z"`
	W      int    `json:"w"`
	H      int    `json:"h"`
	D      int    `json:"d"`
}

// FreeSpace represents an available region in the box
type FreeSpace struct {
	X, Y, Z int
	W, H, D int
}

func (fs FreeSpace) Volume() int {
	return fs.W * fs.H * fs.D
}

// Check if two boxes overlap in 3D space
func boxesOverlap(p1 Placement, x2, y2, z2, w2, h2, d2 int) bool {
	// Two boxes overlap if they overlap on all three axes
	overlapX := p1.X < x2+w2 && p1.X+p1.W > x2
	overlapY := p1.Y < y2+h2 && p1.Y+p1.H > y2
	overlapZ := p1.Z < z2+d2 && p1.Z+p1.D > z2
	return overlapX && overlapY && overlapZ
}

// Check if a placement overlaps with any existing placements
func hasOverlap(placements []Placement, x, y, z, w, h, d int) bool {
	for _, p := range placements {
		if boxesOverlap(p, x, y, z, w, h, d) {
			return true
		}
	}
	return false
}

// Check if placement fits within box bounds
func fitsInBox(box InputBox, x, y, z, w, h, d int) bool {
	return x >= 0 && y >= 0 && z >= 0 &&
		x+w <= box.W && y+h <= box.H && z+d <= box.D
}

// Internal item representation for packing (handling quantity)
type itemToPack struct {
	InputItem
	Volume int
	MaxDim int
}

// --- Sorting ---

type byVolumeDesc []itemToPack

func (a byVolumeDesc) Len() int      { return len(a) }
func (a byVolumeDesc) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byVolumeDesc) Less(i, j int) bool {
	if a[i].Volume != a[j].Volume {
		return a[i].Volume > a[j].Volume
	}
	return a[i].MaxDim > a[j].MaxDim
}

type byBoxVolumeAsc []InputBox

func (a byBoxVolumeAsc) Len() int      { return len(a) }
func (a byBoxVolumeAsc) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byBoxVolumeAsc) Less(i, j int) bool {
	return (a[i].W * a[i].H * a[i].D) < (a[j].W * a[j].H * a[j].D)
}

// Sort free spaces by position (bottom-left-back preference)
type byPositionPreference []FreeSpace

func (a byPositionPreference) Len() int      { return len(a) }
func (a byPositionPreference) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byPositionPreference) Less(i, j int) bool {
	// Prefer lower Y (bottom), then lower Z (back), then lower X (left)
	if a[i].Y != a[j].Y {
		return a[i].Y < a[j].Y
	}
	if a[i].Z != a[j].Z {
		return a[i].Z < a[j].Z
	}
	return a[i].X < a[j].X
}

// --- Core Logic ---

func Pack(inputItems []InputItem, availableBoxes []InputBox) ([]PackedBox, []InputItem) {
	// 1. Expand items based on quantity and calculate properties
	var items []itemToPack
	for _, item := range inputItems {
		for i := 0; i < item.Quantity; i++ {
			maxDim := item.W
			if item.H > maxDim {
				maxDim = item.H
			}
			if item.D > maxDim {
				maxDim = item.D
			}
			items = append(items, itemToPack{
				InputItem: item,
				Volume:    item.W * item.H * item.D,
				MaxDim:    maxDim,
			})
		}
	}

	// 2. Sort items by volume descending
	sort.Sort(byVolumeDesc(items))

	// 3. Sort boxes by volume (smallest first) - helpful for tie-breaking but we try all
	sort.Sort(byBoxVolumeAsc(availableBoxes))

	var packedBoxes []PackedBox
	var unpackedItems []InputItem

	remainingItems := items

	// 4. Packing Loop
	for len(remainingItems) > 0 {
		bestBoxIndex := -1
		var bestPlacements []Placement
		var bestIsPacked []bool
		bestPackedVol := -1

		// Try to pack remaining items into EACH available box type
		for i, box := range availableBoxes {
			placements, isPacked, packedVol := packIntoBox(remainingItems, box)

			if packedVol > 0 {
				boxVol := box.W * box.H * box.D
				// If we haven't picked a box yet, or this one packs MORE volume
				if bestBoxIndex == -1 || packedVol > bestPackedVol {
					bestBoxIndex = i
					bestPlacements = placements
					bestIsPacked = isPacked
					bestPackedVol = packedVol
				} else if packedVol == bestPackedVol {
					// Tie-breaker: Pick smaller box (higher utilization)
					bestBoxVol := availableBoxes[bestBoxIndex].W * availableBoxes[bestBoxIndex].H * availableBoxes[bestBoxIndex].D
					if boxVol < bestBoxVol {
						bestBoxIndex = i
						bestPlacements = placements
						bestIsPacked = isPacked
						bestPackedVol = packedVol
					}
				}
			}
		}

		if bestBoxIndex == -1 {
			// No box fits any of the remaining items
			for _, item := range remainingItems {
				unpackedItems = append(unpackedItems, item.InputItem)
			}
			break
		}

		// Commit the best box
		packedBoxes = append(packedBoxes, PackedBox{
			BoxID:    availableBoxes[bestBoxIndex].ID,
			Contents: bestPlacements,
		})

		// Update remaining items
		var nextRemaining []itemToPack
		for i, packed := range bestIsPacked {
			if !packed {
				nextRemaining = append(nextRemaining, remainingItems[i])
			}
		}
		remainingItems = nextRemaining
	}

	return packedBoxes, unpackedItems
}

// packIntoBox attempts to pack items into a specific box using Extreme Points algorithm
func packIntoBox(items []itemToPack, box InputBox) ([]Placement, []bool, int) {
	// Use Extreme Points (EP) algorithm for better packing
	// Start with the origin as the only extreme point
	extremePoints := []FreeSpace{{
		X: 0, Y: 0, Z: 0,
		W: box.W, H: box.H, D: box.D,
	}}

	var placements []Placement
	isPacked := make([]bool, len(items))
	packedVol := 0

	for i, item := range items {
		bestPoint := -1
		bestRotation := -1
		bestScore := math.MaxInt64

		// Sort extreme points by position preference (bottom-left-back)
		sort.Sort(byPositionPreference(extremePoints))

		// Try each extreme point
		for pi, ep := range extremePoints {
			// Try all 6 rotations
			rotations := getRotations(item.W, item.H, item.D)
			for r, dim := range rotations {
				w, h, d := dim[0], dim[1], dim[2]

				// Check if item fits at this point within box bounds
				if !fitsInBox(box, ep.X, ep.Y, ep.Z, w, h, d) {
					continue
				}

				// Check for overlap with existing placements
				if hasOverlap(placements, ep.X, ep.Y, ep.Z, w, h, d) {
					continue
				}

				// Score: prefer positions closer to origin (bottom-left-back)
				// Using weighted sum: Y is most important (gravity), then Z, then X
				score := ep.Y*1000 + ep.Z*100 + ep.X*10

				// Secondary: prefer tighter fit
				score += (ep.W - w) + (ep.H - h) + (ep.D - d)

				if score < bestScore {
					bestScore = score
					bestPoint = pi
					bestRotation = r
				}
			}
		}

		if bestPoint != -1 {
			// Place item
			ep := extremePoints[bestPoint]
			rotations := getRotations(item.W, item.H, item.D)
			w, h, d := rotations[bestRotation][0], rotations[bestRotation][1], rotations[bestRotation][2]

			placement := Placement{
				ItemID: item.ID,
				X:      ep.X, Y: ep.Y, Z: ep.Z,
				W: w, H: h, D: d,
			}
			placements = append(placements, placement)
			isPacked[i] = true
			packedVol += item.Volume

			// Generate new extreme points from this placement
			extremePoints = updateExtremePoints(extremePoints, placement, box, placements)
		}
	}

	return placements, isPacked, packedVol
}

// updateExtremePoints generates new extreme points after placing an item
func updateExtremePoints(eps []FreeSpace, placed Placement, box InputBox, placements []Placement) []FreeSpace {
	// Generate new potential extreme points at corners of placed item
	newPoints := []FreeSpace{
		// Right of placed item (X+W, Y, Z)
		{X: placed.X + placed.W, Y: placed.Y, Z: placed.Z, W: box.W - (placed.X + placed.W), H: box.H - placed.Y, D: box.D - placed.Z},
		// Top of placed item (X, Y+H, Z)
		{X: placed.X, Y: placed.Y + placed.H, Z: placed.Z, W: box.W - placed.X, H: box.H - (placed.Y + placed.H), D: box.D - placed.Z},
		// Front of placed item (X, Y, Z+D)
		{X: placed.X, Y: placed.Y, Z: placed.Z + placed.D, W: box.W - placed.X, H: box.H - placed.Y, D: box.D - (placed.Z + placed.D)},
	}

	// Filter out points outside box bounds or inside existing placements
	var validPoints []FreeSpace
	for _, ep := range newPoints {
		// Must be within box bounds
		if ep.X >= box.W || ep.Y >= box.H || ep.Z >= box.D {
			continue
		}
		if ep.X < 0 || ep.Y < 0 || ep.Z < 0 {
			continue
		}

		// Check if this point is inside an existing placement
		pointInsidePlacement := false
		for _, p := range placements {
			if ep.X >= p.X && ep.X < p.X+p.W &&
				ep.Y >= p.Y && ep.Y < p.Y+p.H &&
				ep.Z >= p.Z && ep.Z < p.Z+p.D {
				pointInsidePlacement = true
				break
			}
		}
		if !pointInsidePlacement {
			validPoints = append(validPoints, ep)
		}
	}

	// Keep existing extreme points that are still valid
	for _, ep := range eps {
		// Check if this point is now inside the new placement
		if ep.X >= placed.X && ep.X < placed.X+placed.W &&
			ep.Y >= placed.Y && ep.Y < placed.Y+placed.H &&
			ep.Z >= placed.Z && ep.Z < placed.Z+placed.D {
			continue // Point is now occupied
		}
		validPoints = append(validPoints, ep)
	}

	// Remove duplicates
	return removeDuplicatePoints(validPoints)
}

// removeDuplicatePoints removes duplicate extreme points
func removeDuplicatePoints(points []FreeSpace) []FreeSpace {
	seen := make(map[[3]int]bool)
	var result []FreeSpace
	for _, p := range points {
		key := [3]int{p.X, p.Y, p.Z}
		if !seen[key] {
			seen[key] = true
			result = append(result, p)
		}
	}
	return result
}

func getRotations(w, h, d int) [][3]int {
	return [][3]int{
		{w, h, d},
		{w, d, h},
		{h, w, d},
		{h, d, w},
		{d, w, h},
		{d, h, w},
	}
}
