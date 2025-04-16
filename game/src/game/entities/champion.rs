use std::time::{Duration, Instant};
use std::usize;


use crate::errors::GameError;
use crate::game::Cell;
use crate::game::cell::CellContent;
use crate::game::{Action, Board, cell::PlayerId};

use super::{Fighter, Stats};

#[derive(Debug)]
pub struct Champion {
    pub player_id: PlayerId,
    pub team_id: u8,
    pub stats: Stats,
    pub last_attacked: Instant,
    pub row: u16,
    pub col: u16,
}

impl Champion {
    pub fn new(player_id: PlayerId, team_id: u8, row: u16, col: u16) -> Self {
        let stats = Stats {
            attack_damage: 10,
            attack_speed: Duration::from_millis(2500),
            health: 200,
            armor: 5,
        };

        Champion {
            player_id,
            stats,
            last_attacked: Instant::now(),
            team_id,
            row,
            col,
        }
    }

    pub fn take_action(&mut self, action: &Action, board: &mut Board) -> Result<(), GameError> {
        let _ = match action {
            Action::MoveUp => self.move_champion(board, -1, 0),
            Action::MoveDown => self.move_champion(board, 1, 0),
            Action::MoveLeft => self.move_champion(board, 0, -1),
            Action::MoveRight => self.move_champion(board, 0, 1),
            Action::Action1 => Ok(()),
            Action::Action2 => Ok(()),
            Action::InvalidAction => {
                Err(GameError::InvalidInput("InvalidAction found".to_string()))
            }
        };

        Ok(())
    }

    fn move_champion(
        &mut self,
        board: &mut Board,
        d_row: isize,
        d_col: isize,
    ) -> Result<(), GameError> {
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
            return Err(GameError::CannotMoveHere(self.player_id));
        }

        if let Some(new_cell) = board.get_cell(new_row as usize, new_col as usize) {
            if new_cell.is_passable() {
                new_cell.content = Some(CellContent::Champion(self.player_id, self.team_id));
                if let Some(old_cell) = board.get_cell(self.row as usize, self.col as usize) {
                    self.row = new_row;
                    self.col = new_col;
                    old_cell.content = None;
                    return Ok(());
                } else {
                    return Err(GameError::CannotMoveHere(self.player_id));
                }
            } else {
                return Err(GameError::NotFoundCell);
            }
        } else {
            return Err(GameError::NotFoundCell);
        }
    }
}

impl Fighter for Champion {
    fn take_damage(&mut self, damage: u8) {
        let reduced_damage = damage.saturating_sub(self.stats.armor);
        self.stats.health -= reduced_damage as u16;
    }

    fn can_attack(&mut self) -> Option<u8> {
        if self.last_attacked + self.stats.attack_speed < Instant::now() {
            self.last_attacked = Instant::now();
            Some(self.stats.attack_damage)
        }
        else {
            None
        }
    }

    fn scan_range<'a>(&self, board: &'a Board) -> Vec<&'a Cell> {
        // range is implied here with: 6, 8
        let target_area = board.center_view(self.row, self.col, 1, 1);
        target_area
            .iter()
            .flat_map(|row| row.iter())
            .filter(|&&cell| {
                if let Some(content) = &cell.content {
                    match content {
                        CellContent::Champion(_, team_id)
                        | CellContent::Minion(_, team_id)
                        | CellContent::Tower(_, team_id) => *team_id != self.team_id,
                        _ => false,
                    }
                } else {
                    false
                }
            })
            .map(|&cell| cell)
            .collect()
    }
}
