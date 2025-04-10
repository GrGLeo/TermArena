use crate::cell::{Cell, BaseTerrain};
use crate::config;



#[derive(Debug)]
struct Board {
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

    pub fn center_around_player(self, row: u16, col: u16) -> Vec<Vec<Cell>> {
        todo!()
    }
}
