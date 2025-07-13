// Constants
const CELL_SIZE = 10; // Size of each cell in pixels
const DEFAULT_ROWS = 50; // Default map rows
const DEFAULT_COLS = 50; // Default map columns

// DOM Elements
const mapFileInput = document.getElementById('mapFileInput');
const brushTool = document.getElementById('brushTool');
const symmetryToggle = document.getElementById('symmetryToggle');
const saveMapBtn = document.getElementById('saveMapBtn');
const mapCanvas = document.getElementById('mapCanvas');
const coordRowSpan = document.getElementById('coordRow');
const coordColSpan = document.getElementById('coordCol');
const ctx = mapCanvas.getContext('2d');

// Map Data
let currentMap = [];
let mapRows = DEFAULT_ROWS;
let mapCols = DEFAULT_COLS;

// Initialize a default empty map
function initializeMap(rows, cols) {
    mapRows = rows;
    mapCols = cols;
    currentMap = Array(mapRows).fill(null).map(() => Array(mapCols).fill('Floor'));
    mapCanvas.width = mapCols * CELL_SIZE;
    mapCanvas.height = mapRows * CELL_SIZE;
    drawMap();
}

// Draw the map on the canvas
function drawMap() {
    ctx.clearRect(0, 0, mapCanvas.width, mapCanvas.height);
    for (let r = 0; r < mapRows; r++) {
        for (let c = 0; c < mapCols; c++) {
            drawCell(r, c, currentMap[r][c]);
        }
    }
}

// Draw a single cell
function drawCell(row, col, type) {
    ctx.beginPath();
    ctx.rect(col * CELL_SIZE, row * CELL_SIZE, CELL_SIZE, CELL_SIZE);
    switch (type) {
        case 'Wall':
            ctx.fillStyle = '#5C6370'; // Dark Gray
            break;
        case 'Bush':
            ctx.fillStyle = '#98C379'; // Green
            break;
        case 'Floor':
        default:
            ctx.fillStyle = '#21252B'; // Very Dark Blue/Gray
            break;
    }
    ctx.fill();
    ctx.strokeStyle = '#3A3F4B'; // Slightly lighter border
    ctx.stroke();
}

// Event Listeners
mapFileInput.addEventListener('change', (event) => {
    const file = event.target.files[0];
    if (file) {
        const reader = new FileReader();
        reader.onload = (e) => {
            try {
                const data = JSON.parse(e.target.result);
                if (data.rows && data.cols && Array.isArray(data.layout)) {
                    mapRows = data.rows;
                    mapCols = data.cols;
                    currentMap = data.layout;
                    mapCanvas.width = mapCols * CELL_SIZE;
                    mapCanvas.height = mapRows * CELL_SIZE;
                    drawMap();
                } else {
                    alert('Invalid map JSON format.');
                }
            } catch (error) {
                alert('Error parsing JSON: ' + error.message);
            }
        };
        reader.readAsText(file);
    }
});

mapCanvas.addEventListener('mousemove', (event) => {
    const rect = mapCanvas.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const y = event.clientY - rect.top;

    const col = Math.floor(x / CELL_SIZE);
    const row = Math.floor(y / CELL_SIZE);

    coordRowSpan.textContent = row;
    coordColSpan.textContent = col;
});

mapCanvas.addEventListener('click', (event) => {
    const rect = mapCanvas.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const y = event.clientY - rect.top;

    const col = Math.floor(x / CELL_SIZE);
    const row = Math.floor(y / CELL_SIZE);

    const selectedBrush = brushTool.value;
    const enableSymmetry = symmetryToggle.checked;

    // Apply change to the clicked cell
    currentMap[row][col] = selectedBrush;

    // Apply symmetry if enabled
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
