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

var graphicsBuffer *image.RGBA = nil

type maze struct {
	start, finish position
	height, width int
	cells         [][]cell
	rng           *rand.Rand
	solution      []position
}

const (
	maxDimension  = 200
	border        = 40
	cellWidth     = 12
	halfCellWidth = cellWidth / 2
	minImageWidth = cellWidth*4 + border*2
)

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

type cell struct {
	visited  bool
	openings [4]bool
}

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

type position struct {
	x, y int
}

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

func ace(message string) (string, time.Time) {
	return message, time.Now()
}

func tr(message string, start time.Time) {
	fmt.Printf("%v: %v\n", message, time.Since(start))
}

func (m *maze) generate() {
	defer tr(ace("generating maze"))

	stack := push(nil, m.start)
	for !empty(stack) {
		found := false
		p := peek(stack)
		dirs := []direction{north, south, east, west}
		m.rng.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
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

	// FIXME - don't reinitialize this if the size didn't change
	bounds := image.Rect(0, 0, width, m.height*cellWidth+border*2)
	if graphicsBuffer == nil || graphicsBuffer.Bounds() != bounds {
		graphicsBuffer = image.NewRGBA(bounds)
	}
	fill(graphicsBuffer, 0, m.height*cellWidth+border*2, 0, width, image.White)

	for y, row := range m.cells {
		for x, cell := range row {
			m.drawCell(graphicsBuffer, x, y, cell)
		}
	}

	return graphicsBuffer
}

var red = image.NewUniform(color.RGBA{255, 0, 0, 255})

func (m *maze) drawPath(img *image.RGBA, path []position) {
	defer tr(ace("drawing solution"))

	prev := path[0]
	for _, pos := range path {
		if pos.x == prev.x && pos.y == prev.y {
			continue // FIXME - draw a circle
		}
		if pos.x == prev.x {
			vLine(img, pos.x*cellWidth+border+halfCellWidth, pos.y*cellWidth+border+halfCellWidth+1, prev.y*cellWidth+border+halfCellWidth+1, red)
		}
		if pos.y == prev.y {
			hLine(img, pos.x*cellWidth+border+halfCellWidth+1, pos.y*cellWidth+border+halfCellWidth, prev.x*cellWidth+border+halfCellWidth+1, red)
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

var putMaze js.Value = js.Undefined()

func importFunctions() {
	putMaze = js.Global().Get("putMaze")
}

func main() {
	importFunctions()

	generateCb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		generateCallback()
		args[0].Call("preventDefault")
		return nil
	})
	js.Global().Get("document").
		Call("getElementById", "generateButton").
		Call("addEventListener", "click", generateCb)

	wait := make(chan struct{})
	<-wait

	generateCb.Release()
}

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

func export(label string) {
	defer tr(ace("exporting image"))
	putMaze.Invoke(
		js.ValueOf(graphicsBuffer.Bounds().Dy()),
		js.ValueOf(graphicsBuffer.Bounds().Dx()),
		js.ValueOf(uintptr(unsafe.Pointer((*[1]uint8)(graphicsBuffer.Pix)))),
		js.ValueOf(len(graphicsBuffer.Pix)),
		js.ValueOf(label),
	)
}
