package branding

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"strconv"
)

// PNG meraster ikon sederhana (tanpa dependensi SVG).
func PNG(icon, bg, fg string, size int) ([]byte, error) {
	icon, bg, fg = Normalize(icon, bg, fg)
	if size < 48 {
		size = 48
	}
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	bgc := colorFromHex(bg)
	fgc := colorFromHex(fg)
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i] = bgc.R
		img.Pix[i+1] = bgc.G
		img.Pix[i+2] = bgc.B
		img.Pix[i+3] = 255
	}

	pad := float64(size) * 0.22
	box := float64(size) - 2*pad
	g := glyph{img: img, ox: pad, oy: pad, box: box, fg: fgc, bg: bgc}
	g.draw(icon)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func colorFromHex(h string) color.RGBA {
	if len(h) != 7 || h[0] != '#' {
		return color.RGBA{A: 255}
	}
	r, _ := strconv.ParseUint(h[1:3], 16, 8)
	g, _ := strconv.ParseUint(h[3:5], 16, 8)
	b, _ := strconv.ParseUint(h[5:7], 16, 8)
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}

type glyph struct {
	img      *image.RGBA
	ox, oy   float64
	box      float64
	fg, bg   color.RGBA
}

func (g glyph) X(px float64) int { return int(g.ox + px/24*g.box + 0.5) }
func (g glyph) Y(py float64) int { return int(g.oy + py/24*g.box + 0.5) }
func (g glyph) S(v float64) int  { return int(v/24*g.box + 0.5) }

func (g glyph) disk(cx, cy, r float64, c color.RGBA) {
	fillCircle(g.img, g.X(cx), g.Y(cy), g.S(r), c)
}

func (g glyph) rect(x0, y0, x1, y1 float64, c color.RGBA) {
	a, b := g.X(x0), g.Y(y0)
	d, e := g.X(x1), g.Y(y1)
	if a > d {
		a, d = d, a
	}
	if b > e {
		b, e = e, b
	}
	for y := b; y < e; y++ {
		for x := a; x < d; x++ {
			g.img.SetRGBA(x, y, c)
		}
	}
}

func (g glyph) line(x0, y0, x1, y1, w float64) {
	ax, ay := float64(g.X(x0)), float64(g.Y(y0))
	bx, by := float64(g.X(x1)), float64(g.Y(y1))
	thick := float64(g.S(w))
	if thick < 1 {
		thick = 1
	}
	dx, dy := bx-ax, by-ay
	length := math.Hypot(dx, dy)
	if length < 1 {
		fillCircle(g.img, int(ax), int(ay), int(thick/2), g.fg)
		return
	}
	steps := int(length*2) + 1
	r := int(thick/2 + 0.5)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		fillCircle(g.img, int(ax+dx*t+0.5), int(ay+dy*t+0.5), r, g.fg)
	}
}

func (g glyph) ring(cx, cy, r, w float64) {
	strokeCircle(g.img, g.X(cx), g.Y(cy), g.S(r), g.S(w), g.fg)
}

func (g glyph) poly(pts [][2]float64) {
	fillPoly(g.img, pts, g.ox, g.oy, g.box, g.fg)
}

func (g glyph) draw(icon string) {
	switch icon {
	case "apple":
		g.disk(12, 13.2, 7.4, g.fg)
		g.disk(16.8, 11.2, 3.4, g.bg) // gigitan
		g.disk(14.2, 5.2, 2.3, g.fg)  // daun
	case "android":
		g.disk(12, 9, 5.2, g.fg)
		g.rect(7, 10, 17, 17.5, g.fg)
		g.rect(4.8, 11, 7, 15.5, g.fg)
		g.rect(17, 11, 19.2, 15.5, g.fg)
		g.line(8.2, 5.2, 9.6, 7.4, 1.5)
		g.line(15.8, 5.2, 14.4, 7.4, 1.5)
		g.disk(10, 9, 1.1, g.bg)
		g.disk(14, 9, 1.1, g.bg)
	case "linux":
		g.disk(12, 9, 5.6, g.fg)
		g.disk(12, 16.2, 6.6, g.fg)
		g.disk(10, 8.4, 1.2, g.bg)
		g.disk(14, 8.4, 1.2, g.bg)
		g.rect(10.5, 11.2, 13.5, 12.2, g.bg)
	case "windows":
		g.rect(3.8, 4.8, 11, 11.8, g.fg)
		g.rect(13, 3.8, 20.2, 11.8, g.fg)
		g.rect(3.8, 13, 11, 20.2, g.fg)
		g.rect(13, 13, 20.2, 20.5, g.fg)
	case "terminal":
		g.rect(4, 5, 20, 19, g.fg)
		g.rect(5.2, 7.2, 18.8, 17.8, g.bg)
		g.line(7, 10, 10, 13, 1.8)
		g.line(10, 13, 7, 16, 1.8)
		g.rect(12, 15, 17.2, 16.6, g.fg)
	case "spark":
		g.poly([][2]float64{{12, 2.5}, {14.2, 9.5}, {21.5, 12}, {14.2, 14.5}, {12, 21.5}, {9.8, 14.5}, {2.5, 12}, {9.8, 9.5}})
		g.poly([][2]float64{{18.5, 4}, {19.3, 6.5}, {21.8, 7.3}, {19.3, 8.1}, {18.5, 10.6}, {17.7, 8.1}, {15.2, 7.3}, {17.7, 6.5}})
	case "chat":
		g.rect(4.5, 4.5, 19.5, 15, g.fg)
		g.poly([][2]float64{{8.5, 15}, {7.2, 20.5}, {13.2, 15}})
	case "hex":
		g.poly([][2]float64{{12, 2.8}, {20.2, 7.4}, {20.2, 16.6}, {12, 21.2}, {3.8, 16.6}, {3.8, 7.4}})
	case "bot":
		g.rect(6.5, 7, 17.5, 18.5, g.fg)
		g.rect(11, 3.8, 13, 7, g.fg)
		g.disk(10, 11.2, 1.4, g.bg)
		g.disk(14, 11.2, 1.4, g.bg)
		g.rect(9, 14.2, 15, 15.8, g.bg)
	case "globe":
		g.ring(12, 12, 8.4, 1.6)
		g.line(12, 3.6, 12, 20.4, 1.3)
		g.line(3.6, 12, 20.4, 12, 1.3)
		g.ring(12, 12, 4.2, 1.2)
	default: // ayumi
		g.line(12, 4.5, 5.5, 19.5, 2.4)
		g.line(12, 4.5, 18.5, 19.5, 2.4)
		g.line(8.2, 13.8, 15.8, 13.8, 2.2)
	}
}

func fillCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	if r < 1 {
		img.SetRGBA(cx, cy, c)
		return
	}
	r2 := r * r
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r2 {
				img.SetRGBA(cx+x, cy+y, c)
			}
		}
	}
}

func strokeCircle(img *image.RGBA, cx, cy, r, w int, c color.RGBA) {
	if w < 1 {
		w = 1
	}
	outer, inner := r+w/2, r-w/2
	if inner < 0 {
		inner = 0
	}
	o2, i2 := outer*outer, inner*inner
	for y := -outer; y <= outer; y++ {
		for x := -outer; x <= outer; x++ {
			d := x*x + y*y
			if d <= o2 && d >= i2 {
				img.SetRGBA(cx+x, cy+y, c)
			}
		}
	}
}

func fillPoly(img *image.RGBA, pts [][2]float64, ox, oy, box float64, c color.RGBA) {
	if len(pts) < 3 {
		return
	}
	p := make([][2]int, len(pts))
	minY, maxY := math.MaxInt32, 0
	for i, pt := range pts {
		x := int(ox + pt[0]/24*box + 0.5)
		y := int(oy + pt[1]/24*box + 0.5)
		p[i] = [2]int{x, y}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}
	for y := minY; y <= maxY; y++ {
		var nodes []int
		j := len(p) - 1
		for i := 0; i < len(p); i++ {
			yi, yj := p[i][1], p[j][1]
			if (yi < y && yj >= y) || (yj < y && yi >= y) {
				xi, xj := p[i][0], p[j][0]
				nodes = append(nodes, xi+(y-yi)*(xj-xi)/(yj-yi))
			}
			j = i
		}
		for i := 0; i < len(nodes); i++ {
			for k := i + 1; k < len(nodes); k++ {
				if nodes[k] < nodes[i] {
					nodes[i], nodes[k] = nodes[k], nodes[i]
				}
			}
		}
		for i := 0; i+1 < len(nodes); i += 2 {
			for x := nodes[i]; x <= nodes[i+1]; x++ {
				img.SetRGBA(x, y, c)
			}
		}
	}
}
