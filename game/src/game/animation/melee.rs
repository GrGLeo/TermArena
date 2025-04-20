// In animation/melee.rs
use super::{AnimationCommand, AnimationTrait};
use crate::{
    errors::GameError,
    game::{Board, Champion, PlayerId, cell::CellAnimation},
};

#[derive(Debug)]
pub struct MeleeAnimation {
    pub player_id: PlayerId,
    cycle: u8,
    counter: u8,
    last_drawn_row: Option<u16>,
    last_drawn_col: Option<u16>,
    animation_type: CellAnimation,
}

impl MeleeAnimation {
    pub fn new(player_id: PlayerId) -> Self {
        MeleeAnimation {
            player_id,
            cycle: 8,
            counter: 0,
            last_drawn_row: None,
            last_drawn_col: None,
            animation_type: CellAnimation::MeleeHit,
        }
    }

    // This logic might need adjustment based on attack direction,
    // but let's stick to the 8 points for now.
    fn calculate_next_pos(&self, owner_row: u16, owner_col: u16) -> Option<(u16, u16)> {
        match self.counter {
            1 => Some((owner_row, owner_col.saturating_add(1))),
            2 => Some((owner_row.saturating_add(1), owner_col.saturating_add(1))),
            3 => Some((owner_row.saturating_add(1), owner_col)),
            4 => Some((owner_row.saturating_add(1), owner_col.saturating_sub(1))),
            5 => Some((owner_row, owner_col.saturating_sub(1))),
            6 => Some((owner_row.saturating_sub(1), owner_col.saturating_sub(1))),
            7 => Some((owner_row.saturating_sub(1), owner_col)),
            8 => Some((owner_row.saturating_sub(1), owner_col.saturating_add(1))),
            _ => None, // Animation finished
        }
    }
}

impl AnimationTrait for MeleeAnimation {
    fn get_owner_id(&self) -> usize {
        self.player_id
    }

    fn get_animation_type(&self) -> CellAnimation {
        self.animation_type.clone()
    }

    fn attach_target(&mut self, target_id: PlayerId) {
        self.player_id = target_id
    }

    fn get_last_drawn_pos(&self) -> Option<(u16, u16)> {
        match (self.last_drawn_row, self.last_drawn_col) {
            (Some(r), Some(c)) => Some((r, c)),
            _ => None,
        }
    }

    fn next_frame(&mut self, owner_row: u16, owner_col: u16) -> AnimationCommand {
        self.counter = self.counter.saturating_add(1);

        if self.counter > self.cycle {
            // Animation is done
            self.last_drawn_row = None;
            self.last_drawn_col = None;
            AnimationCommand::Done
        } else {
            // Calculate the position for the current frame
            if let Some((next_row, next_col)) = self.calculate_next_pos(owner_row, owner_col) {
                // Update internal state to remember where we are drawing this frame
                self.last_drawn_row = Some(next_row);
                self.last_drawn_col = Some(next_col);
                // Return the Draw command
                AnimationCommand::Draw {
                    row: next_row,
                    col: next_col,
                    animation_type: self.animation_type.clone(),
                }
            } else {
                // Should not happen if counter <= cycle, but handle defensively
                self.last_drawn_row = None;
                self.last_drawn_col = None;
                AnimationCommand::Done
            }
        }
    }
}
