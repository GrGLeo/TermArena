use std::time::{Duration, Instant};
use std::usize;

use crate::errors::GameError;
use crate::game::animation::melee::MeleeAnimation;
use crate::game::animation::AnimationTrait;
use crate::game::Cell;
use crate::game::cell::CellContent;
use crate::game::{Action, Board, cell::PlayerId};

use super::{Fighter, Stats};

#[derive(Debug)]
pub struct Champion {
    pub player_id: PlayerId,
    pub team_id: u8,
    stats: Stats,
    death_counter: u8,
    death_timer: Instant,
    last_attacked: Instant,
    pub row: u16,
    pub col: u16,
}

impl Champion {
    pub fn new(player_id: PlayerId, team_id: u8, row: u16, col: u16) -> Self {
        let stats = Stats {
            attack_damage: 10,
            attack_speed: Duration::from_millis(2500),
            health: 200,
            armor: 205,
        };

        Champion {
            player_id,
            stats,
            death_counter: 0,
            death_timer: Instant::now(),
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
                board.move_cell(self.row as usize, self.col as usize, new_row as usize, new_col as usize);
                self.row = new_row;
                self.col = new_col;
                Ok(())
                /* legacy
                new_cell.content = Some(CellContent::Champion(self.player_id, self.team_id));
                if let Some(old_cell) = board.get_cell(self.row as usize, self.col as usize) {
                    self.row = new_row;
                    self.col = new_col;
                    old_cell.content = None;
                    return Ok(());
                } else {
                    return Err(GameError::CannotMoveHere(self.player_id));
                }
                */ 
            } else {
                return Err(GameError::NotFoundCell);
            }
        } else {
            return Err(GameError::NotFoundCell);
        }
    }
    
    pub fn place_at_base(&mut self, board: &mut Board) {
        let old_row = self.row;
        let old_col = self.col;
        self.row = 197;
        self.col = 2;
        board.move_cell(old_row as usize, old_col as usize, self.row as usize, self.col as usize);
    }

    pub fn is_dead(&self) -> bool {
        if Instant::now() > self.death_timer {
            return false
        } else {
            true
        }
    }
}

impl Fighter for Champion {
    fn take_damage(&mut self, damage: u8) {
        let reduced_damage = damage.saturating_sub(self.stats.armor);
        self.stats.health = self.stats.health.saturating_sub(reduced_damage as u16);
        // Check if champion get killed
        if self.stats.health == 0 {
            self.death_counter += 1;
            let timer = ((self.death_counter as f32).sqrt() * 10.) as u64;
            self.death_timer = Instant::now() + Duration::from_secs(timer);
        }
    }

    fn can_attack(&mut self) -> Option<(u8, Box<dyn AnimationTrait>)> {
        if self.last_attacked + self.stats.attack_speed < Instant::now() {
            self.last_attacked = Instant::now();
            let animation = MeleeAnimation::new(self.player_id);
            Some((self.stats.attack_damage, Box::new(animation)))
        }
        else {
            None
        }
    }


    fn scan_range<'a>(&self, board: &'a Board) -> Option<&'a Cell> {
        // range is implied here with: 3*3 square
        let target_area = board.center_view(self.row, self.col, 3, 3);
        let center_row = target_area.len() / 2;
        let center_col = target_area[0].len() / 2;

        target_area
            .iter()
            .enumerate()
            .flat_map(|(row_index, row)| {
                row.iter().enumerate().map(move |(col_index, cell)| (row_index, col_index, cell))
            })
        .filter_map(|(row, col, cell)| {
            if let Some(content) = &cell.content {
                match content {
                    CellContent::Champion(_, team_id ) | CellContent::Tower(_, team_id) | CellContent::Minion(_, team_id) => {
                        if *team_id != self.team_id {
                            Some((row, col, cell))
                        } else {
                            None
                        }
                    } 
                    _ => None
                }
            } else {
                None
            }
        })
        .min_by(|(r1, c1, _), (r2, c2, _)| {
            let dist1 = r1.abs_diff(center_row) + c1.abs_diff(center_col);
            let dist2 = r2.abs_diff(center_row) + c2.abs_diff(center_col);
            dist1.cmp(&dist2)
        })
        .map(|(_, _, &cell)| cell)
    }
}
