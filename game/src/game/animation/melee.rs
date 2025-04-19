use crate::{errors::GameError, game::{cell::CellAnimation, Board, Champion, PlayerId}};

#[derive(Debug)]
pub struct MeleeAnimation {
    pub player_id: PlayerId,
    cycle: u8,
    counter: u8,
    row: Option<u16>,
    col: Option<u16>,
    animation: CellAnimation,
}

impl MeleeAnimation {
    pub fn new(player_id: PlayerId) -> Self {
        MeleeAnimation {
            player_id,
            cycle: 8,
            counter: 0,
            row: None,
            col: None,
            animation: CellAnimation::MeleeHit,
        }
    }

    pub fn get_id(&self) -> &usize {
        &self.player_id
    }

    pub fn next(&mut self, row: u16, col: u16) -> Result<(u16, u16), GameError> {
        self.counter = self.counter.saturating_add(1);
        if self.counter > self.cycle {
            return Err(GameError::InvalidAnimation)
        }
        match self.counter {
            1 => {
                let new_row = row;
                let new_col = col + 1;
                Ok((new_row, new_col))
            }
            2 => {
                let new_row = row + 1;
                let new_col = col + 1;
                Ok((new_row, new_col))
            }
            3 => {
                let new_row = row + 1;
                let new_col = col;
                Ok((new_row, new_col))
            }
            4 => {
                let new_row = row + 1;
                let new_col = col - 1;
                Ok((new_row, new_col))
            }
            5 => {
                let new_row = row;
                let new_col = col - 1;
                Ok((new_row, new_col))
            }
            6 => {
                let new_row = row - 1;
                let new_col = col - 1;
                Ok((new_row, new_col))
            }
            7 => {
                let new_row = row - 1;
                let new_col = col;
                Ok((new_row, new_col))
            }
            8 => {
                let new_row = row - 1;
                let new_col = col + 1;
                Ok((new_row, new_col))
            }
            _ => {
                return Err(GameError::InvalidAnimation)
            }
        }
    }

    pub fn clean(&mut self, board: &mut Board) {
        match (self.row, self.col) {
            (None, None) => {
            }
            (Some(row), Some(col)) => {
                board.clean_animation(row as usize, col as usize);
            }
            (_, _) => {
            }
        }
    }

    pub fn draw(&mut self, row: u16, col: u16, board: &mut Board) -> Result<(), GameError> {
        board.place_animation(CellAnimation::MeleeHit, row as usize, col as usize);
        self.row = Some(row);
        self.col = Some(col);
        Ok(())
    }
}
