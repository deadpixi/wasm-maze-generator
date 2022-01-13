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
var pixels = undefined;
var lastHeight = undefined;
var lastWidth = undefined;
var lastPointer = undefined;
var lastSize = undefined;
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
    if (newMazeHeight != lastHeight || newMazeWidth != lastWidth) {
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
    
    // Rebuild the view onto the framebuffer if needed.
    if (!pixels || lastPointer != newPointer || lastSize != newSize) {
        console.log("rebuilding view ", " lastPointer = ", lastPointer, " newPointer = ", newPointer);
        pixels = new Uint8ClampedArray(
            exports.mem.buffer,
            newPointer,
            newSize
        );
        canvasImageData.data.set(pixels);
        lastPointer = newPointer;
        lastSize = newSize;
    }
    
    // Place the image onto the canvas.
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