use crate::game::{
    animation::{AnimationCommand, AnimationTrait},
    cell::{CellAnimation, Team},
};

#[derive(Debug, Clone)]
pub enum GameplayEffect {
    Damage(u16),
}

#[derive(Debug, Clone)]
pub struct Projectile {
    pub id: u64,
    pub team_id: Team,
    pub owner_id: u64,
    // Path and Movement
    pub path: Vec<(u16, u16)>,
    path_index: usize,
    // Timing
    speed: u32, // number of tick to move one cell
    tick_counter: u32,
    // Gameplay
    pub payload: GameplayEffect,
    // Rendering
    pub visual_cell_type: CellAnimation,
}

impl Projectile {
    // A basic constructor for initial testing.
    // Bresenham's path calculation will be integrated here later.
    pub fn new(
        id: u64,
        owner_id: u64,
        team_id: Team,
        start_pos: (u16, u16),
        end_pos: (u16, u16),
        speed: u32,
        payload: GameplayEffect,
        visual_cell_type: CellAnimation,
    ) -> Self {
        let path = vec![start_pos, end_pos];

        Projectile {
            id,
            owner_id,
            team_id,
            path,
            path_index: 0,
            speed,
            tick_counter: 0,
            payload,
            visual_cell_type,
        }
    }
}

impl AnimationTrait for Projectile {
    fn next_frame(&mut self, _owner_row: u16, _owner_col: u16) -> AnimationCommand {
        // If the projectile has reached the end of its path, it's done.
        if self.path_index >= self.path.len() {
            return AnimationCommand::Done;
        }

        // Get the position for the *current* frame before updating for the next.
        let (current_row, current_col) = self.path[self.path_index];

        // Update for the *next* frame
        self.tick_counter += 1;
        if self.tick_counter >= self.speed as u32 {
            self.tick_counter = 0;
            self.path_index += 1;
        }

        // Return the command to draw the projectile at its current position.
        AnimationCommand::Draw {
            row: current_row,
            col: current_col,
            animation_type: self.visual_cell_type.clone(),
        }
    }

    fn get_owner_id(&self) -> usize {
        self.owner_id as usize
    }

    fn attach_target(&mut self, target_id: crate::game::PlayerId) {
        todo!()
    }

    fn get_animation_type(&self) -> crate::game::cell::CellAnimation {
        self.visual_cell_type.clone()
    }

    fn get_last_drawn_pos(&self) -> Option<(u16, u16)> {
        if self.path_index > 0 {
            self.path.get(self.path_index - 1).copied()
        } else {
            None
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_projectile_creation() {
        let start_pos = (0, 0);
        let end_pos = (5, 5);
        let speed = 1;
        let payload = GameplayEffect::Damage(10);
        let visual_type = CellAnimation::Projectile;

        let projectile = Projectile::new(1, 100, Team::Red, start_pos, end_pos, speed, payload, visual_type);

        assert_eq!(projectile.id, 1);
        assert_eq!(projectile.owner_id, 100);
        assert_eq!(projectile.path.len(), 2); // Expecting the dummy path
        assert_eq!(projectile.path[0], start_pos);
        assert_eq!(projectile.path[1], end_pos);
        assert_eq!(projectile.path_index, 0);
        assert_eq!(projectile.speed, 1);
        assert_eq!(projectile.tick_counter, 0);
        match projectile.payload {
            GameplayEffect::Damage(amount) => assert_eq!(amount, 10),
            _ => panic!("Unexpected payload type"),
        }
        assert_eq!(projectile.visual_cell_type, CellAnimation::Projectile);
    }

    // Test for Animation trait implementation
    // Note: The AnimationTrait needs to be imported or defined for this test to compile.
    // Assuming it will be imported from super::super::animation::AnimationTrait
    use super::super::super::animation::AnimationTrait;

    #[test]
    fn test_projectile_movement_and_finish() {
        let start_pos = (0, 0);
        let end_pos = (2, 0); // Moving horizontally
        let speed = 1; // Moves every tick
        let payload = GameplayEffect::Damage(10);
        let visual_type = CellAnimation::Projectile;

        let mut projectile =
            Projectile::new(1, 100, Team::Red, start_pos, end_pos, speed, payload, visual_type.clone());

        // Initial state
        assert_eq!(projectile.get_last_drawn_pos(), None); // No previous position
        assert_eq!(projectile.path_index, 0);

        // Tick 1: Should move to the first position in the path (start_pos)
        let command1 = projectile.next_frame(0, 0); // owner_row, owner_col don't matter for projectile
        assert!(
            matches!(command1, AnimationCommand::Draw { row, col, animation_type } if row == start_pos.0 && col == start_pos.1 && animation_type == visual_type)
        );
        assert_eq!(projectile.get_last_drawn_pos(), Some(start_pos));
        assert_eq!(projectile.path_index, 1);

        // Tick 2: Should move to the second position in the path (end_pos)
        let command2 = projectile.next_frame(0, 0);
        assert!(
            matches!(command2, AnimationCommand::Draw { row, col, animation_type } if row == end_pos.0 && col == end_pos.1 && animation_type == visual_type)
        );
        assert_eq!(projectile.get_last_drawn_pos(), Some(end_pos));
        assert_eq!(projectile.path_index, 2);

        // Tick 3: Should be finished
        let command3 = projectile.next_frame(0, 0);
        assert!(matches!(command3, AnimationCommand::Done));
        assert_eq!(projectile.get_last_drawn_pos(), Some(end_pos)); // Last drawn pos remains the same
    }

    #[test]
    fn test_projectile_movement_with_speed_delay() {
        let start_pos = (0, 0);
        let end_pos = (1, 0);
        let speed = 2; // Moves every 2 ticks
        let payload = GameplayEffect::Damage(10);
        let visual_type = CellAnimation::Projectile;

        let mut projectile =
            Projectile::new(1, 100, Team::Red, start_pos, end_pos, speed, payload, visual_type.clone());

        // Initial state
        assert_eq!(projectile.get_last_drawn_pos(), None);
        assert_eq!(projectile.path_index, 0);

        // Tick 1: Should draw at start_pos, path_index remains 0
        let command1 = projectile.next_frame(0, 0);
        assert!(
            matches!(command1, AnimationCommand::Draw { row, col, animation_type } if row == start_pos.0 && col == start_pos.1 && animation_type == visual_type)
        );
        assert_eq!(projectile.get_last_drawn_pos(), None);
        assert_eq!(projectile.path_index, 0);

        // Tick 2: Should draw at start_pos again, path_index advances to 1
        let command2 = projectile.next_frame(0, 0);
        assert!(
            matches!(command2, AnimationCommand::Draw { row, col, animation_type } if row == start_pos.0 && col == start_pos.1 && animation_type == visual_type)
        );
        assert_eq!(projectile.get_last_drawn_pos(), Some(start_pos));
        assert_eq!(projectile.path_index, 1);

        // Tick 3: Should draw at end_pos, path_index remains 1
        let command3 = projectile.next_frame(0, 0);
        assert!(
            matches!(command3, AnimationCommand::Draw { row, col, animation_type } if row == end_pos.0 && col == end_pos.1 && animation_type == visual_type)
        );
        assert_eq!(projectile.get_last_drawn_pos(), Some(start_pos));
        assert_eq!(projectile.path_index, 1);

        // Tick 4: Should draw at end_pos again, path_index advances to 2
        let command4 = projectile.next_frame(0, 0);
        assert!(
            matches!(command4, AnimationCommand::Draw { row, col, animation_type } if row == end_pos.0 && col == end_pos.1 && animation_type == visual_type)
        );
        assert_eq!(projectile.get_last_drawn_pos(), Some(end_pos));
        assert_eq!(projectile.path_index, 2);

        // Tick 5: Should be finished
        let command5 = projectile.next_frame(0, 0);
        assert!(matches!(command5, AnimationCommand::Done));
        assert_eq!(projectile.get_last_drawn_pos(), Some(end_pos));
        assert_eq!(projectile.path_index, 2);
    }
}
