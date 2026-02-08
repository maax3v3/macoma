# macoma

A Go library and CLI tool that converts a colored drawing into a **magic coloring** — a color-by-numbers image where colors are replaced by numbered zones, with a legend mapping each number to its color.

## Installation

### As a library

```bash
go get github.com/maax3v3/macoma
```

### As a CLI

```bash
go build -o macoma ./cmd/macoma
```

## Library Usage

```go
package main

import "github.com/maax3v3/macoma"

func main() {
	// Default: color strategy (detects zones by neighbor color difference)
	opts := macoma.DefaultOptions()
	opts.MaxColors = 15
	err := macoma.ConvertFile("drawing.png", "coloring.png", opts)

	// Or use the border strategy (detects zones by explicit border color)
	opts.DelimiterStrategy = macoma.StrategyBorder
	opts.BorderDelimiterColor = macoma.Color{R: 0, G: 0, B: 0, A: 255}

	// In-memory for more control
	img, _ := macoma.LoadImage("drawing.png")
	result, _ := macoma.Convert(img, opts)
	macoma.SavePNG("coloring.png", result)
}
```

- The `FontRenderer` interface can be implemented to provide custom text rendering (e.g., TTF fonts). Pass it via `Options.Font`.
- Set `Options.DelimiterStrategy` to `macoma.StrategyColor` (default) or `macoma.StrategyBorder`.

## CLI Usage

```bash
macoma --in=<input> --out=<output> [options]
```

### Options

| Flag | Description | Default |
|------|-------------|---------|
| `--in` | Path to input image (PNG, JPEG, WEBP) | *required* |
| `--out` | Path to output image (must be `.png`) | *required* |
| `--delimiter-strategy` | `color` (neighbor difference) or `border` (explicit border color) | `color` |
| `--border-delimiter-color` | Hex color of delimiter lines (border strategy only) | `#000` |
| `--border-delimiter-tolerance` | Tolerance % for border color matching, 0–100 (border strategy only) | `10` |
| `--color-delimiter-tolerance` | Color difference threshold %, 0–100 (color strategy only) | `10` |
| `--max-colors` | Max colors in output (0 = unlimited) | `10` |

### Examples

```bash
# Color strategy (default): zones detected by color differences between neighbors
macoma --in=drawing.png --out=coloring.png --color-delimiter-tolerance=10 --max-colors=15

# Border strategy: zones detected by matching explicit border color
macoma --in=drawing.png --out=coloring.png --delimiter-strategy=border --border-delimiter-color=#000 --border-delimiter-tolerance=10
```

## How It Works

1. Loads the input image
2. Detects zone boundaries using the chosen strategy:
   - **color** (default): marks pixels as delimiters when they differ significantly from a neighbor
   - **border**: matches pixels against a specific border color within a tolerance
3. Groups connected non-delimiter pixels into zones via flood-fill
4. Computes a weighted mean color per zone
5. Reduces distinct colors to `--max-colors` by iteratively merging closest colors (CIELAB distance)
6. Renders the output: white-filled zones with centered number labels, delimiter lines preserved, and a color legend at the bottom

## Supported Formats

- **Input**: PNG, JPEG, WEBP
- **Output**: PNG
