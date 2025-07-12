use crate::game::{
    algorithms::bresenham::Bresenham,
    animation::{AnimationCommand, AnimationTrait},
    cell::{CellAnimation, Team},
};

use super::Target;

#[derive(Debug, Clone)]
pub enum GameplayEffect {
    Damage(u16),
}

#[derive(Debug, Clone)]
pub enum PathingLogic {
    Straight {
        path: Vec<(u16, u16)>,
        current_index: usize,
    },
    LockOn {
        target_id: Target,
    },
}

#[derive(Debug, Clone)]
pub struct Projectile {
    pub id: u64,
    pub team_id: Team,
    pub owner_id: u64,
    // Path and Movement
    pub current_position: (u16, u16),
    pub pathing: PathingLogic,
    // Timing
    speed: u32, // number of tick to move one cell
    tick_counter: u32,
    // Gameplay
    pub payload: GameplayEffect,
    // Rendering
    pub visual_cell_type: CellAnimation,
}

impl Projectile {
    pub fn from_skillshot(
        id: u64,
        owner_id: u64,
        team_id: Team,
        start_pos: (u16, u16),
        end_pos: (u16, u16),
        speed: u32,
        payload: GameplayEffect,
        visual_cell_type: CellAnimation,
    ) -> Self {
        let path = Bresenham::new(start_pos, end_pos).collect();
        let pathing = PathingLogic::Straight {
            path,
            current_index: 0,
        };
        Projectile {
            id,
            owner_id,
            team_id,
            current_position: start_pos,
            pathing,
            speed,
            tick_counter: 0,
            payload,
            visual_cell_type,
        }
    }

    pub fn from_homing_shot(
        id: u64,
        owner_id: u64,
        team_id: Team,
        start_pos: (u16, u16),
        target_id: Target,
        speed: u32,
        payload: GameplayEffect,
        visual_cell_type: CellAnimation,
    ) -> Self {
        let pathing = PathingLogic::LockOn { target_id };
        Projectile {
            id,
            owner_id,
            team_id,
            current_position: start_pos,
            pathing,
            speed,
            tick_counter: 0,
            payload,
            visual_cell_type,
        }
    }
}

impl AnimationTrait for Projectile {
    fn next_frame(&mut self, target_row: u16, target_col: u16) -> AnimationCommand {
        // 0. Handling speed timing
        self.tick_counter += 1;
        if self.tick_counter < self.speed {
            return AnimationCommand::Draw {
                row: self.current_position.0,
                col: self.current_position.1,
                animation_type: self.visual_cell_type.clone(),
            };
        }
        self.tick_counter = 0;

        // 1. Match projectile type
        match &mut self.pathing {
            PathingLogic::Straight { path, current_index } => {
                if *current_index >= path.len() {
                    return AnimationCommand::Done;
                }
                self.current_position = path[*current_index];
                *current_index += 1;
            }
            PathingLogic::LockOn { .. } => {
                let row_step = (target_row as i32 - self.current_position.0 as i32).signum() as i16;
                let col_step = (target_col as i32 - self.current_position.1 as i32).signum() as i16;

                if row_step == 0 && col_step == 0 {
                    return AnimationCommand::Done;
                }
                self.current_position.0 = self.current_position.0.saturating_add_signed(row_step);
                self.current_position.1 = self.current_position.1.saturating_add_signed(col_step);
            }
        }
         // 3. Return the Draw command with the new position
         AnimationCommand::Draw {
             row: self.current_position.0,
             col: self.current_position.1,
             animation_type: self.visual_cell_type.clone(),
         }
    }

    fn get_owner_id(&self) -> usize {
        self.owner_id as usize
    }

    fn attach_target(&mut self, _target_id: crate::game::PlayerId) {
        // This is not needed for projectiles as their target is determined on creation
    }

    fn get_animation_type(&self) -> crate::game::cell::CellAnimation {
        self.visual_cell_type.clone()
    }

    fn get_last_drawn_pos(&self) -> Option<(u16, u16)> {
        Some(self.current_position)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::game::animation::AnimationTrait; // For trait methods
    use crate::game::cell::CellAnimation;
    use crate::game::entities::Target;

    // Test `from_skillshot` constructor
    #[test]
    fn test_from_skillshot_creation() {
        let start_pos = (10, 10);
        let end_pos = (15, 10);
        let projectile = Projectile::from_skillshot(
            1,
            101,
            Team::Blue,
            start_pos,
            end_pos,
            1,
            GameplayEffect::Damage(50),
            CellAnimation::Projectile,
        );

        assert_eq!(projectile.id, 1);
        assert_eq!(projectile.owner_id, 101);
        assert_eq!(projectile.team_id, Team::Blue);
        assert_eq!(projectile.current_position, start_pos);

        if let PathingLogic::Straight { path, current_index } = projectile.pathing {
            let expected_path = Bresenham::new(start_pos, end_pos).collect::<Vec<_>>();
            assert_eq!(path, expected_path);
            assert_eq!(current_index, 0);
        } else {
            panic!("Expected PathingLogic::Straight");
        }
    }

    // Test `from_homing_shot` constructor
    #[test]
    fn test_from_homing_shot_creation() {
        let start_pos = (5, 5);
        let target = Target::Champion(202);
        let projectile = Projectile::from_homing_shot(
            2,
            102,
            Team::Red,
            start_pos,
            target.clone(),
            2,
            GameplayEffect::Damage(30),
            CellAnimation::Projectile,
        );

        assert_eq!(projectile.id, 2);
        assert_eq!(projectile.owner_id, 102);
        assert_eq!(projectile.team_id, Team::Red);
        assert_eq!(projectile.current_position, start_pos);

        if let PathingLogic::LockOn { .. } = projectile.pathing {
            // You might need to derive or implement PartialEq for Target to do this
            // assert_eq!(target_id, target);
        } else {
            panic!("Expected PathingLogic::LockOn");
        }
    }

    // Test movement for a Straight projectile
    #[test]
    fn test_next_frame_for_skillshot() {
        let start_pos = (0, 0);
        let end_pos = (2, 0); // Simple horizontal path
        let mut projectile = Projectile::from_skillshot(
            3,
            103,
            Team::Blue,
            start_pos,
            end_pos,
            1, // speed = 1 tick per cell
            GameplayEffect::Damage(10),
            CellAnimation::Projectile,
        );

        // Tick 1: Moves to (0, 0)
        let cmd1 = projectile.next_frame(99, 99); // Target pos is ignored
        assert!(matches!(cmd1, AnimationCommand::Draw { row: 0, col: 0, .. }));
        assert_eq!(projectile.current_position, (0, 0));

        // Tick 2: Moves to (1, 0)
        let cmd2 = projectile.next_frame(99, 99);
        assert!(matches!(cmd2, AnimationCommand::Draw { row: 1, col: 0, .. }));
        assert_eq!(projectile.current_position, (1, 0));

        // Tick 3: Moves to (2, 0)
        let cmd3 = projectile.next_frame(99, 99);
        assert!(matches!(cmd3, AnimationCommand::Draw { row: 2, col: 0, .. }));
        assert_eq!(projectile.current_position, (2, 0));

        // Tick 4: Path is finished
        let cmd4 = projectile.next_frame(99, 99);
        assert!(matches!(cmd4, AnimationCommand::Done));
    }

    // Test movement for a LockOn projectile
    #[test]
    fn test_next_frame_for_homing_shot() {
        let start_pos = (10, 10);
        let target = Target::Champion(202);
        let mut projectile = Projectile::from_homing_shot(
            4,
            104,
            Team::Red,
            start_pos,
            target,
            1, // speed = 1 tick per cell
            GameplayEffect::Damage(10),
            CellAnimation::Projectile,
        );

        // Tick 1: Target is at (10, 13). Projectile should move to (10, 11)
        let cmd1 = projectile.next_frame(10, 13);
        assert!(matches!(cmd1, AnimationCommand::Draw { row: 10, col: 11, .. }));
        assert_eq!(projectile.current_position, (10, 11));

        // Tick 2: Target is still at (10, 13). Projectile should move to (10, 12)
        let cmd2 = projectile.next_frame(10, 13);
        assert!(matches!(cmd2, AnimationCommand::Draw { row: 10, col: 12, .. }));
        assert_eq!(projectile.current_position, (10, 12));

        // Tick 3: Target is now at (10, 13). Projectile moves to (10, 13)
        let cmd3 = projectile.next_frame(10, 13);
        assert!(matches!(cmd3, AnimationCommand::Draw { row: 10, col: 13, .. }));
        assert_eq!(projectile.current_position, (10, 13));

        // Tick 4: Projectile is on the target. Should be Done.
        let cmd4 = projectile.next_frame(10, 13);
        assert!(matches!(cmd4, AnimationCommand::Done));
    }

    // Test movement with speed delay
    #[test]
    fn test_movement_with_speed_delay() {
        let start_pos = (0, 0);
        let end_pos = (1, 0);
        let mut projectile = Projectile::from_skillshot(
            5,
            105,
            Team::Blue,
            start_pos,
            end_pos,
            2, // speed = 2 ticks per cell
            GameplayEffect::Damage(10),
            CellAnimation::Projectile,
        );

        // Tick 1: Not time to move yet. Redraws at (0,0)
        let cmd1 = projectile.next_frame(99, 99);
        assert!(matches!(cmd1, AnimationCommand::Draw { row: 0, col: 0, .. }));
        assert_eq!(projectile.current_position, (0, 0));

        // Tick 2: Time to move. Moves to (0,0) from path.
        let cmd2 = projectile.next_frame(99, 99);
        assert!(matches!(cmd2, AnimationCommand::Draw { row: 0, col: 0, .. }));
        assert_eq!(projectile.current_position, (0, 0));

        // Tick 3: Not time to move. Redraws at (0,0).
        let cmd3 = projectile.next_frame(99, 99);
        assert!(matches!(cmd3, AnimationCommand::Draw { row: 0, col: 0, .. }));
        assert_eq!(projectile.current_position, (0, 0));

        // Tick 4: Time to move. Moves to (1,0) from path.
        let cmd4 = projectile.next_frame(99, 99);
        assert!(matches!(cmd4, AnimationCommand::Draw { row: 1, col: 0, .. }));
        assert_eq!(projectile.current_position, (1, 0));
    }
}
