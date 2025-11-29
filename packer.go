package main

import (
	"fmt"
	"math"
	"sort"
)

// --- Data Structures ---

type InputItem struct {
	ID       string  `json:"id"`
	W        int     `json:"w"`
	H        int     `json:"h"`
	D        int     `json:"d"`
	Quantity int     `json:"quantity"`
	Weight   float64 `json:"weight,omitempty"`   // Weight in kg
	Fragile  bool    `json:"fragile,omitempty"`  // Cannot be placed under other items
	Priority int     `json:"priority,omitempty"` // Higher priority packed first
}

type InputBox struct {
	ID        string  `json:"id"`
	W         int     `json:"w"`
	H         int     `json:"h"`
	D         int     `json:"d"`
	MaxWeight float64 `json:"max_weight,omitempty"` // Maximum weight capacity in kg
}

type PackedBox struct {
	BoxID    string      `json:"box_id"`
	Contents []Placement `json:"contents"`
}

type Placement struct {
	ItemID  string  `json:"item_id"`
	X       int     `json:"x"`
	Y       int     `json:"y"`
	Z       int     `json:"z"`
	W       int     `json:"w"`
	H       int     `json:"h"`
	D       int     `json:"d"`
	Weight  float64 `json:"weight,omitempty"`
	Fragile bool    `json:"fragile,omitempty"`
}

// ExtremePoint represents a candidate position for placing items
// Using the Extreme Points algorithm instead of free space splitting
type ExtremePoint struct {
	X, Y, Z int
}

// PackingState holds the current state of packing into a single box
type PackingState struct {
	Placements     []Placement
	ExtremePoints  []ExtremePoint
	CurrentWeight  float64
	UsedVolume     int
	Box            InputBox
	MaxHeight      int // Track max height for layer scoring
}

// Internal item representation for packing (handling quantity)
type itemToPack struct {
	InputItem
	Volume      int
	MaxDim      int
	MinDim      int
	SurfaceArea int
	Rotations   [][3]int // Pre-computed rotations
}

// --- Sorting ---

type SortStrategy string

const (
	SortByVolume      SortStrategy = "volume"
	SortByLongestDim  SortStrategy = "longest_dim"
	SortBySurfaceArea SortStrategy = "surface_area"
)

// Support threshold - items must have at least this fraction of base supported
const MinSupportRatio = 0.6

type byVolumeDesc []itemToPack

func (a byVolumeDesc) Len() int      { return len(a) }
func (a byVolumeDesc) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byVolumeDesc) Less(i, j int) bool {
	// Priority first
	if a[i].Priority != a[j].Priority {
		return a[i].Priority > a[j].Priority
	}
	if a[i].Volume != a[j].Volume {
		return a[i].Volume > a[j].Volume
	}
	return a[i].MaxDim > a[j].MaxDim
}

type byLongestDimDesc []itemToPack

func (a byLongestDimDesc) Len() int      { return len(a) }
func (a byLongestDimDesc) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byLongestDimDesc) Less(i, j int) bool {
	if a[i].Priority != a[j].Priority {
		return a[i].Priority > a[j].Priority
	}
	if a[i].MaxDim != a[j].MaxDim {
		return a[i].MaxDim > a[j].MaxDim
	}
	return a[i].Volume > a[j].Volume
}

type bySurfaceAreaDesc []itemToPack

func (a bySurfaceAreaDesc) Len() int      { return len(a) }
func (a bySurfaceAreaDesc) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a bySurfaceAreaDesc) Less(i, j int) bool {
	if a[i].Priority != a[j].Priority {
		return a[i].Priority > a[j].Priority
	}
	if a[i].SurfaceArea != a[j].SurfaceArea {
		return a[i].SurfaceArea > a[j].SurfaceArea
	}
	return a[i].Volume > a[j].Volume
}

type byBoxVolumeAsc []InputBox

func (a byBoxVolumeAsc) Len() int      { return len(a) }
func (a byBoxVolumeAsc) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byBoxVolumeAsc) Less(i, j int) bool {
	return (a[i].W * a[i].H * a[i].D) < (a[j].W * a[j].H * a[j].D)
}

// --- Core Logic ---

// Constants for validation
const (
	MaxDimension = 10000
	MaxQuantity  = 1000
)

// ValidateInputs checks for invalid dimensions and constraints with tier-based limits
func ValidateInputs(items []InputItem, boxes []InputBox, maxItems, maxBoxes int) error {
	if len(items) == 0 {
		return fmt.Errorf("no items provided")
	}
	if len(items) > maxItems {
		return fmt.Errorf("too many items: %d (max %d)", len(items), maxItems)
	}
	if len(boxes) == 0 {
		return fmt.Errorf("no boxes provided")
	}
	if len(boxes) > maxBoxes {
		return fmt.Errorf("too many boxes: %d (max %d)", len(boxes), maxBoxes)
	}

	totalItems := 0
	for i, item := range items {
		if item.W <= 0 || item.H <= 0 || item.D <= 0 {
			return fmt.Errorf("item %d (%s): dimensions must be positive", i, item.ID)
		}
		if item.W > MaxDimension || item.H > MaxDimension || item.D > MaxDimension {
			return fmt.Errorf("item %d (%s): dimension exceeds maximum %d", i, item.ID, MaxDimension)
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("item %d (%s): quantity must be positive", i, item.ID)
		}
		if item.Quantity > MaxQuantity {
			return fmt.Errorf("item %d (%s): quantity exceeds maximum %d", i, item.ID, MaxQuantity)
		}
		if item.Weight < 0 {
			return fmt.Errorf("item %d (%s): weight cannot be negative", i, item.ID)
		}
		totalItems += item.Quantity
	}

	if totalItems > maxItems {
		return fmt.Errorf("total items after quantity expansion: %d (max %d)", totalItems, maxItems)
	}

	for i, box := range boxes {
		if box.W <= 0 || box.H <= 0 || box.D <= 0 {
			return fmt.Errorf("box %d (%s): dimensions must be positive", i, box.ID)
		}
		if box.W > MaxDimension || box.H > MaxDimension || box.D > MaxDimension {
			return fmt.Errorf("box %d (%s): dimension exceeds maximum %d", i, box.ID, MaxDimension)
		}
		if box.MaxWeight < 0 {
			return fmt.Errorf("box %d (%s): max weight cannot be negative", i, box.ID)
		}
	}

	return nil
}

// Pack uses multi-start optimization to find the best packing
func Pack(inputItems []InputItem, availableBoxes []InputBox) ([]PackedBox, []InputItem) {
	return PackMultiStart(inputItems, availableBoxes)
}

// PackMultiStart tries multiple sorting strategies and returns the best result
func PackMultiStart(inputItems []InputItem, availableBoxes []InputBox) ([]PackedBox, []InputItem) {
	strategies := []SortStrategy{SortByVolume, SortByLongestDim, SortBySurfaceArea}

	var bestPackedBoxes []PackedBox
	var bestUnpackedItems []InputItem
	bestUtilization := -1.0

	for _, strategy := range strategies {
		packedBoxes, unpackedItems := PackWithStrategy(inputItems, availableBoxes, strategy)

		// Calculate total utilization
		totalBoxVol := 0
		totalItemVol := 0
		boxMap := make(map[string]InputBox)
		for _, box := range availableBoxes {
			boxMap[box.ID] = box
		}

		for _, pb := range packedBoxes {
			if b, ok := boxMap[pb.BoxID]; ok {
				totalBoxVol += b.W * b.H * b.D
			}
			for _, item := range pb.Contents {
				totalItemVol += item.W * item.H * item.D
			}
		}

		utilization := 0.0
		if totalBoxVol > 0 {
			utilization = float64(totalItemVol) / float64(totalBoxVol)
		}

		// Prefer solutions that pack more items, then higher utilization
		unpackedCount := 0
		for _, u := range unpackedItems {
			unpackedCount += u.Quantity
		}
		bestUnpackedCount := 0
		for _, u := range bestUnpackedItems {
			bestUnpackedCount += u.Quantity
		}

		isBetter := false
		if bestPackedBoxes == nil {
			isBetter = true
		} else if unpackedCount < bestUnpackedCount {
			isBetter = true
		} else if unpackedCount == bestUnpackedCount && utilization > bestUtilization {
			isBetter = true
		}

		if isBetter {
			bestPackedBoxes = packedBoxes
			bestUnpackedItems = unpackedItems
			bestUtilization = utilization
		}
	}

	return bestPackedBoxes, bestUnpackedItems
}

// PackWithStrategy packs items using Extreme Points algorithm with specified sorting
func PackWithStrategy(inputItems []InputItem, availableBoxes []InputBox, strategy SortStrategy) ([]PackedBox, []InputItem) {
	// 1. Expand items based on quantity and calculate properties
	items := expandItems(inputItems)

	// 2. Sort items by chosen strategy
	sortItems(items, strategy)

	// 3. Sort boxes by volume (smallest first)
	sort.Sort(byBoxVolumeAsc(availableBoxes))

	var packedBoxes []PackedBox
	remainingItems := items

	// 4. Packing Loop using Extreme Points
	for len(remainingItems) > 0 {
		bestBoxIndex := -1
		var bestState *PackingState
		var bestIsPacked []bool
		bestPackedVol := -1

		// Try each available box type
		for i, box := range availableBoxes {
			state, isPacked := packIntoBoxEP(remainingItems, box)

			if state.UsedVolume > 0 {
				boxVol := box.W * box.H * box.D
				if bestBoxIndex == -1 || state.UsedVolume > bestPackedVol {
					bestBoxIndex = i
					bestState = state
					bestIsPacked = isPacked
					bestPackedVol = state.UsedVolume
				} else if state.UsedVolume == bestPackedVol {
					// Tie-breaker: Pick smaller box
					bestBoxVol := availableBoxes[bestBoxIndex].W * availableBoxes[bestBoxIndex].H * availableBoxes[bestBoxIndex].D
					if boxVol < bestBoxVol {
						bestBoxIndex = i
						bestState = state
						bestIsPacked = isPacked
						bestPackedVol = state.UsedVolume
					}
				}
			}
		}

		if bestBoxIndex == -1 {
			break
		}

		// Commit the best box
		packedBoxes = append(packedBoxes, PackedBox{
			BoxID:    availableBoxes[bestBoxIndex].ID,
			Contents: bestState.Placements,
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

	// Aggregate unpacked items back to original quantities
	unpackedItems := aggregateItems(remainingItems)

	return packedBoxes, unpackedItems
}

// expandItems converts InputItems to itemToPack with pre-computed properties
func expandItems(inputItems []InputItem) []itemToPack {
	var items []itemToPack
	for _, item := range inputItems {
		for i := 0; i < item.Quantity; i++ {
			dims := []int{item.W, item.H, item.D}
			sort.Ints(dims)
			minDim := dims[0]
			maxDim := dims[2]

			surfaceArea := 2 * (item.W*item.H + item.W*item.D + item.H*item.D)
			items = append(items, itemToPack{
				InputItem:   item,
				Volume:      item.W * item.H * item.D,
				MaxDim:      maxDim,
				MinDim:      minDim,
				SurfaceArea: surfaceArea,
				Rotations:   getRotations(item.W, item.H, item.D),
			})
		}
	}
	return items
}

// sortItems sorts the items slice based on strategy
func sortItems(items []itemToPack, strategy SortStrategy) {
	switch strategy {
	case SortByLongestDim:
		sort.Sort(byLongestDimDesc(items))
	case SortBySurfaceArea:
		sort.Sort(bySurfaceAreaDesc(items))
	default:
		sort.Sort(byVolumeDesc(items))
	}
}

// aggregateItems combines expanded items back to original format with quantities
func aggregateItems(items []itemToPack) []InputItem {
	itemMap := make(map[string]*InputItem)
	for _, item := range items {
		if existing, ok := itemMap[item.ID]; ok {
			existing.Quantity++
		} else {
			newItem := item.InputItem
			newItem.Quantity = 1
			itemMap[item.ID] = &newItem
		}
	}

	var result []InputItem
	for _, item := range itemMap {
		result = append(result, *item)
	}
	return result
}

// packIntoBoxEP uses Extreme Points algorithm to pack items into a box
func packIntoBoxEP(items []itemToPack, box InputBox) (*PackingState, []bool) {
	state := &PackingState{
		Placements:    make([]Placement, 0),
		ExtremePoints: []ExtremePoint{{X: 0, Y: 0, Z: 0}}, // Start with origin
		CurrentWeight: 0,
		UsedVolume:    0,
		Box:           box,
		MaxHeight:     0,
	}

	isPacked := make([]bool, len(items))

	for i, item := range items {
		// Check weight constraint
		if box.MaxWeight > 0 && state.CurrentWeight+item.Weight > box.MaxWeight {
			continue
		}

		placed := tryPlaceItem(state, item)
		if placed {
			isPacked[i] = true
		}
	}

	return state, isPacked
}

// tryPlaceItem attempts to place an item at the best extreme point
func tryPlaceItem(state *PackingState, item itemToPack) bool {
	bestEPIndex := -1
	bestRotation := -1
	bestScore := math.MaxFloat64
	var bestPlacement Placement

	// Try all extreme points
	for epIdx, ep := range state.ExtremePoints {
		// Try all rotations
		for r, dim := range item.Rotations {
			w, h, d := dim[0], dim[1], dim[2]

			// Check if item fits within box bounds
			if ep.X+w > state.Box.W || ep.Y+h > state.Box.H || ep.Z+d > state.Box.D {
				continue
			}

			// Check for collisions with existing placements
			if collidesWithPlacements(state.Placements, ep.X, ep.Y, ep.Z, w, h, d) {
				continue
			}

			// Check support constraint (60% of base must be supported)
			if ep.Y > 0 && !hasAdequateSupport(state.Placements, ep.X, ep.Y, ep.Z, w, d, state.Box) {
				continue
			}

			// Check fragile constraint
			if item.Fragile && hasItemsAbovePosition(state.Placements, ep.X, ep.Y, ep.Z, w, h, d) {
				continue
			}

			// Calculate DBLF score (Deepest-Bottom-Left-Fill)
			score := calculatePlacementScore(state, ep, w, h, d)

			if score < bestScore {
				bestScore = score
				bestEPIndex = epIdx
				bestRotation = r
				bestPlacement = Placement{
					ItemID:  item.ID,
					X:       ep.X,
					Y:       ep.Y,
					Z:       ep.Z,
					W:       w,
					H:       h,
					D:       d,
					Weight:  item.Weight,
					Fragile: item.Fragile,
				}
			}
		}
	}

	if bestEPIndex == -1 {
		return false
	}

	// Place the item
	state.Placements = append(state.Placements, bestPlacement)
	state.UsedVolume += item.Volume
	state.CurrentWeight += item.Weight

	// Update max height for layer tracking
	itemTop := bestPlacement.Y + bestPlacement.H
	if itemTop > state.MaxHeight {
		state.MaxHeight = itemTop
	}

	// Generate new extreme points from this placement
	w := item.Rotations[bestRotation][0]
	h := item.Rotations[bestRotation][1]
	d := item.Rotations[bestRotation][2]
	ep := state.ExtremePoints[bestEPIndex]

	newEPs := generateExtremePoints(ep, w, h, d, state)

	// Remove the used extreme point
	state.ExtremePoints = append(state.ExtremePoints[:bestEPIndex], state.ExtremePoints[bestEPIndex+1:]...)

	// Add new extreme points (avoiding duplicates and invalid ones)
	for _, newEP := range newEPs {
		if isValidExtremePoint(newEP, state) {
			state.ExtremePoints = appendUniqueEP(state.ExtremePoints, newEP)
		}
	}

	// Clean up extreme points that are now inside placed items
	state.ExtremePoints = filterValidExtremePoints(state)

	return true
}

// calculatePlacementScore implements DBLF with layer-based scoring
func calculatePlacementScore(state *PackingState, ep ExtremePoint, w, h, d int) float64 {
	// DBLF: Prioritize Y (bottom), then Z (back), then X (left)
	// Lower scores are better

	// Base DBLF score
	score := float64(ep.Y)*1000 + float64(ep.Z)*100 + float64(ep.X)*10

	// Layer completion bonus: prefer positions that align with current layer height
	if state.MaxHeight > 0 && ep.Y == state.MaxHeight {
		score -= 500 // Bonus for building on current layer
	}

	// Wall contact bonus
	wallContact := 0
	if ep.X == 0 {
		wallContact++
	}
	if ep.Z == 0 {
		wallContact++
	}
	if ep.Y == 0 {
		wallContact++
	}
	if ep.X+w == state.Box.W {
		wallContact++
	}
	if ep.Z+d == state.Box.D {
		wallContact++
	}
	score -= float64(wallContact) * 50

	// Corner bonus (prefer corners)
	corners := 0
	if ep.X == 0 || ep.X+w == state.Box.W {
		corners++
	}
	if ep.Z == 0 || ep.Z+d == state.Box.D {
		corners++
	}
	if corners == 2 {
		score -= 100
	}

	// Height minimization - keep center of gravity low
	score += float64(ep.Y+h) * 0.5

	return score
}

// generateExtremePoints creates new extreme points after placing an item
func generateExtremePoints(ep ExtremePoint, w, h, d int, state *PackingState) []ExtremePoint {
	newEPs := make([]ExtremePoint, 0, 3)

	// EP1: Right of item (X + W, Y, Z)
	newEPs = append(newEPs, ExtremePoint{X: ep.X + w, Y: ep.Y, Z: ep.Z})

	// EP2: On top of item (X, Y + H, Z)
	newEPs = append(newEPs, ExtremePoint{X: ep.X, Y: ep.Y + h, Z: ep.Z})

	// EP3: In front of item (X, Y, Z + D)
	newEPs = append(newEPs, ExtremePoint{X: ep.X, Y: ep.Y, Z: ep.Z + d})

	return newEPs
}

// isValidExtremePoint checks if an EP is within bounds and not inside an item
func isValidExtremePoint(ep ExtremePoint, state *PackingState) bool {
	// Check bounds
	if ep.X < 0 || ep.Y < 0 || ep.Z < 0 {
		return false
	}
	if ep.X >= state.Box.W || ep.Y >= state.Box.H || ep.Z >= state.Box.D {
		return false
	}

	// Check if point is inside any placed item
	for _, p := range state.Placements {
		if ep.X >= p.X && ep.X < p.X+p.W &&
			ep.Y >= p.Y && ep.Y < p.Y+p.H &&
			ep.Z >= p.Z && ep.Z < p.Z+p.D {
			return false
		}
	}

	return true
}

// appendUniqueEP adds an EP if it doesn't already exist
func appendUniqueEP(eps []ExtremePoint, newEP ExtremePoint) []ExtremePoint {
	for _, ep := range eps {
		if ep.X == newEP.X && ep.Y == newEP.Y && ep.Z == newEP.Z {
			return eps
		}
	}
	return append(eps, newEP)
}

// filterValidExtremePoints removes EPs that are inside placed items
func filterValidExtremePoints(state *PackingState) []ExtremePoint {
	valid := make([]ExtremePoint, 0, len(state.ExtremePoints))
	for _, ep := range state.ExtremePoints {
		if isValidExtremePoint(ep, state) {
			valid = append(valid, ep)
		}
	}
	return valid
}

// collidesWithPlacements checks if a proposed placement overlaps existing items
func collidesWithPlacements(placements []Placement, x, y, z, w, h, d int) bool {
	for _, p := range placements {
		if boxesOverlap(x, y, z, w, h, d, p.X, p.Y, p.Z, p.W, p.H, p.D) {
			return true
		}
	}
	return false
}

// boxesOverlap checks if two 3D boxes overlap
func boxesOverlap(x1, y1, z1, w1, h1, d1, x2, y2, z2, w2, h2, d2 int) bool {
	return x1 < x2+w2 && x1+w1 > x2 &&
		y1 < y2+h2 && y1+h1 > y2 &&
		z1 < z2+d2 && z1+d1 > z2
}

// hasAdequateSupport checks if at least MinSupportRatio of the item's base is supported
func hasAdequateSupport(placements []Placement, x, y, z, w, d int, box InputBox) bool {
	// If on the ground, always supported
	if y == 0 {
		return true
	}

	itemBaseArea := w * d
	supportedArea := 0

	// Check each placement that could support this item
	for _, p := range placements {
		// Item must be directly below (top of placed item = bottom of new item)
		if p.Y+p.H != y {
			continue
		}

		// Calculate overlap in XZ plane
		overlapX := max(0, min(x+w, p.X+p.W)-max(x, p.X))
		overlapZ := max(0, min(z+d, p.Z+p.D)-max(z, p.Z))
		supportedArea += overlapX * overlapZ
	}

	// Also count floor support if box bottom is at this level
	// (shouldn't happen since we check y==0 above, but safety)

	supportRatio := float64(supportedArea) / float64(itemBaseArea)
	return supportRatio >= MinSupportRatio
}

// hasItemsAbovePosition checks if placing here would put items above a fragile item
func hasItemsAbovePosition(placements []Placement, x, y, z, w, h, d int) bool {
	// Check if any existing item is above this position
	for _, p := range placements {
		if p.Y >= y+h && overlapsXZ(x, z, w, d, p.X, p.Z, p.W, p.D) {
			return true
		}
	}
	return false
}

func getRotations(w, h, d int) [][3]int {
	// Generate unique rotations (avoid duplicates for cubes)
	rotations := [][3]int{
		{w, h, d},
		{w, d, h},
		{h, w, d},
		{h, d, w},
		{d, w, h},
		{d, h, w},
	}

	// Remove duplicates for items with equal dimensions
	seen := make(map[[3]int]bool)
	unique := make([][3]int, 0, 6)
	for _, r := range rotations {
		if !seen[r] {
			seen[r] = true
			unique = append(unique, r)
		}
	}
	return unique
}

// overlapsXZ checks if two rectangles overlap in the XZ plane
func overlapsXZ(x1, z1, w1, d1, x2, z2, w2, d2 int) bool {
	return x1 < x2+w2 && x1+w1 > x2 && z1 < z2+d2 && z1+d1 > z2
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
