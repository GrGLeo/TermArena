use std::usize;

use crate::cell::{Cell, BaseTerrain};
use crate::config;



#[derive(Debug)]
pub struct Board {
    grid: Vec<Vec<Cell>>,
}

impl Board {
    pub fn new() -> Self {
        let row = config::HEIGHT as usize;
        let col = config::WIDTH as usize;

        let grid = (0..row)
            .map(|_| {
                (0..col)
                    .map(|_| Cell::new(BaseTerrain::Floor))
                    .collect()
            })
        .collect();
        Board{grid}
    }

    pub fn center_around_player(&self, player_row: u16, player_col: u16) -> Vec<Vec<&Cell>> {
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
}
