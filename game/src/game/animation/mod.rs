use std::fmt::Debug;

use super::{
    PlayerId,
    cell::{CellAnimation, Team},
};

pub mod melee;
pub mod tower;

#[derive(Debug, PartialEq, Eq)]
pub enum AnimationCommand {
    Draw {
        row: u16,
        col: u16,
        animation_type: CellAnimation,
    },
    Clear {
        row: u16,
        col: u16,
    },
    Done,
}

pub trait AnimationTrait: Send + Sync + Debug {
    // Method to get the command for the next frame
    // It should update the animation's internal state (like counter, current position)
    // The owner's current position might be needed to calculate the animation location
    fn next_frame(&mut self, owner_row: u16, owner_col: u16) -> AnimationCommand;

    // Method to get the ID of the entity that owns/triggered this animation
    fn get_owner_id(&self) -> usize;

    // Method to attach the target id, for next position calculation
    fn attach_target(&mut self, target_id: PlayerId);

    // Method to get the animation type (MeleeHit, TowerHit, etc.)
    fn get_animation_type(&self) -> CellAnimation;

    // A method to get the last drawn position, so GameManager knows what to clear
    fn get_last_drawn_pos(&self) -> Option<(u16, u16)>;
}
