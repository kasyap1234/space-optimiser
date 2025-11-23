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

type FreeSpace struct {
	X, Y, Z int
	W, H, D int
}

func (fs FreeSpace) Volume() int {
	return fs.W * fs.H * fs.D
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

			// Selection Criteria:
			// 1. Maximize Volume Packed
			// 2. Minimize Box Volume (if packed volume is equal) - implied by sorting boxes asc?
			// Actually, we want the "tightest" fit.
			// If Box A (Vol 100) packs 90, and Box B (Vol 200) packs 90. We prefer Box A.
			// So we can use Utilization = PackedVol / BoxVol.

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

// packIntoBox attempts to pack items into a specific box and returns the result
func packIntoBox(items []itemToPack, box InputBox) ([]Placement, []bool, int) {
	freeSpaces := []FreeSpace{{
		X: 0, Y: 0, Z: 0,
		W: box.W, H: box.H, D: box.D,
	}}

	var placements []Placement
	isPacked := make([]bool, len(items))
	packedVol := 0

	for i, item := range items {
		bestSpaceIndex := -1
		bestRotation := -1 // 0-5
		minVolDiff := math.MaxInt64

		// Check all free spaces
		for si, space := range freeSpaces {
			// Check all 6 rotations
			rotations := getRotations(item.W, item.H, item.D)
			for r, dim := range rotations {
				if dim[0] <= space.W && dim[1] <= space.H && dim[2] <= space.D {
					// Fits!
					// Heuristic: Best Fit (minimize wasted volume in the space)
					volDiff := space.Volume() - item.Volume
					if volDiff < minVolDiff {
						minVolDiff = volDiff
						bestSpaceIndex = si
						bestRotation = r
					}
				}
			}
		}

		if bestSpaceIndex != -1 {
			// Place item
			space := freeSpaces[bestSpaceIndex]
			rotations := getRotations(item.W, item.H, item.D)
			w, h, d := rotations[bestRotation][0], rotations[bestRotation][1], rotations[bestRotation][2]

			placements = append(placements, Placement{
				ItemID: item.ID,
				X:      space.X, Y: space.Y, Z: space.Z,
				W: w, H: h, D: d,
			})
			isPacked[i] = true
			packedVol += item.Volume

			// Remove used space
			freeSpaces[bestSpaceIndex] = freeSpaces[len(freeSpaces)-1]
			freeSpaces = freeSpaces[:len(freeSpaces)-1]

			// Generate new spaces (Split)
			// 1. Right: x+w, remainder width, full height, full depth
			if space.W > w {
				freeSpaces = append(freeSpaces, FreeSpace{
					X: space.X + w, Y: space.Y, Z: space.Z,
					W: space.W - w, H: space.H, D: space.D,
				})
			}
			// 2. Top: x, y+h, remainder height, full depth (constrained width)
			if space.H > h {
				freeSpaces = append(freeSpaces, FreeSpace{
					X: space.X, Y: space.Y + h, Z: space.Z,
					W: w, H: space.H - h, D: space.D,
				})
			}
			// 3. Front: x, y, z+d, remainder depth (constrained width and height)
			if space.D > d {
				freeSpaces = append(freeSpaces, FreeSpace{
					X: space.X, Y: space.Y, Z: space.Z + d,
					W: w, H: h, D: space.D - d,
				})
			}

			// Defragmentation (Merge)
			freeSpaces = mergeFreeSpaces(freeSpaces)
		}
	}

	return placements, isPacked, packedVol
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

// mergeFreeSpaces iterates through the list and merges adjacent spaces
func mergeFreeSpaces(spaces []FreeSpace) []FreeSpace {
	for {
		merged := false
		for i := 0; i < len(spaces); i++ {
			for j := i + 1; j < len(spaces); j++ {
				s1 := spaces[i]
				s2 := spaces[j]

				// Check if they can be merged
				// Case 1: Same X, W, Z, D. Adjacent in Y (Top/Bottom)
				if s1.X == s2.X && s1.W == s2.W && s1.Z == s2.Z && s1.D == s2.D {
					if s1.Y+s1.H == s2.Y { // s1 below s2
						spaces[i].H += s2.H
						spaces = remove(spaces, j)
						merged = true
						break
					} else if s2.Y+s2.H == s1.Y { // s2 below s1
						spaces[i].Y = s2.Y
						spaces[i].H += s2.H
						spaces = remove(spaces, j)
						merged = true
						break
					}
				}

				// Case 2: Same Y, H, Z, D. Adjacent in X (Left/Right)
				if s1.Y == s2.Y && s1.H == s2.H && s1.Z == s2.Z && s1.D == s2.D {
					if s1.X+s1.W == s2.X { // s1 left of s2
						spaces[i].W += s2.W
						spaces = remove(spaces, j)
						merged = true
						break
					} else if s2.X+s2.W == s1.X { // s2 left of s1
						spaces[i].X = s2.X
						spaces[i].W += s2.W
						spaces = remove(spaces, j)
						merged = true
						break
					}
				}

				// Case 3: Same X, W, Y, H. Adjacent in Z (Front/Back)
				if s1.X == s2.X && s1.W == s2.W && s1.Y == s2.Y && s1.H == s2.H {
					if s1.Z+s1.D == s2.Z { // s1 behind s2
						spaces[i].D += s2.D
						spaces = remove(spaces, j)
						merged = true
						break
					} else if s2.Z+s2.D == s1.Z { // s2 behind s1
						spaces[i].Z = s2.Z
						spaces[i].D += s2.D
						spaces = remove(spaces, j)
						merged = true
						break
					}
				}
			}
			if merged {
				break
			}
		}
		if !merged {
			break
		}
	}
	return spaces
}

func remove(slice []FreeSpace, s int) []FreeSpace {
	return append(slice[:s], slice[s+1:]...)
}
