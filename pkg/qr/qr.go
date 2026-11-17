// Package qr makes QR codes (byte mode, medium error correction, versions
// 1 to 10, up to 213 bytes) as SVG pictures, for setting up two-step sign
// in with an authenticator app. It follows ISO/IEC 18004.
package qr

import (
	"errors"
	"fmt"
	"strings"
)

// blocks per version for error correction level M: ec codewords per
// block, then (count, data codewords) for each group
var versionsM = [11]struct {
	ec     int
	groups [][2]int
}{
	{},
	{10, [][2]int{{1, 16}}},
	{16, [][2]int{{1, 28}}},
	{26, [][2]int{{1, 44}}},
	{18, [][2]int{{2, 32}}},
	{24, [][2]int{{2, 43}}},
	{16, [][2]int{{4, 27}}},
	{18, [][2]int{{4, 31}}},
	{22, [][2]int{{2, 38}, {2, 39}}},
	{22, [][2]int{{3, 36}, {2, 37}}},
	{26, [][2]int{{4, 43}, {1, 44}}},
}

var alignment = [11][]int{
	{}, {}, {6, 18}, {6, 22}, {6, 26}, {6, 30}, {6, 34},
	{6, 22, 38}, {6, 24, 42}, {6, 26, 46}, {6, 28, 50},
}

// Code is a square of modules; true is dark.
type Code struct {
	Size    int
	modules [][]bool
	isFunc  [][]bool
	version int
}

// Dark reports whether a module is dark.
func (c *Code) Dark(x, y int) bool { return c.modules[y][x] }

func dataCodewords(v int) int {
	n := 0
	for _, g := range versionsM[v].groups {
		n += g[0] * g[1]
	}
	return n
}

// Encode makes a QR code for the text.
func Encode(text string) (*Code, error) {
	data := []byte(text)
	version := 0
	for v := 1; v <= 10; v++ {
		countBits := 8
		if v >= 10 {
			countBits = 16
		}
		if 4+countBits+8*len(data) <= dataCodewords(v)*8 {
			version = v
			break
		}
	}
	if version == 0 {
		return nil, errors.New("qr: text too long")
	}

	// the bit stream
	var bits []bool
	put := func(val, n int) {
		for i := n - 1; i >= 0; i-- {
			bits = append(bits, (val>>i)&1 == 1)
		}
	}
	put(0b0100, 4)
	if version >= 10 {
		put(len(data), 16)
	} else {
		put(len(data), 8)
	}
	for _, b := range data {
		put(int(b), 8)
	}
	capacity := dataCodewords(version) * 8
	put(0, min(4, capacity-len(bits)))
	for len(bits)%8 != 0 {
		bits = append(bits, false)
	}
	for pad := 0; len(bits) < capacity; pad ^= 1 {
		if pad == 0 {
			put(0xEC, 8)
		} else {
			put(0x11, 8)
		}
	}
	codewords := make([]byte, len(bits)/8)
	for i, b := range bits {
		if b {
			codewords[i/8] |= 1 << (7 - i%8)
		}
	}

	// split into blocks, add error correction, interleave
	ecLen := versionsM[version].ec
	var dataBlocks, ecBlocks [][]byte
	off := 0
	for _, g := range versionsM[version].groups {
		for i := 0; i < g[0]; i++ {
			blk := codewords[off : off+g[1]]
			off += g[1]
			dataBlocks = append(dataBlocks, blk)
			ecBlocks = append(ecBlocks, reedSolomon(blk, ecLen))
		}
	}
	var final []byte
	maxData := 0
	for _, b := range dataBlocks {
		maxData = max(maxData, len(b))
	}
	for i := 0; i < maxData; i++ {
		for _, b := range dataBlocks {
			if i < len(b) {
				final = append(final, b[i])
			}
		}
	}
	for i := 0; i < ecLen; i++ {
		for _, b := range ecBlocks {
			final = append(final, b[i])
		}
	}

	c := newCode(version)
	c.drawFunctionPatterns()
	c.placeData(final)

	// the mask with the lowest penalty
	best, bestPenalty := 0, -1
	for m := 0; m < 8; m++ {
		c.applyMask(m)
		c.drawFormat(m)
		if p := c.penalty(); bestPenalty < 0 || p < bestPenalty {
			best, bestPenalty = m, p
		}
		c.applyMask(m) // undo
	}
	c.applyMask(best)
	c.drawFormat(best)
	return c, nil
}

func newCode(version int) *Code {
	size := version*4 + 17
	c := &Code{Size: size, version: version}
	c.modules = make([][]bool, size)
	c.isFunc = make([][]bool, size)
	for i := range c.modules {
		c.modules[i] = make([]bool, size)
		c.isFunc[i] = make([]bool, size)
	}
	return c
}

func (c *Code) set(x, y int, dark bool) {
	c.modules[y][x] = dark
	c.isFunc[y][x] = true
}

func (c *Code) drawFunctionPatterns() {
	// timing
	for i := 0; i < c.Size; i++ {
		c.set(6, i, i%2 == 0)
		c.set(i, 6, i%2 == 0)
	}
	// finders with their separators
	for _, p := range [][2]int{{3, 3}, {c.Size - 4, 3}, {3, c.Size - 4}} {
		for dy := -4; dy <= 4; dy++ {
			for dx := -4; dx <= 4; dx++ {
				x, y := p[0]+dx, p[1]+dy
				if x < 0 || y < 0 || x >= c.Size || y >= c.Size {
					continue
				}
				d := max(abs(dx), abs(dy))
				c.set(x, y, d != 2 && d != 4)
			}
		}
	}
	// alignment
	pos := alignment[c.version]
	for i, ax := range pos {
		for j, ay := range pos {
			if (i == 0 && j == 0) || (i == 0 && j == len(pos)-1) || (i == len(pos)-1 && j == 0) {
				continue
			}
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					c.set(ax+dx, ay+dy, max(abs(dx), abs(dy)) != 1)
				}
			}
		}
	}
	// reserve the format areas (drawn later) and the dark module
	c.drawFormat(0)
	// version information
	if c.version >= 7 {
		rem := c.version
		for i := 0; i < 12; i++ {
			rem = (rem << 1) ^ ((rem >> 11) * 0x1F25)
		}
		bits := c.version<<12 | rem
		for i := 0; i < 18; i++ {
			dark := (bits>>i)&1 == 1
			a, b := c.Size-11+i%3, i/3
			c.set(a, b, dark)
			c.set(b, a, dark)
		}
	}
}

func (c *Code) drawFormat(mask int) {
	// level M is 00
	data := 0<<3 | mask
	rem := data
	for i := 0; i < 10; i++ {
		rem = (rem << 1) ^ ((rem >> 9) * 0x537)
	}
	bits := (data<<10 | rem) ^ 0x5412
	bit := func(i int) bool { return (bits>>i)&1 == 1 }

	for i := 0; i <= 5; i++ {
		c.set(8, i, bit(i))
	}
	c.set(8, 7, bit(6))
	c.set(8, 8, bit(7))
	c.set(7, 8, bit(8))
	for i := 9; i < 15; i++ {
		c.set(14-i, 8, bit(i))
	}
	for i := 0; i < 8; i++ {
		c.set(c.Size-1-i, 8, bit(i))
	}
	for i := 8; i < 15; i++ {
		c.set(8, c.Size-15+i, bit(i))
	}
	c.set(8, c.Size-8, true)
}

func (c *Code) placeData(data []byte) {
	i := 0
	total := len(data) * 8
	for right := c.Size - 1; right >= 1; right -= 2 {
		if right == 6 {
			right = 5
		}
		for vert := 0; vert < c.Size; vert++ {
			for j := 0; j < 2; j++ {
				x := right - j
				upward := ((right + 1) & 2) == 0
				y := vert
				if upward {
					y = c.Size - 1 - vert
				}
				if c.isFunc[y][x] {
					continue
				}
				if i < total {
					c.modules[y][x] = (data[i/8]>>(7-i%8))&1 == 1
					i++
				}
			}
		}
	}
}

func (c *Code) applyMask(m int) {
	for y := 0; y < c.Size; y++ {
		for x := 0; x < c.Size; x++ {
			if c.isFunc[y][x] {
				continue
			}
			var flip bool
			switch m {
			case 0:
				flip = (x+y)%2 == 0
			case 1:
				flip = y%2 == 0
			case 2:
				flip = x%3 == 0
			case 3:
				flip = (x+y)%3 == 0
			case 4:
				flip = (x/3+y/2)%2 == 0
			case 5:
				flip = x*y%2+x*y%3 == 0
			case 6:
				flip = (x*y%2+x*y%3)%2 == 0
			case 7:
				flip = ((x+y)%2+x*y%3)%2 == 0
			}
			if flip {
				c.modules[y][x] = !c.modules[y][x]
			}
		}
	}
}

func (c *Code) penalty() int {
	n := c.Size
	p := 0
	get := func(x, y int) bool { return c.modules[y][x] }
	// runs of five or more in rows and columns
	for pass := 0; pass < 2; pass++ {
		for a := 0; a < n; a++ {
			run := 1
			for b := 1; b < n; b++ {
				var cur, prev bool
				if pass == 0 {
					cur, prev = get(b, a), get(b-1, a)
				} else {
					cur, prev = get(a, b), get(a, b-1)
				}
				if cur == prev {
					run++
					if run == 5 {
						p += 3
					} else if run > 5 {
						p++
					}
				} else {
					run = 1
				}
			}
		}
	}
	// 2x2 blocks
	for y := 0; y < n-1; y++ {
		for x := 0; x < n-1; x++ {
			v := get(x, y)
			if v == get(x+1, y) && v == get(x, y+1) && v == get(x+1, y+1) {
				p += 3
			}
		}
	}
	// finder-like patterns
	pat1 := []bool{true, false, true, true, true, false, true, false, false, false, false}
	pat2 := []bool{false, false, false, false, true, false, true, true, true, false, true}
	for a := 0; a < n; a++ {
		for b := 0; b+11 <= n; b++ {
			for _, pat := range [][]bool{pat1, pat2} {
				row, col := true, true
				for k := 0; k < 11; k++ {
					if get(b+k, a) != pat[k] {
						row = false
					}
					if get(a, b+k) != pat[k] {
						col = false
					}
				}
				if row {
					p += 40
				}
				if col {
					p += 40
				}
			}
		}
	}
	// balance of dark and light
	dark := 0
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if get(x, y) {
				dark++
			}
		}
	}
	k := abs(dark*20-n*n*10) / (n * n)
	p += k * 10
	return p
}

// ---------------------------------------------------------------- Reed-Solomon

var gfExp, gfLog [512]int

func init() {
	x := 1
	for i := 0; i < 255; i++ {
		gfExp[i] = x
		gfLog[x] = i
		x <<= 1
		if x&0x100 != 0 {
			x ^= 0x11D
		}
	}
	for i := 255; i < 512; i++ {
		gfExp[i] = gfExp[i-255]
	}
}

func gfMul(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	return gfExp[gfLog[a]+gfLog[b]]
}

func reedSolomon(data []byte, ecLen int) []byte {
	// generator polynomial (x - a^0)(x - a^1)...(x - a^(ecLen-1))
	gen := []int{1}
	for i := 0; i < ecLen; i++ {
		next := make([]int, len(gen)+1)
		for j, g := range gen {
			next[j] ^= g
			next[j+1] ^= gfMul(g, gfExp[i])
		}
		gen = next
	}
	rem := make([]int, ecLen)
	for _, d := range data {
		factor := int(d) ^ rem[0]
		copy(rem, rem[1:])
		rem[ecLen-1] = 0
		for j := 0; j < ecLen; j++ {
			rem[j] ^= gfMul(gen[j+1], factor)
		}
	}
	out := make([]byte, ecLen)
	for i, r := range rem {
		out[i] = byte(r)
	}
	return out
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

// SVG draws the code with a quiet zone of four modules.
func (c *Code) SVG() string {
	n := c.Size + 8
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges"><rect width="%d" height="%d" fill="#fff"/><path fill="#000" d="`, n, n, n, n)
	for y := 0; y < c.Size; y++ {
		for x := 0; x < c.Size; x++ {
			if c.modules[y][x] {
				fmt.Fprintf(&b, "M%d %dh1v1h-1z", x+4, y+4)
			}
		}
	}
	b.WriteString(`"/></svg>`)
	return b.String()
}
