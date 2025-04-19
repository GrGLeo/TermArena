use super::{AnimationCommand, AnimationTrait};
use crate::game::{PlayerId, cell::CellAnimation};

#[derive(Debug)]
pub struct TowerHitAnimation {
    pub target_id: PlayerId,
    last_drawn_row: u16,
    last_drawn_col: u16,
    animation_type: CellAnimation,
}

impl TowerHitAnimation {
    pub fn new(tower_row: u16, tower_col: u16) -> Self {
        TowerHitAnimation {
            target_id: 0, // 0 here is equal to None, no target are attach yet
            last_drawn_row: tower_row,
            last_drawn_col: tower_col,
            animation_type: CellAnimation::TowerHit,
        }
    }
    
    fn attach_target(&mut self, target_id: PlayerId) {
        self.target_id = target_id
    }

    fn calculate_next_pos(&mut self, target_row: u16, target_col: u16) -> Option<(u16, u16)> {
        // We calculate the difference between target and last_drawn, and take one step
        // in the target direction
        let row_step = (target_row as i16 - self.last_drawn_row as i16).signum();
        let col_step = (target_col as i16 - self.last_drawn_col as i16).signum();
        // If both step are at 0, then target is hit.
        if row_step == 0 && col_step == 0 {
            return None;
        }
        // We return the new position
        return Some((
            self.last_drawn_row.saturating_add_signed(row_step),
            self.last_drawn_col.saturating_add_signed(col_step),
        ));
    }
}

impl AnimationTrait for TowerHitAnimation {
    fn get_owner_id(&self) -> usize {
        self.target_id
    }
    fn get_animation_type(&self) -> CellAnimation {
        self.animation_type.clone()
    }
    fn get_last_drawn_pos(&self) -> Option<(u16, u16)> {
        return Some((self.last_drawn_row, self.last_drawn_col));
    }

    fn next_frame(&mut self, row: u16, col: u16) -> AnimationCommand {
        if let Some((next_row, next_col)) = self.calculate_next_pos(row, col) {
            self.last_drawn_row = next_row;
            self.last_drawn_col = next_col;

            AnimationCommand::Draw {
                row: next_row,
                col: next_col,
                animation_type: self.animation_type.clone(),
            }
        } else {
            AnimationCommand::Done
        }
    }
}
