use super::cell::{BaseTerrain, Cell, CellContent, EncodedCellValue, TowerId};
use super::entities::tower::Tower;
use serde::Deserialize;
use std::fs::File;
use std::io::Read;

#[derive(Deserialize)]
struct BoardLayout {
    rows: usize,
    cols: usize,
    layout: Vec<Vec<String>>,
}

#[derive(Debug)]
pub struct Board {
    grid: Vec<Vec<Cell>>,
    pub rows: usize,
    pub cols: usize,
}

impl Board {
    pub fn from_json(file_path: &str) -> Result<Self, Box<dyn std::error::Error>> {
        let mut file = File::open(file_path)?;
        let mut contents = String::new();
        file.read_to_string(&mut contents)?;
        
        let board_layout: BoardLayout = serde_json::from_str(&contents)?;
        let mut grid = Vec::with_capacity(board_layout.rows);
        for row in board_layout.layout {
            let mut grid_row = Vec::with_capacity(board_layout.cols);
            for cell in row {
                let base = match cell.as_str() {
                    "wall" => BaseTerrain::Wall,
                    "floor" => BaseTerrain::Floor,
                    "bush" => BaseTerrain::Bush,
                    _ => BaseTerrain::Floor, // Default case
                };
                grid_row.push(Cell::new(base));
            }
            grid.push(grid_row);
        }
        let mut board = Board {
            grid,
            rows: board_layout.rows,
            cols: board_layout.cols,
        };
        
        // For now we place the tower here
        Tower::new(1, 1, 196, 150).place_tower(&mut board);  
        Tower::new(2, 2, 150, 196).place_tower(&mut board);  
        println!("Tower: {:?}", board.grid[196][150]);
        println!("Tower: {:?}", board.grid[196][196]);

        Ok(board)
    }

    pub fn new(rows: usize, cols: usize) -> Self {
        let mut grid = Vec::with_capacity(rows);
        for _ in 0..rows {
            let mut row = Vec::with_capacity(cols);
            for _ in 0..cols {
                row.push(Cell::new(BaseTerrain::Floor));
            }
            grid.push(row)
        }
        Board { grid, rows, cols }
    }

    pub fn get_cell(&mut self, row: usize, col: usize) -> Option<&mut Cell> {
        self.grid.get_mut(row).and_then(|r| r.get_mut(col))
    }

    pub fn place_cell(&mut self, content: CellContent, champ_row: usize, champ_col: usize) {
        if let Some(row) = self.grid.get_mut(champ_row) {
            if let Some(cell) = row.get_mut(champ_col) {
                cell.content = Some(content);
            }
        }
                
    }

    pub fn center_view(&self, player_row: u16, player_col: u16, view_height: u16, view_width: u16) -> Vec<Vec<&Cell>> {
        let view_height = 21;
        let view_width = 51;

        let grid_height = self.grid.len() as u16;
        let grid_width = self.grid.get(0).map_or(0, |r| r.len() as u16);

        let half_height = view_height / 2;
        let half_width = view_width / 2;

        // Calculate  potential min and max row
        let mut min_row = (player_row as i16 - half_height as i16).max(0) as u16;
        let mut max_row = (player_row + half_height).min(grid_height - 1);

        // Adjust if view hit the top
        if min_row == 0 {
            max_row = (view_height - 1).min(grid_height -1);
        }
        // Adjust if view hit the bottom
        if max_row == grid_height - 1 {
            min_row = (grid_height - view_height).max(0);
        }

        // Calculate potential min and max col
        let mut min_col = (player_col as i16 - half_width as i16).max(0) as u16;
        let mut max_col = (player_col + half_width).min(grid_width - 1);
        // Adjust if with hit the left
        if min_col == 0 {
            max_col = (view_width - 1).min(grid_width - 1);
        }
        // Adjust if view hit the right
        if max_col == grid_width - 1 {
            min_col = (grid_width - view_width).max(0);
        }

        self.grid[min_row as usize..= max_row as usize]
            .iter()
            .map(|row| &row[min_col as usize..= max_col as usize])
            .map(|slice| slice.iter().collect())
            .collect()
    }

    pub fn run_length_encode(&self, player_row: u16, player_col: u16) -> Vec<u8> {
        let flattened_grid: Vec<&Cell> = self.center_view(player_row, player_col, 21, 51)
            .into_iter()
            .flat_map(|row| row.into_iter())
            .collect();
        let mut rle: Vec<String> = Vec::new();

        if flattened_grid.is_empty() {
            return Vec::new();
        }

        let mut current_cell_value: EncodedCellValue = EncodedCellValue::from(flattened_grid[0]);
        let mut count = 1;

        for i in 1..flattened_grid.len() {
            let encoded_value = EncodedCellValue::from(flattened_grid[i]);
            if encoded_value == current_cell_value {
                count += 1;
            } else {
                rle.push(format!("{}:{}", current_cell_value as u8, count));
                current_cell_value = encoded_value;
                count = 1;
            }
        }
        rle.push(format!("{}:{}", current_cell_value as u8, count));
        rle.join("|").into_bytes()
    }
}
