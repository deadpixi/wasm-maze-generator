# wasm-maze-generator
A simple WASM maze generator in Go.

There's a [live demo](http://frigidriver.com/mazes).

Build with

	GOOS=js GOARCH=wasm go build
	gzip twistylittlepassages

Upload `*.{gz,html,js}` somewhere.