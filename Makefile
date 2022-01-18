all: wasm_exec.js twistylittlepassages.gz

wasm_exec.js: twistylittlepassages
	cp "$$(go env GOROOT)/misc/wasm/wasm_exec.js" $@

twistylittlepassages: main.go
	GOOS=js GOARCH=wasm go build .

twistylittlepassages.gz: twistylittlepassages
	gzip -9 $<

clean:
	go clean
	rm -f wasm_exec.js twistylittlepassages twistylittlepassages.gz
