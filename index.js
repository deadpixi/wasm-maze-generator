/* This code was adapted from:
 * https://github.com/torch2424/wasm-by-example
 * Which was written by Aaron Turner and released under the
 * Creative Commons Attribution 4.0 License
 * (https://creativecommons.org/licenses/by/4.0/)
 *
 * The adaptations were made by Rob King in 2022.
 * This file is released under the same license as above.
 */

// Global variables, populated during setup and manipulated during generation.
var exports = null
const canvasElement = document.querySelector("canvas");
const canvasContext = canvasElement.getContext("2d");
var canvasImageData = canvasContext.createImageData(
    canvasElement.width,
    canvasElement.height
);

// The actual pixels we're going to draw; a view onto an RGBA buffer.
var pixels = undefined;

// The dimensions of the last time we draw the image. We can avoid
// a lot of work if we only change certain size-dependant things on
// resize.
var lastHeight = undefined;
var lastWidth = undefined;

// lastPointer is the last-seen offset into WASM linear memory of
// the frame buffer and lastSize is the last-seen size of the frame buffer
// again, we can avoid a lot of work if we only do certain things if these
// change
var lastPointer = undefined;
var lastSize = undefined;

// The ImageData we use to populate the canvas.
var imageData = undefined;

// Populate the initial font.
canvasContext.font = "10px serif";

// A function to instantiation a WASM module, working around various
// cross-browser problems.
const wasmBrowserInstantiate = async (wasmModuleUrl, importObject) => {
    let response = undefined;

    const fetchAndInstantiateTask = async () => {
        const wasmArrayBuffer = await fetch(wasmModuleUrl).then(response =>
                response.arrayBuffer()
            );
            return WebAssembly.instantiate(wasmArrayBuffer, importObject);
        };
    return await fetchAndInstantiateTask();
};

// Export the maze to an image.
function exportMaze() {
    let data = canvasElement.toDataURL("image/png");
    let image = new Image();
    image.src = data;
    
    let w = window.open('about:blank');
    setTimeout(function(){
        w.document.write(image.outerHTML);
    }, 0);
}

// Called by our WASM code to paint the maze.
function putMaze(newMazeHeight, newMazeWidth, newPointer, newSize, label) {

    // Resize the canvas if needed.
    if (newMazeHeight != lastHeight || newMazeWidth != lastWidth || !canvasImageData) {
        console.log("resizing canvas");
        canvasElement.height = newMazeHeight;
        canvasElement.width = newMazeWidth;
        canvasContext.clearRect(0, 0, canvasElement.width, canvasElement.height);
        canvasImageData = canvasContext.createImageData(
            canvasElement.width,
            canvasElement.height
        );
        lastHeight = newMazeHeight;
        lastWidth = newMazeWidth;
        lastPointer = undefined;
    }
    
    // Rebuild the view onto the framebuffer if needed. FIXME - is testing byteLength portable?
    if (!pixels || lastPointer != newPointer || lastSize != newSize || pixels.byteLength == 0) {
        console.log("rebuilding view ", " lastPointer = ", lastPointer, " newPointer = ", newPointer);
        pixels = new Uint8ClampedArray(
            exports.mem.buffer,
            newPointer,
            newSize
        );
        lastPointer = newPointer;
        lastSize = newSize;
    }
    
    // Place the image onto the canvas.
    canvasImageData.data.set(pixels);
    canvasContext.putImageData(canvasImageData, 0, 0);
    
    // Draw the label.
    if (label) {
        canvasContext.fillText(label, 40, 39);
    }
    
    // Enable the export button.
    document.getElementById("exportButton").disabled = false;
};

// Defined in wasm_exec.js.
const go = new Go();

// Run our Go code.
const runWasm = async () => {
    // Get the importObject from the go instance.
    const importObject = go.importObject;

    // Instantiate our wasm module
    const wasmModule = await wasmBrowserInstantiate("twistylittlepassages", importObject);

    // Allow the wasm_exec go instance, bootstrap and execute our wasm module
    go.run(wasmModule.instance);

    // Get our exports object, with all of our exported Wasm Properties
    exports = wasmModule.instance.exports;

    // Get our canvas element from our index.html
    canvasContext.clearRect(0, 0, canvasElement.width, canvasElement.height);
    
    // Enable the generate button.
	document.getElementById("generateButton").disabled = false;
};
runWasm();