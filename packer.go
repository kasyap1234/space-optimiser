package main

import (
	"cmp"
	"math"
	"slices"
)

// InputItem represents an item to be packed.
type InputItem struct {
	ID       string `json:"id"`
	W        int    `json:"w"`
	H        int    `json:"h"`
	D        int    `json:"d"`
	Quantity int    `json:"quantity"`
}

// InputBox represents an available box type.
type InputBox struct {
	ID string `json:"id"`
	W  int    `json:"w"`
	H  int    `json:"h"`
	D  int    `json:"d"`
}

// PackedBox represents a box with its packed contents.
type PackedBox struct {
	BoxID    string      `json:"box_id"`
	Contents []Placement `json:"contents"`
}

// Placement represents an item's position and dimensions in a box.
type Placement struct {
	ItemID string `json:"item_id"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Z      int    `json:"z"`
	W      int    `json:"w"`
	H      int    `json:"h"`
	D      int    `json:"d"`
}

// FreeSpace represents an available region in the box.
type FreeSpace struct {
	X, Y, Z int
	W, H, D int
}

func (fs FreeSpace) volume() int {
	return fs.W * fs.H * fs.D
}

func (b InputBox) volume() int {
	return b.W * b.H * b.D
}

// itemToPack is an internal representation for packing (handles quantity expansion).
type itemToPack struct {
	InputItem
	volume int
	maxDim int
}

// Pack distributes items into boxes using the Extreme Points algorithm.
func Pack(inputItems []InputItem, availableBoxes []InputBox) ([]PackedBox, []InputItem) {
	items := expandItems(inputItems)
	sortItemsByVolume(items)

	boxes := slices.Clone(availableBoxes)
	slices.SortFunc(boxes, func(a, b InputBox) int {
		return cmp.Compare(a.volume(), b.volume())
	})

	var packedBoxes []PackedBox
	var unpackedItems []InputItem

	remaining := items
	for len(remaining) > 0 {
		bestIdx, bestPlacements, bestPacked := findBestBox(remaining, boxes)
		if bestIdx == -1 {
			for _, item := range remaining {
				unpackedItems = append(unpackedItems, item.InputItem)
			}
			break
		}

		packedBoxes = append(packedBoxes, PackedBox{
			BoxID:    boxes[bestIdx].ID,
			Contents: bestPlacements,
		})

		remaining = filterUnpacked(remaining, bestPacked)
	}

	return packedBoxes, unpackedItems
}

func expandItems(inputItems []InputItem) []itemToPack {
	var items []itemToPack
	for _, item := range inputItems {
		for range item.Quantity {
			items = append(items, itemToPack{
				InputItem: item,
				volume:    item.W * item.H * item.D,
				maxDim:    max(item.W, item.H, item.D),
			})
		}
	}
	return items
}

func sortItemsByVolume(items []itemToPack) {
	slices.SortFunc(items, func(a, b itemToPack) int {
		if c := cmp.Compare(b.volume, a.volume); c != 0 {
			return c
		}
		return cmp.Compare(b.maxDim, a.maxDim)
	})
}

func findBestBox(items []itemToPack, boxes []InputBox) (int, []Placement, []bool) {
	bestIdx := -1
	var bestPlacements []Placement
	var bestPacked []bool
	bestPackedVol := -1

	for i, box := range boxes {
		placements, packed, packedVol := packIntoBox(items, box)
		if packedVol <= 0 {
			continue
		}

		if bestIdx == -1 || packedVol > bestPackedVol {
			bestIdx, bestPlacements, bestPacked, bestPackedVol = i, placements, packed, packedVol
		} else if packedVol == bestPackedVol && box.volume() < boxes[bestIdx].volume() {
			bestIdx, bestPlacements, bestPacked = i, placements, packed
		}
	}

	return bestIdx, bestPlacements, bestPacked
}

func filterUnpacked(items []itemToPack, packed []bool) []itemToPack {
	var remaining []itemToPack
	for i, isPacked := range packed {
		if !isPacked {
			remaining = append(remaining, items[i])
		}
	}
	return remaining
}

// packIntoBox attempts to pack items into a specific box using the Extreme Points algorithm.
func packIntoBox(items []itemToPack, box InputBox) ([]Placement, []bool, int) {
	extremePoints := []FreeSpace{{
		X: 0, Y: 0, Z: 0,
		W: box.W, H: box.H, D: box.D,
	}}

	var placements []Placement
	packed := make([]bool, len(items))
	packedVol := 0

	for i, item := range items {
		sortByPosition(extremePoints)

		pointIdx, rotIdx := findBestPlacement(extremePoints, item, box, placements)
		if pointIdx == -1 {
			continue
		}

		ep := extremePoints[pointIdx]
		rot := rotations(item.W, item.H, item.D)[rotIdx]

		placement := Placement{
			ItemID: item.ID,
			X:      ep.X, Y: ep.Y, Z: ep.Z,
			W: rot[0], H: rot[1], D: rot[2],
		}
		placements = append(placements, placement)
		packed[i] = true
		packedVol += item.volume

		extremePoints = updateExtremePoints(extremePoints, placement, box, placements)
	}

	return placements, packed, packedVol
}

func sortByPosition(points []FreeSpace) {
	slices.SortFunc(points, func(a, b FreeSpace) int {
		if c := cmp.Compare(a.Y, b.Y); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Z, b.Z); c != 0 {
			return c
		}
		return cmp.Compare(a.X, b.X)
	})
}

func findBestPlacement(points []FreeSpace, item itemToPack, box InputBox, placements []Placement) (int, int) {
	bestPoint := -1
	bestRot := -1
	bestScore := math.MaxInt

	for pi, ep := range points {
		for ri, rot := range rotations(item.W, item.H, item.D) {
			w, h, d := rot[0], rot[1], rot[2]

			if !fitsInBox(box, ep.X, ep.Y, ep.Z, w, h, d) {
				continue
			}
			if hasOverlap(placements, ep.X, ep.Y, ep.Z, w, h, d) {
				continue
			}

			// Score: prefer positions closer to origin (bottom-left-back)
			score := ep.Y*1000 + ep.Z*100 + ep.X*10
			score += (ep.W - w) + (ep.H - h) + (ep.D - d)

			if score < bestScore {
				bestScore = score
				bestPoint = pi
				bestRot = ri
			}
		}
	}

	return bestPoint, bestRot
}

func updateExtremePoints(eps []FreeSpace, placed Placement, box InputBox, placements []Placement) []FreeSpace {
	newPoints := []FreeSpace{
		{X: placed.X + placed.W, Y: placed.Y, Z: placed.Z, W: box.W - (placed.X + placed.W), H: box.H - placed.Y, D: box.D - placed.Z},
		{X: placed.X, Y: placed.Y + placed.H, Z: placed.Z, W: box.W - placed.X, H: box.H - (placed.Y + placed.H), D: box.D - placed.Z},
		{X: placed.X, Y: placed.Y, Z: placed.Z + placed.D, W: box.W - placed.X, H: box.H - placed.Y, D: box.D - (placed.Z + placed.D)},
	}

	var valid []FreeSpace
	for _, ep := range newPoints {
		if ep.X >= box.W || ep.Y >= box.H || ep.Z >= box.D || ep.X < 0 || ep.Y < 0 || ep.Z < 0 {
			continue
		}
		if !isInsidePlacement(ep, placements) {
			valid = append(valid, ep)
		}
	}

	for _, ep := range eps {
		if !isInsidePlaced(ep, placed) {
			valid = append(valid, ep)
		}
	}

	return deduplicatePoints(valid)
}

func isInsidePlacement(ep FreeSpace, placements []Placement) bool {
	for _, p := range placements {
		if ep.X >= p.X && ep.X < p.X+p.W &&
			ep.Y >= p.Y && ep.Y < p.Y+p.H &&
			ep.Z >= p.Z && ep.Z < p.Z+p.D {
			return true
		}
	}
	return false
}

func isInsidePlaced(ep FreeSpace, placed Placement) bool {
	return ep.X >= placed.X && ep.X < placed.X+placed.W &&
		ep.Y >= placed.Y && ep.Y < placed.Y+placed.H &&
		ep.Z >= placed.Z && ep.Z < placed.Z+placed.D
}

func deduplicatePoints(points []FreeSpace) []FreeSpace {
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

func rotations(w, h, d int) [][3]int {
	return [][3]int{
		{w, h, d}, {w, d, h}, {h, w, d},
		{h, d, w}, {d, w, h}, {d, h, w},
	}
}

func fitsInBox(box InputBox, x, y, z, w, h, d int) bool {
	return x >= 0 && y >= 0 && z >= 0 &&
		x+w <= box.W && y+h <= box.H && z+d <= box.D
}

func hasOverlap(placements []Placement, x, y, z, w, h, d int) bool {
	for _, p := range placements {
		if boxesOverlap(p, x, y, z, w, h, d) {
			return true
		}
	}
	return false
}

func boxesOverlap(p Placement, x, y, z, w, h, d int) bool {
	return p.X < x+w && p.X+p.W > x &&
		p.Y < y+h && p.Y+p.H > y &&
		p.Z < z+d && p.Z+p.D > z
}
