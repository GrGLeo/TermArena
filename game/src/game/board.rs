use super::cell::{EncodedCellValue, BaseTerrain, CellContent, Cell};

#[derive(Debug)]
pub struct Board {
    grid: Vec<Vec<Cell>>,
    pub rows: usize,
    pub cols: usize,
}

impl Board {
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

    pub fn place_cell(&mut self, content: CellContent, champ_row: usize, champ_col: usize) {
        if let Some(row) = self.grid.get_mut(champ_row) {
            if let Some(cell) = row.get_mut(champ_col) {
                cell.content = Some(content);
            }
        }
                
    }

    fn center_around_player(&self, player_row: u16, player_col: u16) -> Vec<Vec<&Cell>> {
        let view_height = 21;
        let view_width = 51;

        let grid_height = self.grid.len() as u16;
        let grid_width = self.grid.get(0).map_or(0, |r| r.len() as u16);

        let min_row = (player_row as i16 - (view_height / 2) as i16).max(0) as u16;
        let max_row = (player_row + (view_height / 2)).min(grid_height);

        let min_col = (player_col as i16 - (view_width / 2) as i16).max(0) as u16;
        let max_col = (player_col + (view_width / 2)).min(grid_width);

        let player_grid: Vec<Vec<&Cell>> = self.grid[min_row as usize..=max_row as usize]
            .iter()
            .map(|row| &row[min_col as usize..=max_col as usize])
            .map(|slice| slice.iter().collect())
            .collect();
        player_grid
    }

    pub fn run_length_encode(&self, player_row: u16, player_col: u16) -> Vec<u8> {
        let flattened_grid: Vec<&Cell> = self.center_around_player(player_row, player_col)
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
