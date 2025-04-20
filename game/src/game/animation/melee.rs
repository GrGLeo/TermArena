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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_new_melee_animation() {
        let player_id = 10;
        let animation = MeleeAnimation::new(player_id);

        assert_eq!(animation.player_id, player_id);
        assert_eq!(animation.cycle, 8); // Assuming default cycle is 8 based on implementation
        assert_eq!(animation.counter, 0);
        assert!(animation.last_drawn_row.is_none());
        assert!(animation.last_drawn_col.is_none());
        assert_eq!(animation.animation_type, CellAnimation::MeleeHit);
    }

    #[test]
    fn test_melee_animation_trait_methods() {
        let player_id = 20;
        let mut animation = MeleeAnimation::new(player_id);

        // Test get_owner_id
        assert_eq!(animation.get_owner_id(), player_id);

        // Test get_animation_type
        assert_eq!(animation.get_animation_type(), CellAnimation::MeleeHit);

        // Test attach_target (Note: implementation sets the owner_id to target_id)
        let target_id = 30;
        animation.attach_target(target_id);
        assert_eq!(
            animation.get_owner_id(),
            target_id,
            "attach_target should update the owner_id"
        );

        // Test get_last_drawn_pos (initially None)
        assert!(animation.get_last_drawn_pos().is_none());

        // get_last_drawn_pos should return Some after a Draw command (tested in next_frame sequence test)
    }

    #[test]
    fn test_melee_animation_next_frame_sequence() {
        let player_id = 40;
        let mut animation = MeleeAnimation::new(player_id);
        let owner_row = 5;
        let owner_col = 5;

        // The animation cycle is 8 frames (counter goes from 1 to 8)
        for i in 1..=animation.cycle {
            let command = animation.next_frame(owner_row, owner_col);

            // Expect a Draw command for the first 8 frames
            match command {
                AnimationCommand::Draw {
                    row,
                    col,
                    animation_type,
                } => {
                    println!("Frame {} - Draw at ({}, {})", i, row, col);
                    // Verify the animation type is correct
                    assert_eq!(animation_type, CellAnimation::MeleeHit);

                    // Verify the calculated position based on the owner and counter
                    // The calculate_next_pos logic moves in 8 directions
                    let expected_pos = match i {
                        1 => Some((owner_row, owner_col.saturating_add(1))), // Right
                        2 => Some((owner_row.saturating_add(1), owner_col.saturating_add(1))), // Down-Right
                        3 => Some((owner_row.saturating_add(1), owner_col)), // Down
                        4 => Some((owner_row.saturating_add(1), owner_col.saturating_sub(1))), // Down-Left
                        5 => Some((owner_row, owner_col.saturating_sub(1))), // Left
                        6 => Some((owner_row.saturating_sub(1), owner_col.saturating_sub(1))), // Up-Left
                        7 => Some((owner_row.saturating_sub(1), owner_col)),                   // Up
                        8 => Some((owner_row.saturating_sub(1), owner_col.saturating_add(1))), // Up-Right
                        _ => None, // Should not happen in this loop
                    };
                    let (expected_row, expected_col) =
                        expected_pos.expect("Expected position should be Some");
                    assert_eq!(
                        row, expected_row,
                        "Frame {} - Expected row {}",
                        i, expected_row
                    );
                    assert_eq!(
                        col, expected_col,
                        "Frame {} - Expected col {}",
                        i, expected_col
                    );

                    // Verify last_drawn_pos is updated
                    let last_drawn = animation.get_last_drawn_pos();
                    assert!(
                        last_drawn.is_some(),
                        "Frame {} - last_drawn_pos should be Some",
                        i
                    );
                    assert_eq!(
                        last_drawn.unwrap(),
                        (row, col),
                        "Frame {} - last_drawn_pos should match drawn position",
                        i
                    );
                }
                _ => panic!("Frame {} - Expected Draw command, but got {:?}", i, command),
            }
        }

        // After 8 frames, the next call should return Done
        let final_command = animation.next_frame(owner_row, owner_col);
        assert_eq!(
            final_command,
            AnimationCommand::Done,
            "After cycle, next_frame should return Done"
        );

        // After Done, last_drawn_pos should be None
        assert!(
            animation.get_last_drawn_pos().is_none(),
            "After Done command, last_drawn_pos should be None"
        );
    }
}
