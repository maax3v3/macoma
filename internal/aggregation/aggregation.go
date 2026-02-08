package aggregation

import (
	"math"

	"github.com/maax3v3/macoma/internal/color"
)

// ColorEntry represents a resulting color with its assigned number.
type ColorEntry struct {
	Number int
	Color  color.RGBA
}

// ColorMap maps each zone ID to a ColorEntry.
type ColorMap struct {
	Entries  []ColorEntry // the distinct palette entries
	ZoneMap  []int        // zoneID -> index into Entries
}

// ReduceColors takes per-zone colors and reduces them to at most maxColors
// distinct colors by iteratively merging the two closest colors (in CIELAB space).
// If maxColors is 0, no reduction is performed.
// Returns a ColorMap that maps each zone to a numbered color entry.
func ReduceColors(zoneColors []color.RGBA, maxColors int) *ColorMap {
	n := len(zoneColors)
	if n == 0 {
		return &ColorMap{}
	}

	// Build initial groups: group zones that already have the exact same color
	type colorGroup struct {
		color   color.RGBA
		zoneIDs []int
		weights []int // pixel count per zone (here we treat each zone equally with weight 1)
	}

	groupIndex := make(map[color.RGBA]int)
	var groups []colorGroup

	for i, c := range zoneColors {
		if idx, ok := groupIndex[c]; ok {
			groups[idx].zoneIDs = append(groups[idx].zoneIDs, i)
			groups[idx].weights = append(groups[idx].weights, 1)
		} else {
			groupIndex[c] = len(groups)
			groups = append(groups, colorGroup{
				color:   c,
				zoneIDs: []int{i},
				weights: []int{1},
			})
		}
	}

	// Iteratively merge closest pair until we are within maxColors
	for maxColors > 0 && len(groups) > maxColors {
		// Find the two closest groups
		bestDist := math.MaxFloat64
		bestI, bestJ := 0, 1
		for i := 0; i < len(groups); i++ {
			for j := i + 1; j < len(groups); j++ {
				d := color.DistanceLAB(groups[i].color, groups[j].color)
				if d < bestDist {
					bestDist = d
					bestI = i
					bestJ = j
				}
			}
		}

		// Merge bestJ into bestI
		mergedZones := append(groups[bestI].zoneIDs, groups[bestJ].zoneIDs...)
		mergedWeights := append(groups[bestI].weights, groups[bestJ].weights...)

		// Compute new mean color
		totalWeight := 0
		for _, w := range mergedWeights {
			totalWeight += w
		}
		colors := make([]color.RGBA, 0, len(mergedZones))
		weights := make([]int, 0, len(mergedZones))
		for k, zID := range mergedZones {
			colors = append(colors, zoneColors[zID])
			weights = append(weights, mergedWeights[k])
		}
		groups[bestI] = colorGroup{
			color:   color.WeightedMean(colors, weights),
			zoneIDs: mergedZones,
			weights: mergedWeights,
		}

		// Remove bestJ
		groups = append(groups[:bestJ], groups[bestJ+1:]...)
	}

	// Build the result
	cm := &ColorMap{
		Entries: make([]ColorEntry, len(groups)),
		ZoneMap: make([]int, n),
	}
	for i, g := range groups {
		cm.Entries[i] = ColorEntry{
			Number: i + 1, // 1-based numbering
			Color:  g.color,
		}
		for _, zID := range g.zoneIDs {
			cm.ZoneMap[zID] = i
		}
	}

	return cm
}
