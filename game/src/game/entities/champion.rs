use std::usize;

use crate::game::cell::CellContent;
use crate::game::{cell::PlayerId, Action, Board};
use crate::errors::GameError;

#[derive(Debug)]
pub struct Champion {
    pub player_id: PlayerId,
    pub team_id: u8,
    pub row: u16,
    pub col: u16,
}

impl Champion {
    pub fn new(player_id: PlayerId, team_id: u8, row: u16, col: u16) -> Self {
        Champion { player_id, team_id, row, col }
    }

    pub fn take_action(&mut self, action: &Action, board: &mut Board) -> Result<(), GameError> {
        let _ = match action {
            Action::MoveUp => self.move_champion(board, -1, 0),
            Action::MoveDown => self.move_champion(board, 1, 0),
            Action::MoveLeft => self.move_champion(board, 0, -1),
            Action::MoveRight => self.move_champion(board, 0, 1),
            Action::Action1 => Ok(()),
            Action::Action2 => Ok(()),
            Action::InvalidAction =>  Err(GameError::InvalidInput("InvalidAction found".to_string()))
        };

        Ok(())
    }

    fn move_champion(&mut self, board: &mut Board, d_row: isize, d_col: isize) -> Result<(), GameError> {
        let new_row = if d_row < 0 {
            self.row.saturating_sub(d_row.unsigned_abs() as u16)
        } else {
            self.row.saturating_add(d_row as u16)
        };

        let new_col = if d_col < 0 {
            self.col.saturating_sub(d_col.unsigned_abs() as u16)
        } else {
            self.col.saturating_add(d_col as u16)
        };

        if new_row >= board.rows as u16 || new_col >= board.cols as u16 {
            return Err(GameError::CannotMoveHere(self.player_id))
        }

        if let Some(new_cell) = board.get_cell(new_row as usize, new_col as usize) {
            if new_cell.is_passable() {
                new_cell.content = Some(CellContent::Champion(self.player_id, self.team_id));
                if let Some(old_cell) = board.get_cell(self.row as usize, self.col as usize) {
                    self.row = new_row;
                    self.col = new_col;
                    old_cell.content = None;
                    return Ok(())
                } else {
                    return Err(GameError::CannotMoveHere(self.player_id))
                }
            } else {
                return Err(GameError::NotFoundCell)
            }
        } else {
                return Err(GameError::NotFoundCell)
        }
    }
}



