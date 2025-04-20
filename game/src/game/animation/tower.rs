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

    fn attach_target(&mut self, target_id: PlayerId) {
        self.target_id = target_id
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::game::PlayerId; // Import PlayerId

    #[test]
    fn test_new_tower_hit_animation() {
        let tower_row = 10;
        let tower_col = 20;
        let animation = TowerHitAnimation::new(tower_row, tower_col);

        assert_eq!(animation.target_id, 0); // Initially 0 (None)
        assert_eq!(animation.last_drawn_row, tower_row);
        assert_eq!(animation.last_drawn_col, tower_col);
        assert_eq!(animation.animation_type, CellAnimation::TowerHit);
    }

     #[test]
    fn test_tower_hit_animation_trait_methods() {
        let tower_row = 10;
        let tower_col = 20;
        let mut animation = TowerHitAnimation::new(tower_row, tower_col);

        // Test get_animation_type
        assert_eq!(animation.get_animation_type(), CellAnimation::TowerHit);

        // Test get_last_drawn_pos (initially set in new)
        assert_eq!(animation.get_last_drawn_pos(), Some((tower_row, tower_col)));

        // Test attach_target
        let target_player_id: PlayerId = 5;
        animation.attach_target(target_player_id);
        assert_eq!(animation.target_id, target_player_id);

        // Test get_owner_id (Note: implementation returns target_id)
        assert_eq!(animation.get_owner_id(), target_player_id, "get_owner_id should return the attached target_id");
    }


    #[test]
    fn test_tower_hit_animation_next_frame_sequence() {
        // Simulate a tower at a static position
        let tower_static_row = 15;
        let tower_static_col = 15;
        let target_player_id: PlayerId = 1; // Dummy target ID


        // Create animation starting at a different point, moving towards the tower's static position
        // Note: The animation's next_frame calculates steps from its last_drawn_pos towards the owner's (tower's) position.
        // Based on current code implementation, the animation moves from its starting point towards the tower's current location.
        let initial_anim_row = 10;
        let initial_anim_col = 10;
        let mut animation = TowerHitAnimation {
            target_id: 0, // Will be set by attach_target
            last_drawn_row: initial_anim_row,
            last_drawn_col: initial_anim_col,
            animation_type: CellAnimation::TowerHit,
        };

        animation.attach_target(target_player_id);


        // Step through the animation until it reaches the tower's static position
        let mut current_row = initial_anim_row;
        let mut current_col = initial_anim_col;

        // Loop a reasonable number of times to ensure it reaches the target or goes beyond if logic is flawed
        for i in 0..50 { // Max steps to reach 15,15 from 10,10 is around 5 steps
            let command = animation.next_frame(tower_static_row, tower_static_col); // Pass tower's position as owner_row, owner_col

            if current_row == tower_static_row && current_col == tower_static_col {
                 // If we have reached the target position, the command should be Done
                assert_eq!(command, AnimationCommand::Done, "Frame {} - Expected Done command after reaching target", i);
                break; // Animation is done
            }


            // Otherwise, expect a Draw command
            match command {
                AnimationCommand::Draw { row, col, animation_type } => {
                    println!("Frame {} - Draw at ({}, {}). Moving towards ({}, {})", i, row, col, tower_static_row, tower_static_col);
                    assert_eq!(animation_type, CellAnimation::TowerHit, "Frame {} - Animation type should be TowerHit", i);

                    // Verify that the animation moved one step closer to the tower's static position
                    let row_diff = tower_static_row as i16 - current_row as i16;
                    let col_diff = tower_static_col as i16 - current_col as i16;

                    let expected_next_row = current_row.saturating_add_signed(row_diff.signum());
                    let expected_next_col = current_col.saturating_add_signed(col_diff.signum());

                    assert_eq!(row, expected_next_row, "Frame {} - Expected row {}", i, expected_next_row);
                    assert_eq!(col, expected_next_col, "Frame {} - Expected col {}", i, expected_next_col);

                    current_row = row;
                    current_col = col;

                     // Verify last_drawn_pos is updated
                    let last_drawn = animation.get_last_drawn_pos();
                    assert!(last_drawn.is_some(), "Frame {} - last_drawn_pos should be Some", i);
                    assert_eq!(last_drawn.unwrap(), (row, col), "Frame {} - last_drawn_pos should match drawn position", i);

                },
                _ => panic!("Frame {} - Expected Draw command, but got {:?}", i, command),
            }
        }

         // If the loop finished without hitting the break, it means the animation didn't finish
         if current_row != tower_static_row || current_col != tower_static_col {
             panic!("Animation did not reach the target position ({}, {}) within the expected steps", tower_static_row, tower_static_col);
         }
    }
}
