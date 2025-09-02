// Constants
const BASE_CELL_SIZE = 10; // Base size of each cell in pixels
const DEFAULT_ROWS = 50; // Default map rows
const DEFAULT_COLS = 50; // Default map columns
const ZOOM_FACTOR = 1.2; // How much to zoom in/out each step

// DOM Elements
const mapFileInput = document.getElementById('mapFileInput');
const brushTool = document.getElementById('brushTool');
const brushSizeInput = document.getElementById('brushSize');
const symmetryToggle = document.getElementById('symmetryToggle');
const saveMapBtn = document.getElementById('saveMapBtn');
const drawDiagonalWallsBtn = document.getElementById('drawDiagonalWallsBtn');
const mapRowsInput = document.getElementById('mapRows');
const mapColsInput = document.getElementById('mapCols');
const resizeMapBtn = document.getElementById('resizeMapBtn');
const zoomInBtn = document.getElementById('zoomInBtn');
const zoomOutBtn = document.getElementById('zoomOutBtn');
const mapCanvas = document.getElementById('mapCanvas');
const coordRowSpan = document.getElementById('coordRow');
const coordColSpan = document.getElementById('coordCol');
const ctx = mapCanvas.getContext('2d');

// Map Data
let currentMap = [];
let mapRows = DEFAULT_ROWS;
let mapCols = DEFAULT_COLS;
let currentZoom = 1.0; // Current zoom level
let isPainting = false; // Flag for paintbrush mode
let currentPaintBrush = 'floor'; // Stores the currently selected brush type (floor, wall, bush)
let currentBrushSize = 1; // Current brush size (1x1, 3x3, etc.)
let isPanning = false; // Flag for panning mode
let lastPanX = 0; // Last X position during panning
let lastPanY = 0; // Last Y position during panning
let offsetX = 0; // Current X offset for panning
let offsetY = 0; // Current Y offset for panning
let lineStart = null; // Stores the starting point for line drawing {row, col}
let lineEnd = null; // Stores the ending point for line drawing {row, col}

// Helper to get current cell size based on zoom
function getCellSize() {
    return BASE_CELL_SIZE * currentZoom;
}

// Initialize a default empty map or load existing
function initializeMap(rows, cols) {
    mapRows = rows;
    mapCols = cols;
    currentMap = Array(mapRows).fill(null).map(() => Array(mapCols).fill('floor'));
    updateCanvasDimensions();
    drawMap();

    // Update input fields with current map dimensions
    mapRowsInput.value = mapRows;
    mapColsInput.value = mapCols;
}

// Resize map function
function resizeMap() {
    const newRows = parseInt(mapRowsInput.value);
    const newCols = parseInt(mapColsInput.value);

    if (isNaN(newRows) || isNaN(newCols) || newRows < 10 || newCols < 10) {
        alert('Please enter valid numbers for rows and columns (minimum 10).');
        return;
    }

    const oldMap = currentMap;
    const oldRows = mapRows;
    const oldCols = mapCols;

    mapRows = newRows;
    mapCols = newCols;
    currentMap = Array(mapRows).fill(null).map(() => Array(mapCols).fill('floor'));

    // Copy existing map data to the new map
    for (let r = 0; r < Math.min(oldRows, mapRows); r++) {
        for (let c = 0; c < Math.min(oldCols, mapCols); c++) {
            currentMap[r][c] = oldMap[r][c];
        }
    }

    updateCanvasDimensions();
    drawMap();
}

// Function to draw diagonal walls
function drawDiagonalWalls() {
    const enableSymmetry = symmetryToggle.checked;
    const wallType = 'wall';

    // Diagonal from bottom-left to top-right
    

    // Line 2: offset -10
    drawLine(10, mapRows - 1, mapCols - 1, 10, wallType, enableSymmetry);

    // Line 3: offset +10
    drawLine(0, mapRows - 11, mapCols - 11, 0, wallType, enableSymmetry);

    // Diagonal from top-left to bottom-right
    

    // Line 5: offset -10
    drawLine(10, 0, mapCols - 1, mapRows - 11, wallType, enableSymmetry);

    // Line 6: offset +10
    drawLine(0, 10, mapCols - 11, mapRows - 1, wallType, enableSymmetry);

    drawMap(); // Redraw the map after drawing walls
}

// Update canvas dimensions based on current map size and zoom
function updateCanvasDimensions() {
    const cellSize = getCellSize();
    mapCanvas.width = mapCols * cellSize;
    mapCanvas.height = mapRows * cellSize;
}

// Draw the map on the canvas
function drawMap() {
    ctx.clearRect(0, 0, mapCanvas.width, mapCanvas.height);
    const cellSize = getCellSize();

    ctx.save(); // Save the current canvas state
    ctx.translate(offsetX, offsetY); // Apply panning offset

    for (let r = 0; r < mapRows; r++) {
        for (let c = 0; c < mapCols; c++) {
            drawCell(r, c, currentMap[r][c], cellSize);
        }
    }

    // Draw diagonal quadrant delimiters
    ctx.strokeStyle = '#FFD700'; // Gold color for delimiters
    ctx.lineWidth = 2; // Thicker lines

    // Diagonal line from bottom-left to top-right
    ctx.beginPath();
    ctx.moveTo(0, mapCanvas.height);
    ctx.lineTo(mapCanvas.width, 0);
    ctx.stroke();

    // Diagonal line from top-left to bottom-right
    ctx.beginPath();
    ctx.moveTo(0, 0);
    ctx.lineTo(mapCanvas.width, mapCanvas.height);
    ctx.stroke();

    ctx.lineWidth = 1; // Reset line width for other drawings
    ctx.strokeStyle = '#3A3F4B'; // Reset stroke style

    // Draw line preview if line tool is active
    if (brushTool.value.toLowerCase() === 'line' && lineStart && lineEnd) {
        ctx.strokeStyle = '#61afef'; // Blue for line preview
        ctx.lineWidth = 2;
        ctx.beginPath();
        ctx.moveTo(lineStart.col * cellSize + cellSize / 2, lineStart.row * cellSize + cellSize / 2);
        ctx.lineTo(lineEnd.col * cellSize + cellSize / 2, lineEnd.row * cellSize + cellSize / 2);
        ctx.stroke();
        ctx.lineWidth = 1; // Reset
    }

    ctx.restore(); // Restore the canvas state to remove the translation
}

// Draw a single cell
function drawCell(row, col, type, cellSize) {
    ctx.beginPath();
    ctx.rect(col * cellSize, row * cellSize, cellSize, cellSize);
    switch (type) {
        case 'wall':
            ctx.fillStyle = '#5C6370'; // Dark Gray
            break;
        case 'bush':
            ctx.fillStyle = '#98C379'; // Green
            break;
        case 'floor':
        default:
            ctx.fillStyle = '#21252B'; // Very Dark Blue/Gray
            break;
    }
    ctx.fill();
    ctx.strokeStyle = '#3A3F4B'; // Slightly lighter border
    ctx.stroke();
}

// Zoom functions
function zoomIn() {
    currentZoom *= ZOOM_FACTOR;
    updateCanvasDimensions();
    drawMap();
}

function zoomOut() {
    currentZoom /= ZOOM_FACTOR;
    // Prevent zooming out too much
    if (currentZoom < 0.1) currentZoom = 0.1;
    updateCanvasDimensions();
    drawMap();
}

function applyBrush(row, col) {
    const selectedTool = brushTool.value.toLowerCase(); // Get the selected tool (brush or fill)
    const enableSymmetry = symmetryToggle.checked;

    if (selectedTool === 'fill') {
        // If the fill tool is selected, perform flood fill with the current paintbrush
        floodFill(row, col, currentMap[row][col], currentPaintBrush, enableSymmetry);
    } else {
        // Calculate half-size for centering the brush
        const halfSize = Math.floor(currentBrushSize / 2);

        for (let rOffset = -halfSize; rOffset <= halfSize; rOffset++) {
            for (let cOffset = -halfSize; cOffset <= halfSize; cOffset++) {
                const targetRow = row + rOffset;
                const targetCol = col + cOffset;

                // Apply change to the clicked cell and surrounding area
                if (targetRow >= 0 && targetRow < mapRows && targetCol >= 0 && targetCol < mapCols) {
                    currentMap[targetRow][targetCol] = currentPaintBrush; // Use currentPaintBrush for painting
                }

                // Apply symmetry if enabled (180-degree rotational symmetry)
                if (enableSymmetry) {
                    const symRow = mapRows - 1 - targetRow;
                    const symCol = mapCols - 1 - targetCol;
                    if (symRow >= 0 && symRow < mapRows && symCol >= 0 && symCol < mapCols) {
                        currentMap[symRow][symCol] = currentPaintBrush; // Use currentPaintBrush for painting
                    }
                }
            }
        }
        currentPaintBrush = selectedTool; // Update the current paintbrush after applying
    }
    drawMap(); // Redraw the map after changes
}

// Bresenham's Line Algorithm
function drawLine(x0, y0, x1, y1, type, enableSymmetry) {
    const dx = Math.abs(x1 - x0);
    const dy = Math.abs(y1 - y0);
    const sx = (x0 < x1) ? 1 : -1;
    const sy = (y0 < y1) ? 1 : -1;
    let err = dx - dy;

    while (true) {
        if (y0 >= 0 && y0 < mapRows && x0 >= 0 && x0 < mapCols) {
            currentMap[y0][x0] = type;
        }
        if (enableSymmetry) {
            const symY = mapRows - 1 - y0;
            const symX = mapCols - 1 - x0;
            if (symY >= 0 && symY < mapRows && symX >= 0 && symX < mapCols) {
                currentMap[symY][symX] = type;
            }
        }

        if (x0 === x1 && y0 === y1) break;
        const e2 = 2 * err;
        if (e2 > -dy) {
            err -= dy;
            x0 += sx;
        }
        if (e2 < dx) {
            err += dx;
            y0 += sy;
        }
    }
}

// Bresenham's Line Algorithm
function drawLine(x0, y0, x1, y1, type, enableSymmetry) {
    const dx = Math.abs(x1 - x0);
    const dy = Math.abs(y1 - y0);
    const sx = (x0 < x1) ? 1 : -1;
    const sy = (y0 < y1) ? 1 : -1;
    let err = dx - dy;

    while (true) {
        if (y0 >= 0 && y0 < mapRows && x0 >= 0 && x0 < mapCols) {
            currentMap[y0][x0] = type;
        }
        if (enableSymmetry) {
            const symY = mapRows - 1 - y0;
            const symX = mapCols - 1 - x0;
            if (symY >= 0 && symY < mapRows && symX >= 0 && symX < mapCols) {
                currentMap[symY][symX] = type;
            }
        }

        if (x0 === x1 && y0 === y1) break;
        const e2 = 2 * err;
        if (e2 > -dy) {
            err -= dy;
            x0 += sx;
        }
        if (e2 < dx) {
            err += dx;
            y0 += sy;
        }
    }
}

// Flood fill algorithm
function floodFill(r, c, targetType, replacementType, enableSymmetry) {
    if (targetType === replacementType) {
        return; // Already the desired type
    }
    if (r < 0 || r >= mapRows || c < 0 || c >= mapCols) {
        return; // Out of bounds
    }
    if (currentMap[r][c] !== targetType) {
        return; // Not the target type
    }

    currentMap[r][c] = replacementType; // Change the cell

    // Recursively call for neighbors
    floodFill(r + 1, c, targetType, replacementType, enableSymmetry);
    floodFill(r - 1, c, targetType, replacementType, enableSymmetry);
    floodFill(r, c + 1, targetType, replacementType, enableSymmetry);
    floodFill(r, c - 1, targetType, replacementType, enableSymmetry);

    if (enableSymmetry) {
        const symR = mapRows - 1 - r;
        const symC = mapCols - 1 - c;
        if (symR >= 0 && symR < mapRows && symC >= 0 && symC < mapCols) {
            // Only fill if the symmetric cell is also the target type
            if (currentMap[symR][symC] === targetType) {
                currentMap[symR][symC] = replacementType;
                // Recursively call for symmetric neighbors
                floodFill(symR + 1, symC, targetType, replacementType, enableSymmetry);
                floodFill(symR - 1, symC, targetType, replacementType, enableSymmetry);
                floodFill(symR, symC + 1, targetType, replacementType, enableSymmetry);
                floodFill(symR, symC - 1, targetType, replacementType, enableSymmetry);
            }
        }
    }
}

// Event Listeners
mapFileInput.addEventListener('change', (event) => {
    console.log('File input change event triggered.');
    const file = event.target.files[0];
    if (file) {
        console.log('File selected:', file.name);
        const reader = new FileReader();
        reader.onload = (e) => {
            console.log('FileReader loaded content.');
            try {
                const data = JSON.parse(e.target.result);
                console.log('Parsed JSON data:', data);
                if (data.rows && data.cols && Array.isArray(data.layout)) {
                    mapRows = data.rows;
                    mapCols = data.cols;
                    // Ensure loaded layout uses lowercase strings for consistency
                    currentMap = data.layout.map(row => row.map(cell => cell.toLowerCase()));
                    updateCanvasDimensions(); // Update dimensions after loading new map
                    drawMap();
                    console.log('Map loaded and drawn successfully.');

                    // Update input fields with loaded map dimensions
                    mapRowsInput.value = mapRows;
                    mapColsInput.value = mapCols;
                } else {
                    alert('Invalid map JSON format. Missing rows, cols, or layout array.');
                    console.error('Invalid map JSON format:', data);
                }
            } catch (error) {
                alert('Error parsing JSON: ' + error.message);
                console.error('Error parsing JSON:', error);
            }
        };
        reader.readAsText(file);
    } else {
        console.log('No file selected.');
    }
});

resizeMapBtn.addEventListener('click', resizeMap);

drawDiagonalWallsBtn.addEventListener('click', drawDiagonalWalls);

zoomInBtn.addEventListener('click', zoomIn);
zoomOutBtn.addEventListener('click', zoomOut);

mapCanvas.addEventListener('mousemove', (event) => {
    const rect = mapCanvas.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const y = event.clientY - rect.top;

    const cellSize = getCellSize();
    const col = Math.floor((x - offsetX) / cellSize);
    const row = Math.floor((y - offsetY) / cellSize);

    coordRowSpan.textContent = row;
    coordColSpan.textContent = col;

    if (isPanning) {
        const dx = event.clientX - lastPanX;
        const dy = event.clientY - lastPanY;
        offsetX += dx;
        offsetY += dy;
        lastPanX = event.clientX;
        lastPanY = event.clientY;
        drawMap();
    } else if (isPainting && brushTool.value.toLowerCase() === 'line') {
        lineEnd = { row, col };
        drawMap(); // Redraw to show line preview
    }
});

mapCanvas.addEventListener('mousedown', (event) => {
    if (event.button === 2) { // Right-click
        isPanning = true;
        lastPanX = event.clientX;
        lastPanY = event.clientY;
        mapCanvas.style.cursor = 'grabbing';
    } else { // Left-click
        isPainting = true;
        const rect = mapCanvas.getBoundingClientRect();
        const x = event.clientX - rect.left;
        const y = event.clientY - rect.top;

        const cellSize = getCellSize();
        const col = Math.floor((x - offsetX) / cellSize);
        const row = Math.floor((y - offsetY) / cellSize);

        if (brushTool.value.toLowerCase() === 'line') {
            lineStart = { row, col };
            lineEnd = { row, col }; // Initialize lineEnd for immediate preview
        } else {
            applyBrush(row, col);
        }
    }
});

mapCanvas.addEventListener('mouseup', (event) => {
    isPainting = false;
    isPanning = false;
    mapCanvas.style.cursor = 'grab';

    if (brushTool.value.toLowerCase() === 'line' && lineStart) {
        const rect = mapCanvas.getBoundingClientRect();
        const x = event.clientX - rect.left;
        const y = event.clientY - rect.top;

        const cellSize = getCellSize();
        const col = Math.floor((x - offsetX) / cellSize);
        const row = Math.floor((y - offsetY) / cellSize);

        drawLine(lineStart.col, lineStart.row, col, row, currentPaintBrush, symmetryToggle.checked);
        lineStart = null;
        lineEnd = null;
        drawMap();
    }
});

mapCanvas.addEventListener('contextmenu', (event) => {
    event.preventDefault(); // Prevent context menu on right-click
});

mapCanvas.addEventListener('mouseleave', () => {
    isPainting = false;
});

// Event listener for brush tool change
brushTool.addEventListener('change', () => {
    const selectedTool = brushTool.value.toLowerCase();
    if (selectedTool === 'floor' || selectedTool === 'wall' || selectedTool === 'bush') {
        currentPaintBrush = selectedTool;
    }
});

brushSizeInput.addEventListener('change', () => {
    const newSize = parseInt(brushSizeInput.value);
    if (!isNaN(newSize) && newSize >= 1) {
        currentBrushSize = newSize;
    } else {
        brushSizeInput.value = currentBrushSize; // Revert to old size if invalid
    }
});

saveMapBtn.addEventListener('click', () => {
    const mapData = {
        rows: mapRows,
        cols: mapCols,
        layout: currentMap
    };
    const jsonString = JSON.stringify(mapData, null, 2);
    const blob = new Blob([jsonString], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'map.json';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
});

// Initial map setup
initializeMap(DEFAULT_ROWS, DEFAULT_COLS);

