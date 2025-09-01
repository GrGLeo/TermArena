use super::{AnimationCommand, AnimationTrait};
use crate::game::{PlayerId, cell::CellAnimation};

#[derive(Debug)]
pub struct MeleeAnimation {
    pub owner_id: PlayerId,
    pub target_id: Option<PlayerId>,
    cycle: u8,
    counter: u8,
    last_drawn_row: Option<u16>,
    last_drawn_col: Option<u16>,
}

impl MeleeAnimation {
    pub fn new(owner_id: PlayerId) -> Self {
        MeleeAnimation {
            owner_id,
            target_id: None,
            cycle: 2,
            counter: 0,
            last_drawn_row: None,
            last_drawn_col: None,
        }
    }
}

impl AnimationTrait for MeleeAnimation {
    fn get_owner_id(&self) -> usize {
        self.target_id.unwrap_or(self.owner_id)
    }

    fn get_animation_type(&self) -> CellAnimation {
        if self.counter == 1 {
            CellAnimation::MeleeHitOne
        } else {
            CellAnimation::MeleeHitTwo
        }
    }

    fn attach_target(&mut self, target_id: PlayerId) {
        self.target_id = Some(target_id);
    }

    fn get_last_drawn_pos(&self) -> Option<(u16, u16)> {
        match (self.last_drawn_row, self.last_drawn_col) {
            (Some(r), Some(c)) => Some((r, c)),
            _ => None,
        }
    }

    fn next_frame(&mut self, target_row: u16, target_col: u16) -> AnimationCommand {
        self.counter = self.counter.saturating_add(1);

        if self.counter > self.cycle {
            self.last_drawn_row = None;
            self.last_drawn_col = None;
            AnimationCommand::Done
        } else {
            self.last_drawn_row = Some(target_row);
            self.last_drawn_col = Some(target_col);
            AnimationCommand::Draw {
                row: target_row,
                col: target_col,
                animation_type: self.get_animation_type(),
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_new_melee_animation() {
        let owner_id = 10;
        let animation = MeleeAnimation::new(owner_id);

        assert_eq!(animation.owner_id, owner_id);
        assert_eq!(animation.cycle, 2);
        assert_eq!(animation.counter, 0);
        assert!(animation.last_drawn_row.is_none());
        assert!(animation.last_drawn_col.is_none());
        assert!(animation.target_id.is_none());
    }

    #[test]
    fn test_melee_animation_trait_methods() {
        let owner_id = 20;
        let mut animation = MeleeAnimation::new(owner_id);

        // Test get_owner_id without target
        assert_eq!(animation.get_owner_id(), owner_id);

        // Test attach_target
        let target_id = 30;
        animation.attach_target(target_id);
        assert_eq!(animation.target_id, Some(target_id));

        // Test get_owner_id with target
        assert_eq!(animation.get_owner_id(), target_id);

        // Test get_last_drawn_pos (initially None)
        assert!(animation.get_last_drawn_pos().is_none());
    }

    #[test]
    fn test_melee_animation_next_frame_sequence() {
        let owner_id = 40;
        let mut animation = MeleeAnimation::new(owner_id);
        let target_row = 5;
        let target_col = 5;

        // Frame 1
        let command1 = animation.next_frame(target_row, target_col);
        match command1 {
            AnimationCommand::Draw {
                row,
                col,
                animation_type,
            } => {
                assert_eq!(row, target_row);
                assert_eq!(col, target_col);
                assert_eq!(animation_type, CellAnimation::MeleeHitOne);
                assert_eq!(
                    animation.get_last_drawn_pos(),
                    Some((target_row, target_col))
                );
            }
            _ => panic!("Expected Draw command for frame 1"),
        }

        // Frame 2
        let command2 = animation.next_frame(target_row, target_col);
        match command2 {
            AnimationCommand::Draw {
                row,
                col,
                animation_type,
            } => {
                assert_eq!(row, target_row);
                assert_eq!(col, target_col);
                assert_eq!(animation_type, CellAnimation::MeleeHitTwo);
                assert_eq!(
                    animation.get_last_drawn_pos(),
                    Some((target_row, target_col))
                );
            }
            _ => panic!("Expected Draw command for frame 2"),
        }

        // Frame 3 (Done)
        let command3 = animation.next_frame(target_row, target_col);
        assert_eq!(command3, AnimationCommand::Done);
        assert!(animation.get_last_drawn_pos().is_none());
    }
}

