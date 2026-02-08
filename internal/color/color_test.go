package color

import (
	"image/color"
	"math"
	"testing"
)

func TestParseHex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    RGBA
		wantErr bool
	}{
		{
			name:  "6-digit black with hash",
			input: "#000000",
			want:  RGBA{0, 0, 0, 255},
		},
		{
			name:  "6-digit white with hash",
			input: "#FFFFFF",
			want:  RGBA{255, 255, 255, 255},
		},
		{
			name:  "6-digit lowercase",
			input: "#ff00ff",
			want:  RGBA{255, 0, 255, 255},
		},
		{
			name:  "6-digit without hash",
			input: "AB12CD",
			want:  RGBA{0xAB, 0x12, 0xCD, 255},
		},
		{
			name:  "3-digit black",
			input: "#000",
			want:  RGBA{0, 0, 0, 255},
		},
		{
			name:  "3-digit white",
			input: "#FFF",
			want:  RGBA{255, 255, 255, 255},
		},
		{
			name:  "3-digit color",
			input: "#F0A",
			want:  RGBA{0xFF, 0x00, 0xAA, 255},
		},
		{
			name:  "3-digit without hash",
			input: "abc",
			want:  RGBA{0xAA, 0xBB, 0xCC, 255},
		},
		{
			name:    "invalid length 1",
			input:   "#F",
			wantErr: true,
		},
		{
			name:    "invalid length 4",
			input:   "#FFFF",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "non-hex characters 6-digit",
			input:   "#ZZZZZZ",
			wantErr: true,
		},
		{
			name:    "non-hex characters 3-digit",
			input:   "#GGG",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHex(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFromStdColor(t *testing.T) {
	tests := []struct {
		name  string
		input color.Color
		want  RGBA
	}{
		{"opaque red", color.RGBA{255, 0, 0, 255}, RGBA{255, 0, 0, 255}},
		{"opaque white", color.White, RGBA{255, 255, 255, 255}},
		{"opaque black", color.Black, RGBA{0, 0, 0, 255}},
		{"transparent", color.RGBA{0, 0, 0, 0}, RGBA{0, 0, 0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromStdColor(tt.input)
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestToStdColor(t *testing.T) {
	c := RGBA{10, 20, 30, 255}
	std := c.ToStdColor()
	if std.R != 10 || std.G != 20 || std.B != 30 || std.A != 255 {
		t.Errorf("got %+v, want {10,20,30,255}", std)
	}
}

func TestRoundTripStdColor(t *testing.T) {
	original := RGBA{42, 128, 200, 255}
	roundTripped := FromStdColor(original.ToStdColor())
	if roundTripped != original {
		t.Errorf("round-trip failed: got %+v, want %+v", roundTripped, original)
	}
}

func TestToLAB(t *testing.T) {
	tests := []struct {
		name string
		c    RGBA
		wantL, wantA, wantB float64
		tolerance            float64
	}{
		{
			name: "black",
			c:    RGBA{0, 0, 0, 255},
			wantL: 0, wantA: 0, wantB: 0,
			tolerance: 0.5,
		},
		{
			name: "white",
			c:    RGBA{255, 255, 255, 255},
			wantL: 100, wantA: 0, wantB: 0,
			tolerance: 0.5,
		},
		{
			name: "red has positive a*",
			c:    RGBA{255, 0, 0, 255},
			wantL: 53.2, wantA: 80.1, wantB: 67.2,
			tolerance: 1.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lab := tt.c.ToLAB()
			if math.Abs(lab.L-tt.wantL) > tt.tolerance {
				t.Errorf("L: got %.2f, want ~%.2f", lab.L, tt.wantL)
			}
			if math.Abs(lab.A-tt.wantA) > tt.tolerance {
				t.Errorf("A: got %.2f, want ~%.2f", lab.A, tt.wantA)
			}
			if math.Abs(lab.B-tt.wantB) > tt.tolerance {
				t.Errorf("B: got %.2f, want ~%.2f", lab.B, tt.wantB)
			}
		})
	}
}

func TestDistanceLAB(t *testing.T) {
	t.Run("identical colors have zero distance", func(t *testing.T) {
		c := RGBA{100, 150, 200, 255}
		if d := DistanceLAB(c, c); d != 0 {
			t.Errorf("got %f, want 0", d)
		}
	})

	t.Run("black vs white is large", func(t *testing.T) {
		d := DistanceLAB(RGBA{0, 0, 0, 255}, RGBA{255, 255, 255, 255})
		if d < 90 {
			t.Errorf("black-white distance too small: %f", d)
		}
	})

	t.Run("symmetry", func(t *testing.T) {
		a := RGBA{255, 0, 0, 255}
		b := RGBA{0, 0, 255, 255}
		if DistanceLAB(a, b) != DistanceLAB(b, a) {
			t.Error("distance is not symmetric")
		}
	})

	t.Run("similar colors closer than dissimilar", func(t *testing.T) {
		red := RGBA{255, 0, 0, 255}
		orange := RGBA{255, 128, 0, 255}
		blue := RGBA{0, 0, 255, 255}
		dSimilar := DistanceLAB(red, orange)
		dDissimilar := DistanceLAB(red, blue)
		if dSimilar >= dDissimilar {
			t.Errorf("expected red-orange (%f) < red-blue (%f)", dSimilar, dDissimilar)
		}
	})
}

func TestDistanceRGB(t *testing.T) {
	t.Run("identical colors have zero distance", func(t *testing.T) {
		c := RGBA{50, 50, 50, 255}
		if d := DistanceRGB(c, c); d != 0 {
			t.Errorf("got %f, want 0", d)
		}
	})

	t.Run("black vs white", func(t *testing.T) {
		d := DistanceRGB(RGBA{0, 0, 0, 255}, RGBA{255, 255, 255, 255})
		expected := math.Sqrt(3 * 255 * 255)
		if math.Abs(d-expected) > 0.001 {
			t.Errorf("got %f, want %f", d, expected)
		}
	})

	t.Run("single channel difference", func(t *testing.T) {
		d := DistanceRGB(RGBA{100, 0, 0, 255}, RGBA{200, 0, 0, 255})
		if math.Abs(d-100) > 0.001 {
			t.Errorf("got %f, want 100", d)
		}
	})
}

func TestWeightedMean(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		got := WeightedMean(nil, nil)
		if got != (RGBA{}) {
			t.Errorf("expected zero RGBA, got %+v", got)
		}
	})

	t.Run("single color", func(t *testing.T) {
		c := RGBA{100, 150, 200, 255}
		got := WeightedMean([]RGBA{c}, nil)
		if got != c {
			t.Errorf("got %+v, want %+v", got, c)
		}
	})

	t.Run("equal weights average", func(t *testing.T) {
		colors := []RGBA{
			{0, 0, 0, 255},
			{100, 100, 100, 255},
		}
		got := WeightedMean(colors, nil)
		if got.R != 50 || got.G != 50 || got.B != 50 {
			t.Errorf("got %+v, want {50,50,50,255}", got)
		}
	})

	t.Run("weighted towards heavier color", func(t *testing.T) {
		colors := []RGBA{
			{0, 0, 0, 255},
			{200, 200, 200, 255},
		}
		weights := []int{1, 3}
		got := WeightedMean(colors, weights)
		// Expected: 200*3/4 = 150
		if got.R != 150 || got.G != 150 || got.B != 150 {
			t.Errorf("got %+v, want {150,150,150,255}", got)
		}
	})

	t.Run("all zero weights returns zero", func(t *testing.T) {
		colors := []RGBA{{100, 100, 100, 255}}
		weights := []int{0}
		got := WeightedMean(colors, weights)
		if got != (RGBA{}) {
			t.Errorf("expected zero RGBA, got %+v", got)
		}
	})
}

func TestIsLight(t *testing.T) {
	tests := []struct {
		name string
		c    RGBA
		want bool
	}{
		{"white is light", RGBA{255, 255, 255, 255}, true},
		{"black is not light", RGBA{0, 0, 0, 255}, false},
		{"bright yellow is light", RGBA{255, 255, 0, 255}, true},
		{"dark blue is not light", RGBA{0, 0, 128, 255}, false},
		{"mid gray", RGBA{128, 128, 128, 255}, false},
		{"light gray", RGBA{200, 200, 200, 255}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.c.IsLight()
			if got != tt.want {
				t.Errorf("RGBA%+v.IsLight() = %v, want %v", tt.c, got, tt.want)
			}
		})
	}
}

func TestMaxRGBDistance(t *testing.T) {
	expected := math.Sqrt(3 * 255 * 255)
	if math.Abs(MaxRGBDistance-expected) > 0.001 {
		t.Errorf("got %f, want %f", MaxRGBDistance, expected)
	}
}
