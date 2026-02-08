# macoma

A CLI tool that converts a colored drawing into a **magic coloring** — a color-by-numbers image where colors are replaced by numbered zones, with a legend mapping each number to its color.

## Installation

```bash
go build -o macoma .
```

## Usage

```bash
macoma --in=<input> --out=<output> [options]
```

### Options

| Flag | Description | Default |
|------|-------------|---------|
| `--in` | Path to input image (PNG, JPEG, WEBP) | *required* |
| `--out` | Path to output image (must be `.png`) | *required* |
| `--delimiter-color` | Hex color of the drawing delimiter lines | `#000` |
| `--delimiter-tolerance` | Tolerance % for delimiter color (0–100) | `10` |
| `--max-colors` | Max colors in output (0 = unlimited) | `10` |

### Example

```bash
macoma --in=drawing.png --out=coloring.png --delimiter-color=#000 --delimiter-tolerance=10 --max-colors=15
```

## How It Works

1. Loads the input image
2. Separates delimiter lines from filler pixels using the delimiter color + tolerance
3. Groups connected filler pixels into zones via flood-fill
4. Computes a weighted mean color per zone
5. Reduces distinct colors to `--max-colors` by iteratively merging closest colors (CIELAB distance)
6. Renders the output: white-filled zones with centered number labels, delimiter lines preserved, and a color legend at the bottom

## Supported Formats

- **Input**: PNG, JPEG, WEBP
- **Output**: PNG
