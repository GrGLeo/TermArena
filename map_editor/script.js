// Constants
const BASE_CELL_SIZE = 10; // Base size of each cell in pixels
const DEFAULT_ROWS = 50; // Default map rows
const DEFAULT_COLS = 50; // Default map columns
const ZOOM_FACTOR = 1.2; // How much to zoom in/out each step

// DOM Elements
const mapFileInput = document.getElementById('mapFileInput');
const brushTool = document.getElementById('brushTool');
const symmetryToggle = document.getElementById('symmetryToggle');
const saveMapBtn = document.getElementById('saveMapBtn');
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
    for (let r = 0; r < mapRows; r++) {
        for (let c = 0; c < mapCols; c++) {
            drawCell(r, c, currentMap[r][c], cellSize);
        }
    }
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
                    currentMap = data.layout;
                    updateCanvasDimensions(); // Update dimensions after loading new map
                    drawMap();
                    console.log('Map loaded and drawn successfully.');
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

zoomInBtn.addEventListener('click', zoomIn);
zoomOutBtn.addEventListener('click', zoomOut);

mapCanvas.addEventListener('mousemove', (event) => {
    const rect = mapCanvas.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const y = event.clientY - rect.top;

    const cellSize = getCellSize();
    const col = Math.floor(x / cellSize);
    const row = Math.floor(y / cellSize);

    coordRowSpan.textContent = row;
    coordColSpan.textContent = col;
});

mapCanvas.addEventListener('click', (event) => {
    const rect = mapCanvas.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const y = event.clientY - rect.top;

    const cellSize = getCellSize();
    const col = Math.floor(x / cellSize);
    const row = Math.floor(y / cellSize);

    const selectedBrush = brushTool.value;
    const enableSymmetry = symmetryToggle.checked;

    // Apply change to the clicked cell
    if (row >= 0 && row < mapRows && col >= 0 && col < mapCols) {
        currentMap[row][col] = selectedBrush;
    }

    // Apply symmetry if enabled (180-degree rotational symmetry)
    if (enableSymmetry) {
        const symRow = mapRows - 1 - row;
        const symCol = mapCols - 1 - col;
        if (symRow >= 0 && symRow < mapRows && symCol >= 0 && symCol < mapCols) {
            currentMap[symRow][symCol] = selectedBrush;
        }
    }
    drawMap(); // Redraw the map after changes
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
