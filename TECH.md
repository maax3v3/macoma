# Macoma — Image Processing Algorithms

This document describes the algorithms used by macoma to convert a source image into a paint-by-numbers coloring page. The pipeline runs in seven sequential steps.

## Pipeline Overview

```
Input Image
    │
    ▼
┌──────────────────────┐
│ 1. Load Image        │  Decode PNG / JPEG / WebP
└──────────┬───────────┘
           ▼
┌──────────────────────┐
│ 2. Detect Delimiters │  Classify pixels as border or filler
└──────────┬───────────┘
           ▼
┌──────────────────────┐
│ 3. Find Zones        │  Flood-fill filler pixels into regions
└──────────┬───────────┘
           ▼
┌──────────────────────┐
│ 4. Compute Colors    │  Weighted mean color per zone
└──────────┬───────────┘
           ▼
┌──────────────────────┐
│ 5. Reduce Colors     │  Agglomerative merging in CIELAB
└──────────┬───────────┘
           ▼
┌──────────────────────┐
│ 6. Render Output     │  White canvas + black borders + labels + legend
└──────────┬───────────┘
           ▼
┌──────────────────────┐
│ 7. Save Image        │  Encode to PNG
└──────────┘───────────┘
```

---

## Step 1 — Image Loading

**Package:** `internal/imaging`

Decodes the input file based on extension. Supported formats:

| Format | Decoder |
|--------|---------|
| PNG    | `image/png` (stdlib) |
| JPEG   | `image/jpeg` (stdlib) |
| WebP   | `golang.org/x/image/webp` |

Path normalization expands `~` to the user home directory and resolves relative paths to absolute.

---

## Step 2 — Delimiter Detection

**Package:** `internal/detection`

A **delimiter pixel** is a pixel that belongs to a zone boundary (the black lines in the final output). Two strategies are available, selected via the `--delimiter-strategy` flag.

### Strategy: `border`

**Implementation:** `BorderDelimiter`

Designed for source images that already contain explicit borders drawn in a known color (e.g. black outlines).

**Algorithm:**

For each pixel `P`:

1. Convert `P` to the internal `RGBA` type.
2. Compute the Euclidean RGB distance to the configured border color:

```
d = √((R₁ - R₂)² + (G₁ - G₂)² + (B₁ - B₂)²)
```

3. If `d ≤ tolerance`, mark `P` as a delimiter.

**Threshold derivation:**

```
threshold = (TolerancePct / 100) × MaxRGBDistance
MaxRGBDistance = √(255² × 3) ≈ 441.67
```

**Complexity:** O(W × H) — one distance computation per pixel.

### Strategy: `color` (default)

**Implementation:** `ColorDelimiter`

Designed for source images without pre-drawn borders (e.g. photographs, digital art). Detects edges by analyzing local color variation.

**Algorithm — Local Range Filter with Chebyshev Distance:**

For each pixel `P(x, y)`:

1. Examine a **5×5 neighborhood** (radius = 2) centered on `P`.
2. Track the per-channel minimum and maximum across all pixels in the window:
   - `minR, maxR, minG, maxG, minB, maxB`
3. Compute the **Chebyshev distance** (maximum per-channel range):

```
maxDiff = max(maxR - minR, maxG - minG, maxB - minB)
```

4. If `maxDiff > threshold`, mark `P` as a delimiter.

**Threshold derivation:**

```
threshold = (TolerancePct / 100) × 255
```

**Why Chebyshev over Euclidean:**

Euclidean distance distributes sensitivity equally across all channels. When two colors differ primarily in a single channel (e.g. black `(0,0,0)` vs dark green `(10,40,10)`), the Euclidean distance may fall below threshold while the green channel alone clearly diverges. Chebyshev distance catches these single-channel differences:

| Colors | Euclidean² | Chebyshev | Detected @ 10%? |
|--------|-----------|-----------|------------------|
| `(0,0,0)` vs `(10,40,10)` | 1800 < 1950 | 40 > 25 | Euclid: ✗, Cheby: ✓ |
| `(100,100,100)` vs `(130,130,130)` | 2700 > 1950 | 30 > 25 | Both: ✓ |

**Why a range filter (not pairwise neighbor comparison):**

A per-pixel neighbor comparison (comparing each pixel to its immediate neighbors) misses anti-aliased edges where each individual pixel-to-pixel step is below threshold but the cumulative change across the transition zone is significant. The 5×5 range filter window spans the entire transition, catching both sides of the boundary in a single measurement. This produces naturally thick (~5 px), continuous, gap-free borders with no need for morphological post-processing.

**Complexity:** O(W × H × 25) — 25 lookups per pixel for the 5×5 window.

### Parallelization

Both strategies use `parallelRows`, which divides the image height into 8 row bands and processes each in a separate goroutine. Workers only write to their own rows, requiring no synchronization.

### Precomputed RGB Buffer

The `ColorDelimiter` precomputes a flat `[]color.RGBA` buffer from the `image.Image` interface. This avoids repeated virtual dispatch on `img.At()` during the inner loop, which is a significant performance gain for large images.

---

## Step 3 — Zone Finding

**Package:** `internal/zone`

Identifies connected regions of filler (non-delimiter) pixels.

**Algorithm — BFS Flood-Fill:**

1. Allocate a label map (`[]int`, same size as image), initialized to `-1`.
2. Scan pixels in raster order (top-to-bottom, left-to-right).
3. For each unlabeled filler pixel, start a BFS flood-fill:
   - Use a FIFO queue seeded with the pixel.
   - Expand to **4-connected neighbors** (up, down, left, right).
   - A neighbor is added if it is within bounds, is not a delimiter, and is unlabeled.
   - All reached pixels are assigned the current zone ID.
4. Increment the zone ID and continue scanning.

**Output:**
- `[]Zone` — each zone stores its ID and a list of all its pixel coordinates.
- `[]int` (label map) — maps each pixel position to its zone index, or `-1` for delimiters.

**Complexity:** O(W × H) — each pixel is visited exactly once.

### Interior Point Computation

For placing zone number labels, each zone computes an **interior point** that is:
1. Inside the zone (not just the centroid, which may fall outside concave shapes).
2. As far from the zone boundary as possible (for readability).
3. As close to the centroid as possible (for visual centering).

**Algorithm:**

1. Compute the geometric centroid of the zone.
2. BFS from all boundary pixels inward to compute a **distance-to-edge map** in O(n).
3. If the centroid has `distance ≥ margin` (15 px for large zones, 5 px for small), use it.
4. Otherwise, find the pixel closest to the centroid with `distance ≥ margin`.
5. Fallback: pick the deepest interior pixel closest to the centroid.

---

## Step 4 — Zone Color Computation

**Package:** `internal/zone`

Computes the representative color for each zone.

**Algorithm:**

For each zone, compute the **arithmetic mean** of all pixel colors:

```
R_zone = (1/N) × Σ R_i
G_zone = (1/N) × Σ G_i
B_zone = (1/N) × Σ B_i
```

Where N is the number of pixels in the zone.

**Parallelization:** Uses a worker pool of 8 goroutines consuming zone indices from a channel.

**Complexity:** O(total pixels across all zones) = O(W × H).

---

## Step 5 — Color Reduction

**Package:** `internal/aggregation`

Reduces the number of distinct colors to at most `maxColors` (configurable, default 10).

**Algorithm — Agglomerative Hierarchical Clustering (CIELAB):**

1. **Initial grouping:** zones with identical RGB colors are grouped together.
2. **Iterative merging:** while `|groups| > maxColors`:
   a. Find the two groups with the smallest **CIELAB Euclidean distance** between their representative colors.
   b. Merge them into one group.
   c. Recompute the representative color as the **weighted mean** (in RGB) of all zone colors in the merged group.
3. Assign a 1-based number to each final group.

**CIELAB Color Space:**

CIELAB (CIE L\*a\*b\*) is a perceptually uniform color space where equal Euclidean distances correspond to roughly equal perceived color differences. The conversion pipeline is:

```
sRGB → Linear RGB → XYZ (D65 illuminant) → CIELAB
```

Key transforms:
- **sRGB to linear:** inverse gamma correction (`v/12.92` for v ≤ 0.04045, else `((v+0.055)/1.055)^2.4`)
- **Linear RGB to XYZ:** 3×3 matrix multiplication (D65 reference white)
- **XYZ to LAB:** cube-root compression with linear segment below `δ = 6/29`

Using CIELAB for merging decisions ensures perceptually similar colors are merged first, preserving visually distinct colors as long as possible.

**Complexity:** O(G² × M) where G is the initial number of distinct colors and M = G − maxColors merge iterations. Each iteration scans all pairs to find the closest.

---

## Step 6 — Rendering

**Package:** `internal/renderer`

Produces the final paint-by-numbers image.

### Canvas Setup

1. Create an RGBA image of size `W × (H + legendHeight)`.
2. Fill entirely with **white** `(255, 255, 255)`.

### Border Drawing

Iterate over all pixels. Where `delimiterMap.At(x, y)` is true, set the pixel to **black** `(0, 0, 0)`. This draws the zone boundaries.

### Zone Number Labels

For each zone:
1. Look up its assigned color number from the `ColorMap`.
2. Compute the zone's interior point (see Step 3).
3. Draw the number string at that position using the `BitmapFont` renderer.

**Font sizing heuristic:**
```
base = min(W, H) / 30
if zones > 50:  base × 0.7
if zones > 200: base × 0.5
clamped to [7, 40], then divided by 4 for in-drawing labels
```

**Bitmap font:** hardcoded 5×7 pixel glyph bitmaps for digits 0–9, scaled by an integer factor. Each "on" bit becomes a `scale × scale` block.

### Legend

Drawn below the main image, separated by a thin gray line.

For each color entry:
1. Draw a **filled circle** (midpoint distance test: `dx² + dy² ≤ r²`).
2. Draw a **circle border** (parametric arc, step = 0.01 radians).
3. Draw the color number centered inside.

Text color is automatically **black** or **white** based on the fill color's relative luminance (`0.2126·R + 0.7152·G + 0.0722·B > 0.5`).

Legend layout adapts to image width, wrapping entries into rows and centering each row.

---

## Step 7 — Image Saving

**Package:** `internal/imaging`

Encodes the rendered `*image.RGBA` to PNG and writes to disk. Path normalization is applied (same as loading).

---

## Performance Summary

| Step | Complexity | Parallelized |
|------|-----------|--------------|
| Load | O(W×H) | No (I/O bound) |
| Delimiter detection | O(W×H×25) for color, O(W×H) for border | Yes (8 row-band workers) |
| Zone finding | O(W×H) | No (sequential BFS) |
| Zone colors | O(W×H) | Yes (8-worker pool) |
| Color reduction | O(G²×M) | No (G typically small) |
| Rendering | O(W×H + Z) | Partial (zone labels parallelized) |
| Save | O(W×H) | No (I/O bound) |

Where W×H = total pixels, G = distinct color count, M = merge iterations, Z = number of zones.
