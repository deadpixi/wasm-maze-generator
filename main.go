package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math/rand"
	"strconv"
	"syscall/js"
	"time"
	"unsafe"
)

// Simple timing functions, put
// tr(ace(message))
// at the top of a timed function.
func ace(message string) (string, time.Time) {
	return message, time.Now()
}

func tr(message string, start time.Time) {
	fmt.Printf("%v: %v\n", message, time.Since(start))
}

// The frame buffer storing our image.
var frameBuffer *image.RGBA = nil

// Mazes are simple structures, with a slice-of-slices
// for cells. It's not significanlty faster to use a
// single-dimensional array.
type maze struct {
	start, finish position
	height, width int
	cells         [][]cell
	rng           *rand.Rand
	solution      []position
}

const (
	maxDimension  = 200                    // Maximum number of cells in height and/or width
	border        = 40                     // Border (in pixels) around the maze
	cellWidth     = 12                     // Width/height (in pixels) of a single cell
	halfCellWidth = cellWidth / 2          // Used to find the midpoint of a cell
	minImageWidth = cellWidth*4 + border*2 // Minimum width of generated image (in pixels)
)

// Directions, and displacements to move in a given direction.
type direction int

const (
	north direction = iota
	south
	east
	west
)

var dx = map[direction]int{
	north: 0,
	south: 0,
	east:  1,
	west:  -1,
}

var dy = map[direction]int{
	north: -1,
	south: +1,
	east:  0,
	west:  0,
}

var op = map[direction]direction{
	north: south,
	south: north,
	east:  west,
	west:  east,
}

// A single cell.
type cell struct {
	visited  bool    // True if we've visited this cell on this walk.
	openings [4]bool // Whether a given wall is open.
}

// Build a new maze with the given height and width.
// Randomness is taken from the given RNG.
// oppositeStart means to place start/end at opposing corners.
func newMaze(height, width int, rng *rand.Rand, oppositeStart bool) *maze {
	if width < 2 || height < 2 || rng == nil {
		panic("invalid call to newMaze")
	}

	cells := make([][]cell, height)
	for i := range cells {
		cells[i] = make([]cell, width)
	}

	start := position{rng.Intn(width), 0}
	end := position{rng.Intn(width), height - 1}
	if oppositeStart {
		start = position{0, 0}
		end = position{width - 1, height - 1}
	}
	return &maze{
		start:  start,
		finish: end,
		height: height,
		width:  width,
		cells:  cells,
		rng:    rng,
	}
}

// Position is simply x/y coordinates.
type position struct {
	x, y int
}

// We use a stack of positions when generating and solving
// the maze. This avoids using the call stack. Go has a very
// deep call stack on most targets, but I'm not comfortable
// asking WASM can give us a ~1500-level stack.
type stack []position

func push(s stack, p position) stack {
	return append(s, p)
}

func peek(s stack) position {
	if len(s) == 0 {
		panic("stack underflow")
	}
	return s[len(s)-1]
}

func pop(s stack) stack {
	return s[:len(s)-1]
}

func empty(s stack) bool {
	return len(s) == 0
}

// We precompute all possible permutations of orders to try digging.
// This speeds up maze generation by ~25% from shuffling the directions
// on each iteration through the maze generation loop.
var permutations = [][]direction{
	[]direction{north, south, east, west},
	[]direction{north, south, west, east},
	[]direction{north, east, south, west},
	[]direction{north, east, west, south},
	[]direction{north, west, south, east},
	[]direction{north, west, east, south},
	[]direction{south, north, east, west},
	[]direction{south, north, west, east},
	[]direction{south, east, north, west},
	[]direction{south, east, west, north},
	[]direction{south, west, north, east},
	[]direction{south, west, east, north},
	[]direction{east, north, south, west},
	[]direction{east, north, west, south},
	[]direction{east, south, north, west},
	[]direction{east, south, west, north},
	[]direction{east, west, north, south},
	[]direction{east, west, south, north},
	[]direction{west, north, south, east},
	[]direction{west, north, east, south},
	[]direction{west, south, north, east},
	[]direction{west, south, east, north},
	[]direction{west, east, north, south},
	[]direction{west, east, south, north},
}

func (m *maze) generate() {
	defer tr(ace("generating maze"))

	stack := []position{m.start}
	for !empty(stack) {
		found := false
		p := peek(stack)
		dirs := permutations[m.rng.Intn(len(permutations))]
		for _, dir := range dirs {
			nx, ny := p.x+dx[dir], p.y+dy[dir]
			if nx >= 0 && nx < m.width && ny >= 0 && ny < m.height && !m.cells[ny][nx].visited {
				m.cells[p.y][p.x].openings[dir] = true
				m.cells[ny][nx].openings[op[dir]] = true
				m.cells[ny][nx].visited = true
				stack = push(stack, position{nx, ny})
				found = true
				if nx == m.finish.x && ny == m.finish.y && (len(m.solution) == 0 || len(stack) < len(m.solution)) {
					m.solution = make([]position, len(stack))
					copy(m.solution, stack)
				}
				break
			}
		}

		if !found {
			stack = pop(stack)
		}
	}
}

func (m *maze) draw() *image.RGBA {
	defer tr(ace("drawing maze"))

	width := m.width*cellWidth + border*2
	if width < minImageWidth {
		width = minImageWidth
	}

	bounds := image.Rect(0, 0, width, m.height*cellWidth+border*2)
	if frameBuffer == nil || frameBuffer.Bounds() != bounds {
		frameBuffer = image.NewRGBA(bounds)
	}
	fill(frameBuffer, 0, m.height*cellWidth+border*2, 0, width, image.White)

	for y, row := range m.cells {
		for x, cell := range row {
			m.drawCell(frameBuffer, x, y, cell)
		}
	}

	return frameBuffer
}

var red = image.NewUniform(color.RGBA{255, 0, 0, 255})

func (m *maze) drawPath(img *image.RGBA, path []position) {
	defer tr(ace("drawing solution"))

	prev := path[0]
	for _, pos := range path[1:] {
		if pos.x == prev.x {
			first, last := prev, pos
			if first.y > last.y {
				first, last = last, first
			}
			vLine(img, first.x*cellWidth+border+halfCellWidth, first.y*cellWidth+border+halfCellWidth, last.y*cellWidth+border+halfCellWidth, red)
		}
		if pos.y == prev.y {
			first, last := prev, pos
			if first.x > last.x {
				first, last = last, first
			}
			hLine(img, first.x*cellWidth+border+halfCellWidth, first.y*cellWidth+border+halfCellWidth, last.x*cellWidth+border+halfCellWidth, red)
		}
		prev = pos
	}
}

func fill(img *image.RGBA, y0, y1, x0, x1 int, color color.Color) {
	defer tr(ace("clearing image"))
	draw.Draw(img, img.Bounds(), &image.Uniform{color}, image.Point{0, 0}, draw.Src)
}

func hLine(img *image.RGBA, x1, y, x2 int, col image.Image) {
	draw.Draw(img, image.Rect(x1, y, x2+1, y+1), col, image.Point{0, 0}, draw.Over)
}

func vLine(img *image.RGBA, x, y1, y2 int, col image.Image) {
	draw.Draw(img, image.Rect(x, y1, x+1, y2+1), col, image.Point{0, 0}, draw.Over)
}

func (m *maze) drawCell(img *image.RGBA, x, y int, c cell) {
	if !c.openings[north] && !(x == m.start.x && y == m.start.y) {
		hLine(img, x*cellWidth+border, y*cellWidth+border, x*cellWidth+border+cellWidth, image.Black)
	}

	if !c.openings[south] && !(x == m.finish.x && y == m.finish.y) {
		hLine(img, x*cellWidth+border, y*cellWidth+border+cellWidth, x*cellWidth+border+cellWidth, image.Black)
	}

	if !c.openings[west] {
		vLine(img, x*cellWidth+border, y*cellWidth+border, y*cellWidth+border+cellWidth, image.Black)
	}

	if !c.openings[east] {
		vLine(img, x*cellWidth+border+cellWidth, y*cellWidth+border, y*cellWidth+border+cellWidth, image.Black)
	}
}

// We import a function called putMaze, which is written in JavaScript.
// TinyGo makes this slightly easier, but this really isn't too bad:
var putMaze js.Value = js.Global().Get("putMaze")

func main() {
	generateCb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		generateCallback()
		args[0].Call("preventDefault")
		return nil
	})
	defer generateCb.Release()

	js.Global().Get("document").
		Call("getElementById", "generateButton").
		Call("addEventListener", "click", generateCb)

	// spin a while...spin FOREVER
	select {}
}

// The actual function called to generate mazes.
func generateCallback() {
	defer tr(ace("total time"))

	height, width, solution, label, oppositeStart, seed, err := getArguments()
	if err != nil || height < 2 || width < 2 || height > maxDimension || width > maxDimension {
		fmt.Printf("Error: %s\n", err)
		return
	}

	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	m := newMaze(int(height), int(width), rand.New(rand.NewSource(seed)), oppositeStart)
	m.generate()

	img := m.draw()
	if solution {
		m.drawPath(img, m.solution)
	}

	labelText := ""
	if label {
		labelText = fmt.Sprintf("%dx%d %x", m.height, m.width, seed)
	}
	export(labelText)
}

// Grab our parameters from JS land.
func getArguments() (height, width int64, solution, label, oppositeStart bool, seed int64, err error) {
	document := js.Global().Get("document")

	height, err = strconv.ParseInt(document.Call("getElementById", "mazeHeight").Get("value").String(), 10, 16)
	width, err = strconv.ParseInt(document.Call("getElementById", "mazeWidth").Get("value").String(), 10, 16)
	seed, err = strconv.ParseInt(document.Call("getElementById", "randomSeed").Get("value").String(), 10, 64)
	solution = document.Call("getElementById", "showSolution").Get("checked").Truthy()
	label = document.Call("getElementById", "labelMaze").Get("checked").Truthy()
	oppositeStart = document.Call("getElementById", "oppositeStart").Get("checked").Truthy()

	return
}

// Export the frame buffer. We invoke putMaze here, which actually
// puts the pixel data into the canvas.
//
// Note that image.RGBA.Pix just happens to be in the correct format
// for Canvas ImageData. This means that we can simply pass a pointer
// into WASM linear memory and the JS side can pick it up with no
// copying. We do a safe cast from the slice to the underlying array,
// and then an unsafe cast to a uintptr, which is the offset of the
// frame buffer in linear memory.
func export(label string) {
	defer tr(ace("exporting frame buffer"))
	putMaze.Invoke(
		js.ValueOf(frameBuffer.Bounds().Dy()),
		js.ValueOf(frameBuffer.Bounds().Dx()),
		js.ValueOf(uintptr(unsafe.Pointer((*[1]uint8)(frameBuffer.Pix)))),
		js.ValueOf(len(frameBuffer.Pix)),
		js.ValueOf(label),
	)
}
