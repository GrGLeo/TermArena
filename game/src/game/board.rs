use crate::game::cell::Team;

use super::cell::{BaseTerrain, Cell, CellAnimation, CellContent, EncodedCellValue};
use super::entities::tower::Tower;
use serde::Deserialize;
use std::fs::File;
use std::io::Read;
use std::usize;

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
        for (i, row) in board_layout.layout.iter().enumerate() {
            let mut grid_row = Vec::with_capacity(board_layout.cols);
            for (j, cell) in row.iter().enumerate() {
                let base = match cell.as_str() {
                    "wall" => BaseTerrain::Wall,
                    "floor" => BaseTerrain::Floor,
                    "bush" => BaseTerrain::Bush,
                    _ => BaseTerrain::Floor, // Default case
                };
                grid_row.push(Cell::new(base, (i as u16, j as u16)));
            }
            grid.push(grid_row);
        }
        let mut board = Board {
            grid,
            rows: board_layout.rows,
            cols: board_layout.cols,
        };
        
        // For now we place the tower here
        Tower::new(1, Team::Blue, 196, 150).place_tower(&mut board);  
        Tower::new(2, Team::Red, 150, 196).place_tower(&mut board);  
        println!("Tower: {:?}", board.grid[196][150]);
        println!("Tower: {:?}", board.grid[196][196]);

        Ok(board)
    }

    pub fn new(rows: usize, cols: usize) -> Self {
        let mut grid = Vec::with_capacity(rows);
        for i in 0..rows {
            let mut row = Vec::with_capacity(cols);
            for j in 0..cols {
                row.push(Cell::new(BaseTerrain::Floor, (i as u16, j as u16)));
            }
            grid.push(row)
        }
        Board { grid, rows, cols }
    }

    pub fn get_cell(&mut self, row: usize, col: usize) -> Option<&Cell> {
        self.grid.get(row).and_then(|r| r.get(col))
    }

    pub fn change_base(&mut self, new_base: BaseTerrain, row: usize, col: usize) {
        let cell = &mut self.grid[row][col];
        cell.base = new_base
    }

    pub fn move_cell(&mut self, old_row: usize, old_col: usize, new_row: usize, new_col: usize) {
        let content: Option<CellContent>;
        {
            let old_cell = &mut self.grid[old_row][old_col];
            content = old_cell.content.clone();
            old_cell.content = None;
        }
        let new_cell = &mut self.grid[new_row][new_col];
        new_cell.content = content;
    }

    pub fn place_cell(&mut self, content: CellContent, champ_row: usize, champ_col: usize) {
        if let Some(row) = self.grid.get_mut(champ_row) {
            if let Some(cell) = row.get_mut(champ_col) {
                cell.content = Some(content);
            }
        }
    }

    pub fn clear_cell(&mut self, row: usize, col: usize) {
        self.grid[row][col].content = None;
    }

    pub fn place_animation(&mut self, animation: CellAnimation, animation_row: usize, animation_col: usize) {
        if let Some(row) = self.grid.get_mut(animation_row) {
            if let Some(cell) = row.get_mut(animation_col) {
                cell.animation = Some(animation);
            }
        }
    }

    pub fn clean_animation(&mut self, row: usize, col: usize) {
        self.grid[row][col].animation = None;
    }

    pub fn center_view(&self, player_row: u16, player_col: u16, view_height: u16, view_width: u16) -> Vec<Vec<&Cell>> {
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

#[cfg(test)]
mod tests {
    use crate::game::cell::Team;

    use super::*;

    #[test]
    fn test_new_board() {
        let rows = 10;
        let cols = 20;
        let board = Board::new(rows, cols);

        assert_eq!(board.rows, rows);
        assert_eq!(board.cols, cols);
        assert_eq!(board.grid.len(), rows);
        for row in board.grid.iter() {
            assert_eq!(row.len(), cols);
            for cell in row.iter() {
                assert_eq!(cell.base, BaseTerrain::Floor);
                assert!(cell.content.is_none());
                assert!(cell.animation.is_none());
            }
        }
    }

    #[test]
    fn test_change_base() {
        let mut board = Board::new(5, 5);
        let row = 2;
        let col = 3;
        let new_base = BaseTerrain::Wall;

        board.change_base(new_base, row, col);

        let cell = board.get_cell(row, col).expect("Cell should exist");
        assert_eq!(cell.base, new_base);
    }

    #[test]
    fn test_place_and_clear_cell() {
        let mut board = Board::new(5, 5);
        let row = 1;
        let col = 1;
        let content = CellContent::Champion(1, Team::Red);

        // Place content
        board.place_cell(content.clone(), row, col);
        let cell_after_place = board.get_cell(row, col).expect("Cell should exist");
        assert_eq!(cell_after_place.content, Some(content));

        // Clear content
        board.clear_cell(row, col);
        let cell_after_clear = board.get_cell(row, col).expect("Cell should exist");
        assert!(cell_after_clear.content.is_none());
    }

    #[test]
    fn test_place_and_clean_animation() {
        let mut board = Board::new(5, 5);
        let row = 3;
        let col = 4;
        let animation = CellAnimation::MeleeHit;

        // Place animation
        board.place_animation(animation.clone(), row, col);
        let cell_after_place = board.get_cell(row, col).expect("Cell should exist");
        assert_eq!(cell_after_place.animation, Some(animation));

        // Clean animation
        board.clean_animation(row, col);
        let cell_after_clean = board.get_cell(row, col).expect("Cell should exist");
        assert!(cell_after_clean.animation.is_none());
    }

    #[test]
    fn test_get_cell() {
        let mut board = Board::new(5, 5);

        // Test valid coordinates
        let cell = board.get_cell(2, 2);
        assert!(cell.is_some(), "Should get a cell at valid coordinates");

        // Test out-of-bounds row
        let cell_out_of_bounds_row = board.get_cell(5, 2);
        assert!(cell_out_of_bounds_row.is_none(), "Should not get a cell with out-of-bounds row");

        // Test out-of-bounds col
        let cell_out_of_bounds_col = board.get_cell(2, 5);
        assert!(cell_out_of_bounds_col.is_none(), "Should not get a cell with out-of-bounds col");

        // Test out-of-bounds row and col
        let cell_out_of_bounds_both = board.get_cell(5, 5);
        assert!(cell_out_of_bounds_both.is_none(), "Should not get a cell with out-of-bounds row and col");
    }

    #[test]
    fn test_move_cell() {
        let mut board = Board::new(5, 5);
        let old_row = 1;
        let old_col = 1;
        let new_row = 3;
        let new_col = 3;
        let content = CellContent::Champion(1, Team::Red);

        // Place initial content
        board.place_cell(content.clone(), old_row, old_col);
        let cell_at_old_pos_before_move = board.get_cell(old_row, old_col).expect("Cell should exist");
        assert_eq!(cell_at_old_pos_before_move.content, Some(content.clone()), "Content should be at the old position before move");

        // Move the cell content
        board.move_cell(old_row, old_col, new_row, new_col);

        // Check the old position (should be empty)
        let cell_at_old_pos_after_move = board.get_cell(old_row, old_col).expect("Cell should exist");
        assert!(cell_at_old_pos_after_move.content.is_none(), "Old position should be empty after move");

        // Check the new position (should have the content)
        let cell_at_new_pos_after_move = board.get_cell(new_row, new_col).expect("Cell should exist");
        assert_eq!(cell_at_new_pos_after_move.content, Some(content), "New position should have the content after move");
    }

    #[test]
    fn test_center_view() {
        // Create a larger board to test view centering and edge cases
        let rows = 50;
        let cols = 50;
        let mut board = Board::new(rows, cols);

        // Change some base terrains to make the view distinguishable
        board.change_base(BaseTerrain::Wall, 0, 0); // Top-left corner
        board.change_base(BaseTerrain::Bush, rows - 1, cols - 1); // Bottom-right corner
        board.change_base(BaseTerrain::Wall, 0, cols - 1); // Top-right corner
        board.change_base(BaseTerrain::Bush, rows - 1, 0); // Bottom-left corner
        board.change_base(BaseTerrain::Wall, rows / 2, cols / 2); // Center

        let view_height = 5;
        let view_width = 7;

        // Test view when player is in the center (not near edges)
        let center_player_row = rows / 2;
        let center_player_col = cols / 2;
        let center_view = board.center_view(center_player_row as u16, center_player_col as u16, view_height, view_width);

        assert_eq!(center_view.len(), view_height as usize);
        assert_eq!(center_view[0].len(), view_width as usize);
        // Check a cell that should be in the center of the view
        // The center of the view is (view_height/2, view_width/2) relative to the view's top-left
        // This corresponds to the player's position on the board
        let center_view_center_row = view_height as usize / 2;
        let center_view_center_col = view_width as usize / 2;
        assert_eq!(center_view[center_view_center_row][center_view_center_col].base, BaseTerrain::Wall, "Center cell in center view should be Wall");


        // Test view when player is in the top-left corner
        let top_left_player_row = 0;
        let top_left_player_col = 0;
        let top_left_view = board.center_view(top_left_player_row as u16, top_left_player_col as u16, view_height, view_width);

        assert_eq!(top_left_view.len(), view_height as usize);
        assert_eq!(top_left_view[0].len(), view_width as usize);
        assert_eq!(top_left_view[0][0].base, BaseTerrain::Wall, "Top-left cell in top-left view should be Wall");


        // Test view when player is in the bottom-right corner
        let bottom_right_player_row = rows - 1;
        let bottom_right_player_col = cols - 1;
         let bottom_right_view = board.center_view(bottom_right_player_row as u16, bottom_right_player_col as u16, view_height, view_width);

        assert_eq!(bottom_right_view.len(), view_height as usize);
        assert_eq!(bottom_right_view[0].len(), view_width as usize);
        let bottom_right_view_corner_row = view_height as usize - 1;
        let bottom_right_view_corner_col = view_width as usize - 1;
        assert_eq!(bottom_right_view[bottom_right_view_corner_row][bottom_right_view_corner_col].base, BaseTerrain::Bush, "Bottom-right cell in bottom-right view should be Bush");

        // Test view when player is near the top edge
        let top_edge_player_row = 1;
        let top_edge_player_col = cols / 2;
         let top_edge_view = board.center_view(top_edge_player_row as u16, top_edge_player_col as u16, view_height, view_width);

        assert_eq!(top_edge_view.len(), view_height as usize);
        assert_eq!(top_edge_view[0].len(), view_width as usize);


        // Test view when player is near the left edge
        let left_edge_player_row = rows / 2;
        let left_edge_player_col = 1;
         let left_edge_view = board.center_view(left_edge_player_row as u16, left_edge_player_col as u16, view_height, view_width);

        assert_eq!(left_edge_view.len(), view_height as usize);
        assert_eq!(left_edge_view[0].len(), view_width as usize);
    }

    #[test]
    fn test_run_length_encode() {
        // Create a small board with varied cell types
        let mut board = Board::new(3, 4);
        board.change_base(BaseTerrain::Wall, 0, 0);
        board.change_base(BaseTerrain::Bush, 0, 1);
        board.place_cell(CellContent::Champion(1, Team::Red), 1, 1);
        board.place_animation(CellAnimation::MeleeHit, 2, 3);
        board.change_base(BaseTerrain::TowerDestroyed, 2, 0);

        // Set player position and view dimensions to cover the entire small board
        let player_row = 1; // Center row
        let player_col = 1; // Center col
        let view_height = 3; // Match board height
        let view_width = 4; // Match board width

        // Get the view and flatten it for RLE
        let flattened_grid: Vec<&Cell> = board.center_view(player_row, player_col, view_height, view_width)
            .into_iter()
            .flat_map(|row| row.into_iter())
            .collect();

        let mut rle: Vec<String> = Vec::new();

        if !flattened_grid.is_empty() {
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
        }

        let encoded_bytes = rle.join("|").into_bytes();

        let expected_rle = "0:1|2:1|1:3|4:1|1:2|3:1|1:2|9:1";

        assert_eq!(String::from_utf8(encoded_bytes).expect("Valid UTF-8 string"), expected_rle, "Run-length encoding did not match expected output");
    }
}
