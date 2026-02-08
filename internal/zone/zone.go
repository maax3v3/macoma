package zone

import (
	"image"

	"github.com/maax3v3/macoma/internal/color"
	"github.com/maax3v3/macoma/internal/detection"
)

// Zone represents a connected region of filler (non-delimiter) pixels.
type Zone struct {
	ID     int
	Pixels []image.Point // all pixel coordinates in this zone
}

// Centroid returns the geometric center of the zone.
func (z *Zone) Centroid() image.Point {
	if len(z.Pixels) == 0 {
		return image.Point{}
	}
	var sx, sy int
	for _, p := range z.Pixels {
		sx += p.X
		sy += p.Y
	}
	return image.Point{
		X: sx / len(z.Pixels),
		Y: sy / len(z.Pixels),
	}
}

// InteriorPoint returns a point guaranteed to be inside the zone.
// It computes the centroid and, if the centroid falls outside the zone
// (e.g. for concave shapes), returns the zone pixel closest to the centroid
// while maintaining a margin from the zone boundary.
//
// Uses BFS from boundary pixels to compute distance-to-edge in O(n),
// making it independent of the margin value.
func (z *Zone) InteriorPoint() image.Point {
	if len(z.Pixels) == 0 {
		return image.Point{}
	}
	centroid := z.Centroid()

	// Build a set for O(1) membership check
	members := make(map[image.Point]struct{}, len(z.Pixels))
	for _, p := range z.Pixels {
		members[p] = struct{}{}
	}

	// Desired margin from zone boundary
	margin := 15
	if len(z.Pixels) < 100 {
		margin = 5
	}

	// Compute distance-to-boundary for every zone pixel via BFS.
	// Boundary pixels are zone pixels that have at least one 4-neighbor
	// outside the zone. Their distance is 0. We propagate inward.
	dist := make(map[image.Point]int, len(z.Pixels))
	var queue []image.Point
	dirs := [4]image.Point{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}

	for _, p := range z.Pixels {
		isBoundary := false
		for _, d := range dirs {
			n := image.Point{X: p.X + d.X, Y: p.Y + d.Y}
			if _, ok := members[n]; !ok {
				isBoundary = true
				break
			}
		}
		if isBoundary {
			dist[p] = 0
			queue = append(queue, p)
		} else {
			dist[p] = -1 // unvisited
		}
	}

	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		nd := dist[p] + 1
		for _, d := range dirs {
			n := image.Point{X: p.X + d.X, Y: p.Y + d.Y}
			if dd, ok := dist[n]; ok && dd == -1 {
				dist[n] = nd
				queue = append(queue, n)
			}
		}
	}

	// Check centroid first
	if d, ok := dist[centroid]; ok && d >= margin {
		return centroid
	}

	// Find the zone pixel closest to centroid with distance >= margin
	bestSq := int(^uint(0) >> 1)
	best := image.Point{}
	found := false
	for _, p := range z.Pixels {
		if dist[p] < margin {
			continue
		}
		dx := p.X - centroid.X
		dy := p.Y - centroid.Y
		sq := dx*dx + dy*dy
		if sq < bestSq {
			bestSq = sq
			best = p
			found = true
		}
	}
	if found {
		return best
	}

	// No pixel meets the full margin — pick the deepest interior pixel
	// closest to the centroid (maximise distance-to-edge, break ties by
	// proximity to centroid).
	bestEdgeDist := -1
	bestSq = int(^uint(0) >> 1)
	for _, p := range z.Pixels {
		d := dist[p]
		dx := p.X - centroid.X
		dy := p.Y - centroid.Y
		sq := dx*dx + dy*dy
		if d > bestEdgeDist || (d == bestEdgeDist && sq < bestSq) {
			bestEdgeDist = d
			bestSq = sq
			best = p
		}
	}
	return best
}

// FindZones performs flood-fill on filler pixels to identify connected zones.
// Returns a slice of zones and a label map (same dimensions as the delimiter map)
// where each filler pixel's value is its zone index (0-based), and delimiter
// pixels have value -1.
func FindZones(dm *detection.Map) ([]Zone, []int) {
	w, h := dm.Width, dm.Height
	labels := make([]int, w*h)
	for i := range labels {
		labels[i] = -1
	}

	var zones []Zone
	zoneID := 0

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			if dm.IsDelimiter[idx] || labels[idx] != -1 {
				continue
			}
			// BFS flood-fill
			zone := Zone{ID: zoneID}
			queue := []image.Point{{X: x, Y: y}}
			labels[idx] = zoneID

			for len(queue) > 0 {
				p := queue[0]
				queue = queue[1:]
				zone.Pixels = append(zone.Pixels, p)

				// 4-connected neighbors
				for _, d := range [4]image.Point{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
					nx, ny := p.X+d.X, p.Y+d.Y
					if nx < 0 || nx >= w || ny < 0 || ny >= h {
						continue
					}
					ni := ny*w + nx
					if dm.IsDelimiter[ni] || labels[ni] != -1 {
						continue
					}
					labels[ni] = zoneID
					queue = append(queue, image.Point{X: nx, Y: ny})
				}
			}

			zones = append(zones, zone)
			zoneID++
		}
	}

	return zones, labels
}

// ZoneColors holds the aggregated color for each zone.
type ZoneColors struct {
	Colors []color.RGBA // indexed by zone ID
}

// ComputeZoneColors computes the weighted mean color for each zone by
// reading pixel colors from the source image.
func ComputeZoneColors(zones []Zone, img image.Image) *ZoneColors {
	zc := &ZoneColors{
		Colors: make([]color.RGBA, len(zones)),
	}

	// Process zones in parallel
	type result struct {
		idx int
		c   color.RGBA
	}
	ch := make(chan result, len(zones))

	// Use a simple worker pool
	work := make(chan int, len(zones))
	for i := range zones {
		work <- i
	}
	close(work)

	numWorkers := 8
	if len(zones) < numWorkers {
		numWorkers = len(zones)
	}

	for w := 0; w < numWorkers; w++ {
		go func() {
			for i := range work {
				z := &zones[i]
				colors := make([]color.RGBA, len(z.Pixels))
				for j, p := range z.Pixels {
					colors[j] = color.FromStdColor(img.At(p.X, p.Y))
				}
				ch <- result{idx: i, c: color.WeightedMean(colors, nil)}
			}
		}()
	}

	for range zones {
		r := <-ch
		zc.Colors[r.idx] = r.c
	}

	return zc
}
