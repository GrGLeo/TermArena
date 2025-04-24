use std::{
    collections::VecDeque,
    time::{Duration, Instant},
};
use strum_macros::EnumIter;

use crate::{
    errors::GameError,
    game::{
        Board, Cell, CellContent, MinionId,
        animation::{AnimationTrait, melee::MeleeAnimation},
        cell::Team,
        pathfinding::{find_path_on_board, is_adjacent_to_goal},
    },
};

use super::{Fighter, Stats, Target, reduced_damage};

type MinionPath = (u16, u16);

#[derive(Debug, PartialEq, Eq, EnumIter)]
pub enum Lane {
    Top,
    Mid,
    Bottom,
}

#[derive(Debug)]
pub struct Minion {
    pub minion_id: MinionId,
    pub team_id: Team,
    lane: Lane,
    path: Option<VecDeque<(u16, u16)>>,
    stats: Stats,
    current_path: MinionPath,
    minion_path: Vec<MinionPath>,
    checkpoint: usize,
    last_attacked: Instant,
    pub row: u16,
    pub col: u16,
}

impl Minion {
    pub fn new(minion_id: MinionId, team_id: Team, lane: Lane) -> Self {
        let stats = Stats {
            attack_damage: 6,
            attack_speed: Duration::from_millis(2500),
            health: 40,
            max_health: 40,
            armor: 0,
        };

        let (row, col, paths) = match team_id {
            Team::Blue => match lane {
                Lane::Top => (
                    184,
                    10,
                    vec![(120, 8), (39, 7), (7, 39), (8, 120), (10, 184)],
                ),
                Lane::Mid => (
                    184,
                    17,
                    vec![(148, 67), (115, 82), (82, 115), (67, 148), (17, 184)],
                ),
                Lane::Bottom => (
                    191,
                    17,
                    vec![(191, 79), (196, 150), (150, 196), (79, 191), (17, 191)],
                ),
            },
            Team::Red => match lane {
                Lane::Top => (
                    10,
                    184,
                    vec![(8, 120), (7, 39), (39, 7), (120, 8), (184, 10)],
                ),
                Lane::Mid => (
                    17,
                    184,
                    vec![(67, 148), (82, 115), (115, 82), (148, 67), (184, 17)],
                ),
                Lane::Bottom => (
                    17,
                    191,
                    vec![(79, 191), (150, 196), (196, 150), (191, 79), (191, 17)],
                ),
            },
        };
        let path = paths[0];

        Self {
            minion_id,
            team_id,
            lane,
            path: None,
            stats,
            current_path: path,
            minion_path: paths,
            checkpoint: 0,
            last_attacked: Instant::now(),
            row,
            col,
        }
    }

    fn change_goal(&mut self) {
        if self.checkpoint < self.minion_path.len() {
            self.checkpoint += 1;
            self.current_path = self.minion_path[self.checkpoint as usize];
        }
    }

    pub fn is_dead(&self) -> bool {
        if self.stats.health <= 0 { true } else { false }
    }

    pub fn movement_phase(&mut self, board: &mut Board) -> Result<(), GameError> {
        if is_adjacent_to_goal((self.row, self.col), self.current_path) {
            self.change_goal();
        }
        if let Some(mut path) = self.path.take() {
            if let Some(next_step) = path.pop_front() {
                let row_step = (next_step.0 as i16 - self.row as i16).signum() as isize;
                let col_step = (next_step.1 as i16 - self.col as i16).signum() as isize;
                match self.move_minion(board, row_step, col_step) {
                    Ok(_) => {
                        if !path.is_empty() {
                            self.path = Some(path);
                        }
                        return Ok(());
                    }
                    Err(_) => self.path = None,
                }
            } else {
                self.path = None;
            }
        }
        // scan aggro range 10*10 aggro range for now
        // and move toward closest target
        let target_pos = if let Some(cell) = self.get_potential_target(board, (10, 10)) {
            cell.position
        } else {
            self.current_path
        };
        // If already adjacent to the cell we don't need to move
        if is_adjacent_to_goal((self.row, self.col), target_pos) {
            return Ok(());
        }
        // else simply move one step toward current goal
        let row_step = (target_pos.0 as i16 - self.row as i16).signum() as isize;
        let col_step = (target_pos.1 as i16 - self.col as i16).signum() as isize;
        match self.move_minion(board, row_step, col_step) {
            Ok(_) => return Ok(()),
            Err(_) => {
                if let Some(calculated_path) =
                    find_path_on_board(board, (self.row, self.col), target_pos)
                {
                    self.path = Some(calculated_path);
                    return Ok(());
                } else {
                    return Err(GameError::CannotMoveHere(self.minion_id));
                }
            }
        }
    }

    pub fn attack_phase(
        &mut self,
        board: &mut Board,
        new_animations: &mut Vec<Box<dyn AnimationTrait>>,
        pending_damages: &mut Vec<(Target, u16)>,
    ) {
        if let Some(enemy) = self.get_potential_target(board, (3, 3)) {
            println!("Minion found target in melee range: {:?}", enemy);
            match &enemy.content {
                Some(content) => match content {
                    CellContent::Tower(id, _) => {
                        if let Some((damage, animation)) = self.can_attack() {
                            println!("Raw damage: {}", damage);
                            new_animations.push(animation);
                            pending_damages.push((Target::Tower(*id), damage))
                        }
                    }
                    CellContent::Minion(id, _) => {
                        if let Some((damage, animation)) = self.can_attack() {
                            new_animations.push(animation);
                            pending_damages.push((Target::Minion(*id), damage))
                        }
                    }
                    CellContent::Champion(id, _) => {
                        if let Some((damage, animation)) = self.can_attack() {
                            new_animations.push(animation);
                            pending_damages.push((Target::Champion(*id), damage))
                        }
                    }
                    _ => return,
                },
                None => return,
            }
        }
    }

    fn move_minion(
        &mut self,
        board: &mut Board,
        row_step: isize,
        col_step: isize,
    ) -> Result<(), GameError> {
        let new_row = if row_step < 0 {
            self.row.saturating_sub(row_step.unsigned_abs() as u16)
        } else {
            self.row.saturating_add(row_step as u16)
        };

        let new_col = if col_step < 0 {
            self.col.saturating_sub(col_step.unsigned_abs() as u16)
        } else {
            self.col.saturating_add(col_step as u16)
        };

        if new_row >= board.rows as u16 || new_col >= board.cols as u16 {
            return Err(GameError::CannotMoveHere(self.minion_id));
        }

        if let Some(new_cell) = board.get_cell(new_row as usize, new_col as usize) {
            if new_cell.is_passable() {
                board.move_cell(
                    self.row as usize,
                    self.col as usize,
                    new_row as usize,
                    new_col as usize,
                );
                self.row = new_row;
                self.col = new_col;
                Ok(())
            } else {
                return Err(GameError::CannotMoveHere(self.minion_id));
            }
        } else {
            return Err(GameError::NotFoundCell);
        }
    }
}

impl Fighter for Minion {
    fn take_damage(&mut self, damage: u16) {
        let reduced_damage = reduced_damage(damage, self.stats.armor);
        self.stats.health = self.stats.health.saturating_sub(reduced_damage as u16);
    }

    fn can_attack(&mut self) -> Option<(u16, Box<dyn AnimationTrait>)> {
        if self.last_attacked + self.stats.attack_speed < Instant::now() {
            self.last_attacked = Instant::now();
            let animation = MeleeAnimation::new(self.minion_id);
            Some((self.stats.attack_damage, Box::new(animation)))
        } else {
            None
        }
    }

    fn get_potential_target<'a>(&self, board: &'a Board, range: (u16, u16)) -> Option<&'a Cell> {
        let (row_range, col_range) = range;
        let target_area = board.center_view(self.row, self.col, row_range, col_range);
        let center_row = target_area.len() / 2;
        let center_col = target_area[0].len() / 2;

        target_area
            .iter()
            .enumerate()
            .flat_map(|(row_index, row)| {
                row.iter()
                    .enumerate()
                    .map(move |(col_index, cell)| (row_index, col_index, cell))
            })
            .filter_map(|(row, col, cell)| {
                if let Some(content) = &cell.content {
                    match content {
                        CellContent::Champion(_, team_id)
                        | CellContent::Tower(_, team_id)
                        | CellContent::Minion(_, team_id) => {
                            if *team_id != self.team_id {
                                Some((row, col, cell))
                            } else {
                                None
                            }
                        }
                        _ => None,
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::game::{
        Board, MinionId,
        cell::Team,
        cell::{BaseTerrain, CellContent},
    };

    fn create_dummy_board(rows: usize, cols: usize) -> Board {
        Board::new(rows, cols)
    }

    fn manhattan_distance(r1: u16, c1: u16, r2: u16, c2: u16) -> u16 {
        r1.abs_diff(r2) + c1.abs_diff(c2)
    }

    fn calculate_expected_next_pos(
        current_row: u16,
        current_col: u16,
        goal_row: u16,
        goal_col: u16,
    ) -> (u16, u16) {
        let row_diff = goal_row as i16 - current_row as i16;
        let col_diff = goal_col as i16 - current_col as i16;

        let next_row = current_row.saturating_add_signed(row_diff.signum());
        let next_col = current_col.saturating_add_signed(col_diff.signum());
        (next_row, next_col)
    }

    #[test]
    fn test_new_minion() {
        let minion_id = 1;
        let stats = Stats {
            attack_damage: 6,
            attack_speed: Duration::from_millis(2500),
            health: 40,
            max_health: 40,
            armor: 8,
        };

        // Test Blue Team Minions
        let blue_top_minion = Minion::new(minion_id, Team::Blue, Lane::Top);
        assert_eq!(blue_top_minion.minion_id, minion_id);
        assert_eq!(blue_top_minion.team_id, Team::Blue);
        assert_eq!(blue_top_minion.lane, Lane::Top);
        assert_eq!(blue_top_minion.stats.health, stats.health); // Check a few stats fields
        assert_eq!(blue_top_minion.row, 184);
        assert_eq!(blue_top_minion.col, 10);
        assert_eq!(blue_top_minion.current_path, (120, 8));

        // Test Red Team Minions
        let red_top_minion = Minion::new(minion_id, Team::Red, Lane::Top);
        assert_eq!(red_top_minion.minion_id, minion_id);
        assert_eq!(red_top_minion.team_id, Team::Red);
        assert_eq!(red_top_minion.lane, Lane::Top);
        assert_eq!(red_top_minion.stats.health, stats.health);
        assert_eq!(red_top_minion.row, 10);
        assert_eq!(red_top_minion.col, 184);
        assert_eq!(red_top_minion.current_path, (8, 120));
    }

    #[test]
    fn test_move_minion_to_passable_cell() {
        let mut board = create_dummy_board(10, 10);
        let minion_id: MinionId = 1;
        let team_id = Team::Blue;
        let initial_row = 5;
        let initial_col = 5;

        // Create a minion and place it on the board
        let mut minion = Minion::new(minion_id, team_id, Lane::Mid);
        minion.row = initial_row; // Set initial position manually for testing
        minion.col = initial_col;
        let minion_content = CellContent::Minion(minion_id, team_id);
        board.place_cell(
            minion_content.clone(),
            initial_row as usize,
            initial_col as usize,
        );

        // Test moving right (d_row = 0, d_col = 1)
        let d_row: isize = 0;
        let d_col: isize = 1;
        let expected_new_row = initial_row;
        let expected_new_col = initial_col + 1;

        let move_result = minion.move_minion(&mut board, d_row, d_col);

        assert!(
            move_result.is_ok(),
            "Moving to a passable cell should succeed"
        );
        assert_eq!(
            minion.row, expected_new_row,
            "Minion row should update after moving"
        );
        assert_eq!(
            minion.col, expected_new_col,
            "Minion col should update after moving"
        );

        // Verify board state: old cell empty, new cell has minion content
        let old_cell = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Old cell should exist");
        assert!(
            old_cell.content.is_none(),
            "Old cell should be empty after minion moves"
        );
        let new_cell = board
            .get_cell(expected_new_row as usize, expected_new_col as usize)
            .expect("New cell should exist");
        assert_eq!(
            new_cell.content,
            Some(minion_content.clone()),
            "New cell should have minion content after moving"
        );

        // Test moving down (d_row = 1, d_col = 0) - Reset position first
        minion.row = initial_row;
        minion.col = initial_col;
        board.place_cell(
            minion_content.clone(),
            initial_row as usize,
            initial_col as usize,
        ); // Re-place minion
        board.clear_cell(expected_new_row as usize, expected_new_col as usize); // Clear previous spot

        let d_row_down: isize = 1;
        let d_col_down: isize = 0;
        let expected_new_row_down = initial_row + 1;
        let expected_new_col_down = initial_col;

        let move_result_down = minion.move_minion(&mut board, d_row_down, d_col_down);
        assert!(
            move_result_down.is_ok(),
            "Moving down to a passable cell should succeed"
        );
        assert_eq!(
            minion.row, expected_new_row_down,
            "Minion row should update after moving down"
        );
        assert_eq!(
            minion.col, expected_new_col_down,
            "Minion col should update after moving down"
        );
        // Verify board state
        let old_cell_down = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Old cell should exist");
        assert!(
            old_cell_down.content.is_none(),
            "Old cell should be empty after minion moves down"
        );
        let new_cell_down = board
            .get_cell(
                expected_new_row_down as usize,
                expected_new_col_down as usize,
            )
            .expect("New cell should exist");
        assert_eq!(
            new_cell_down.content,
            Some(minion_content.clone()),
            "New cell should have minion content after moving down"
        );

        // Add tests for other directions (left, up, and diagonals if minion can move diagonally) similarly
        // Based on the code, it handles d_row and d_col independently, so it supports diagonal movement.
        // Test moving up-left (d_row = -1, d_col = -1) - Reset position first
        minion.row = initial_row;
        minion.col = initial_col;
        board.place_cell(
            minion_content.clone(),
            initial_row as usize,
            initial_col as usize,
        ); // Re-place minion
        board.clear_cell(
            expected_new_row_down as usize,
            expected_new_col_down as usize,
        ); // Clear previous spot

        let d_row_upleft: isize = -1;
        let d_col_upleft: isize = -1;
        let expected_new_row_upleft = initial_row - 1;
        let expected_new_col_upleft = initial_col - 1;

        let move_result_upleft = minion.move_minion(&mut board, d_row_upleft, d_col_upleft);
        assert!(
            move_result_upleft.is_ok(),
            "Moving up-left to a passable cell should succeed"
        );
        assert_eq!(
            minion.row, expected_new_row_upleft,
            "Minion row should update after moving up-left"
        );
        assert_eq!(
            minion.col, expected_new_col_upleft,
            "Minion col should update after moving up-left"
        );
        // Verify board state
        let old_cell_upleft = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Old cell should exist");
        assert!(
            old_cell_upleft.content.is_none(),
            "Old cell should be empty after minion moves up-left"
        );
        let new_cell_upleft = board
            .get_cell(
                expected_new_row_upleft as usize,
                expected_new_col_upleft as usize,
            )
            .expect("New cell should exist");
        assert_eq!(
            new_cell_upleft.content,
            Some(minion_content),
            "New cell should have minion content after moving up-left"
        );
    }

    #[test]
    fn test_minion_change_first_goal() {
        let minion_id: MinionId = 1;
        let team_id = Team::Blue;
        let initial_row = 191;
        let initial_col = 179;

        let mut minion = Minion::new(minion_id, team_id, Lane::Bottom);
        minion.row = initial_row;
        minion.col = initial_col;
        minion.change_goal();
        let expected_minion_path = (196, 150);
        assert_eq!(
            minion.current_path, expected_minion_path,
            "Incorrect goal was set"
        )
    }

    #[test]
    fn test_minion_change_later_goal() {
        let minion_id: MinionId = 1;
        let team_id = Team::Red;
        let initial_row = 39;
        let initial_col = 7;

        let mut minion = Minion::new(minion_id, team_id, Lane::Top);
        minion.checkpoint = 2;
        minion.row = initial_row;
        minion.col = initial_col;
        minion.change_goal();
        let expected_minion_path = (120, 8);
        assert_eq!(
            minion.current_path, expected_minion_path,
            "Incorrect goal was set"
        )
    }

    #[test]
    fn test_move_minion_out_of_bounds() {
        let mut board = create_dummy_board(10, 10); // 0-9 rows, 0-9 cols
        let minion_id: MinionId = 1;
        let team_id = Team::Blue;
        let initial_row = 0; // Place minion at top-left edge
        let initial_col = 0;

        let mut minion = Minion::new(minion_id, team_id, Lane::Mid);
        minion.row = initial_row;
        minion.col = initial_col;
        let minion_content = CellContent::Minion(minion_id, team_id);
        board.place_cell(
            minion_content.clone(),
            initial_row as usize,
            initial_col as usize,
        );

        // Test moving up from row 0 (out of bounds)
        let d_row: isize = -1;
        let d_col: isize = 0;
        let move_result = minion.move_minion(&mut board, d_row, d_col);

        println!("{:?}", move_result);
        assert!(
            move_result.is_err(),
            "Moving out of bounds should return an error"
        );
        assert_eq!(
            move_result.unwrap_err(),
            GameError::CannotMoveHere(minion_id),
            "Error should be CannotMoveHere"
        );
        assert_eq!(
            minion.row, initial_row,
            "Minion row should not change after failed move"
        );
        assert_eq!(
            minion.col, initial_col,
            "Minion col should not change after failed move"
        );
        // Verify board state: minion should still be in the original cell
        let initial_cell = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Initial cell should exist");
        assert_eq!(
            initial_cell.content,
            Some(minion_content.clone()),
            "Minion should remain in the initial cell after failed move"
        );

        // Test moving left from col 0 (out of bounds)
        let d_row_left: isize = 0;
        let d_col_left: isize = -1;
        let move_result_left = minion.move_minion(&mut board, d_row_left, d_col_left);

        assert!(
            move_result_left.is_err(),
            "Moving out of bounds should return an error"
        );
        assert_eq!(
            move_result_left.unwrap_err(),
            GameError::CannotMoveHere(minion_id),
            "Error should be CannotMoveHere"
        );
        assert_eq!(
            minion.row, initial_row,
            "Minion row should not change after failed move"
        );
        assert_eq!(
            minion.col, initial_col,
            "Minion col should not change after failed move"
        );
        // Verify board state remains unchanged
        let initial_cell_left = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Initial cell should exist");
        assert_eq!(
            initial_cell_left.content,
            Some(minion_content),
            "Minion should remain in the initial cell after failed move"
        );

        // Add tests for moving out of bounds from other edges/corners similarly...
        // Test moving down from row 9
        let mut minion_bottom = Minion::new(minion_id + 1, team_id, Lane::Mid);
        let initial_row_bottom = 9;
        let initial_col_bottom = 5;
        minion_bottom.row = initial_row_bottom;
        minion_bottom.col = initial_col_bottom;
        let minion_content_bottom = CellContent::Minion(minion_id + 1, team_id);
        board.place_cell(
            minion_content_bottom.clone(),
            initial_row_bottom as usize,
            initial_col_bottom as usize,
        );

        let d_row_down: isize = 1;
        let d_col_down: isize = 0;
        let move_result_down = minion_bottom.move_minion(&mut board, d_row_down, d_col_down);

        assert!(
            move_result_down.is_err(),
            "Moving out of bounds should return an error"
        );
        assert_eq!(
            move_result_down.unwrap_err(),
            GameError::CannotMoveHere(minion_bottom.minion_id),
            "Error should be CannotMoveHere"
        );
        assert_eq!(
            minion_bottom.row, initial_row_bottom,
            "Minion row should not change after failed move"
        );
        assert_eq!(
            minion_bottom.col, initial_col_bottom,
            "Minion col should not change after failed move"
        );
    }

    #[test]
    fn test_move_minion_into_impassable_cell() {
        let mut board = create_dummy_board(10, 10);
        let minion_id: MinionId = 1;
        let team_id = Team::Blue;
        let initial_row = 5;
        let initial_col = 5;

        let mut minion = Minion::new(minion_id, team_id, Lane::Mid);
        minion.row = initial_row;
        minion.col = initial_col;
        let minion_content = CellContent::Minion(minion_id, team_id);
        board.place_cell(
            minion_content.clone(),
            initial_row as usize,
            initial_col as usize,
        );

        // Test moving into a Wall cell
        let wall_row = initial_row;
        let wall_col = initial_col + 1;
        board.change_base(BaseTerrain::Wall, wall_row as usize, wall_col as usize);

        let d_row_wall: isize = 0;
        let d_col_wall: isize = 1;
        let move_result_wall = minion.move_minion(&mut board, d_row_wall, d_col_wall);

        // Based on your code, moving into an impassable cell returns GameError::NotFoundCell
        assert!(
            move_result_wall.is_err(),
            "Moving into a wall should return an error"
        );
        assert_eq!(
            move_result_wall.unwrap_err(),
            GameError::CannotMoveHere(minion_id),
            "Error should be CannotMoveHere for impassable cell"
        ); // Note: Error type from code
        assert_eq!(
            minion.row, initial_row,
            "Minion row should not change after failed move"
        );
        assert_eq!(
            minion.col, initial_col,
            "Minion col should not change after failed move"
        );
        // Verify board state remains unchanged
        let initial_cell_wall = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Initial cell should exist");
        assert_eq!(
            initial_cell_wall.content,
            Some(minion_content.clone()),
            "Minion should remain in the initial cell after failed move into wall"
        );
        let wall_cell = board
            .get_cell(wall_row as usize, wall_col as usize)
            .expect("Wall cell should exist");
        assert_eq!(
            wall_cell.base,
            BaseTerrain::Wall,
            "Wall cell should remain a Wall"
        );

        // Test moving into a cell with other content
        board.change_base(BaseTerrain::Floor, wall_row as usize, wall_col as usize); // Reset base
        let other_content_row = initial_row + 1;
        let other_content_col = initial_col;
        board.place_cell(
            CellContent::Champion(99, Team::Red),
            other_content_row as usize,
            other_content_col as usize,
        ); // Place other content

        let d_row_content: isize = 1;
        let d_col_content: isize = 0;
        let move_result_content = minion.move_minion(&mut board, d_row_content, d_col_content);

        // Based on your code, moving into an impassable cell returns GameError::NotFoundCell
        assert!(
            move_result_content.is_err(),
            "Moving into a cell with content should return an error"
        );
        assert_eq!(
            move_result_content.unwrap_err(),
            GameError::CannotMoveHere(minion_id),
            "Error should be CannotMoveHere for cell with content"
        ); // Note: Error type from code
        assert_eq!(
            minion.row, initial_row,
            "Minion row should not change after failed move"
        );
        assert_eq!(
            minion.col, initial_col,
            "Minion col should not change after failed move"
        );
        // Verify board state remains unchanged
        let initial_cell_content = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Initial cell should exist");
        assert_eq!(
            initial_cell_content.content,
            Some(minion_content),
            "Minion should remain in the initial cell after failed move into content"
        );
        let content_cell = board
            .get_cell(other_content_row as usize, other_content_col as usize)
            .expect("Content cell should exist");
        assert!(
            content_cell.content.is_some(),
            "Content cell should still have content"
        );
    }

    #[test]
    fn test_minion_turn_move_towards_goal() {
        let minion_id: MinionId = 1;
        let team_id = Team::Blue;
        let minion_content = CellContent::Minion(minion_id, team_id);

        // --- Test Case 1: Move Down-Right ---
        let mut board1 = create_dummy_board(200, 200);
        let initial_row1 = 10;
        let initial_col1 = 10;
        let goal_row1 = 20;
        let goal_col1 = 20;

        let mut minion1 = Minion::new(minion_id, team_id, Lane::Mid);
        minion1.row = initial_row1;
        minion1.col = initial_col1;
        minion1.current_path = (goal_row1, goal_col1); // Set the goal
        board1.place_cell(
            minion_content.clone(),
            initial_row1 as usize,
            initial_col1 as usize,
        );

        let expected_next_pos1 =
            calculate_expected_next_pos(initial_row1, initial_col1, goal_row1, goal_col1);
        println!(
            "Case 1: Initial: ({}, {}), Goal: ({}, {}), Expected Next: ({}, {})",
            initial_row1,
            initial_col1,
            goal_row1,
            goal_col1,
            expected_next_pos1.0,
            expected_next_pos1.1
        );

        // Call minion_turn (assuming updated signature fn minion_turn(&mut self, board: &mut Board))
        minion1.movement_phase(&mut board1);

        // Assert the minion's position
        assert_eq!(
            minion1.row, expected_next_pos1.0,
            "Case 1: Minion should move one step towards the goal (row)"
        );
        assert_eq!(
            minion1.col, expected_next_pos1.1,
            "Case 1: Minion should move one step towards the goal (col)"
        );

        // Assert the board state
        let old_cell1 = board1
            .get_cell(initial_row1 as usize, initial_col1 as usize)
            .expect("Case 1: Old cell should exist");
        assert!(
            old_cell1.content.is_none(),
            "Case 1: Old cell should be empty after minion moves"
        );
        let new_cell1 = board1
            .get_cell(expected_next_pos1.0 as usize, expected_next_pos1.1 as usize)
            .expect("Case 1: New cell should exist");
        assert_eq!(
            new_cell1.content,
            Some(minion_content.clone()),
            "Case 1: New cell should have minion content"
        );

        // --- Test Case 2: Move Up-Left ---
        let mut board2 = create_dummy_board(200, 200);
        let initial_row2 = 50;
        let initial_col2 = 50;
        let goal_row2 = 40; // Goal is above and to the left
        let goal_col2 = 40;

        let mut minion2 = Minion::new(minion_id, team_id, Lane::Mid);
        minion2.row = initial_row2;
        minion2.col = initial_col2;
        minion2.current_path = (goal_row2, goal_col2); // Set the goal
        board2.place_cell(
            minion_content.clone(),
            initial_row2 as usize,
            initial_col2 as usize,
        );

        let expected_next_pos2 =
            calculate_expected_next_pos(initial_row2, initial_col2, goal_row2, goal_col2);
        println!(
            "Case 2: Initial: ({}, {}), Goal: ({}, {}), Expected Next: ({}, {})",
            initial_row2,
            initial_col2,
            goal_row2,
            goal_col2,
            expected_next_pos2.0,
            expected_next_pos2.1
        );

        // Call minion_turn
        minion2.movement_phase(&mut board2);

        // Assert the minion's position
        assert_eq!(
            minion2.row, expected_next_pos2.0,
            "Case 2: Minion should move one step towards the goal (row)"
        );
        assert_eq!(
            minion2.col, expected_next_pos2.1,
            "Case 2: Minion should move one step towards the goal (col)"
        );

        // Assert the board state
        let old_cell2 = board2
            .get_cell(initial_row2 as usize, initial_col2 as usize)
            .expect("Case 2: Old cell should exist");
        assert!(
            old_cell2.content.is_none(),
            "Case 2: Old cell should be empty after minion moves"
        );
        let new_cell2 = board2
            .get_cell(expected_next_pos2.0 as usize, expected_next_pos2.1 as usize)
            .expect("Case 2: New cell should exist");
        assert_eq!(
            new_cell2.content,
            Some(minion_content.clone()),
            "Case 2: New cell should have minion content"
        );

        // --- Test Case 3: Move Straight Up ---
        let mut board3 = create_dummy_board(200, 200);
        let initial_row3 = 100;
        let initial_col3 = 100;
        let goal_row3 = 90; // Goal is straight up
        let goal_col3 = 100; // Same column

        let mut minion3 = Minion::new(minion_id, team_id, Lane::Mid);
        minion3.row = initial_row3;
        minion3.col = initial_col3;
        minion3.current_path = (goal_row3, goal_col3); // Set the goal
        board3.place_cell(
            minion_content.clone(),
            initial_row3 as usize,
            initial_col3 as usize,
        );

        let expected_next_pos3 =
            calculate_expected_next_pos(initial_row3, initial_col3, goal_row3, goal_col3);
        println!(
            "Case 3: Initial: ({}, {}), Goal: ({}, {}), Expected Next: ({}, {})",
            initial_row3,
            initial_col3,
            goal_row3,
            goal_col3,
            expected_next_pos3.0,
            expected_next_pos3.1
        );

        // Call minion_turn
        minion3.movement_phase(&mut board3);

        // Assert the minion's position
        assert_eq!(
            minion3.row, expected_next_pos3.0,
            "Case 3: Minion should move one step towards the goal (row)"
        );
        assert_eq!(
            minion3.col, expected_next_pos3.1,
            "Case 3: Minion should move one step towards the goal (col)"
        );

        // Assert the board state
        let old_cell3 = board3
            .get_cell(initial_row3 as usize, initial_col3 as usize)
            .expect("Case 3: Old cell should exist");
        assert!(
            old_cell3.content.is_none(),
            "Case 3: Old cell should be empty after minion moves"
        );
        let new_cell3 = board3
            .get_cell(expected_next_pos3.0 as usize, expected_next_pos3.1 as usize)
            .expect("Case 3: New cell should exist");
        assert_eq!(
            new_cell3.content,
            Some(minion_content.clone()),
            "Case 3: New cell should have minion content"
        );
    }

    #[test]
    fn test_get_potential_target_no_enemy() {
        let mut board = create_dummy_board(50, 50);
        let minion_id: MinionId = 1;
        let minion_team = Team::Blue;
        let minion_row = 25; // Center minion on a large board
        let minion_col = 25;
        let mut minion = Minion::new(minion_id, minion_team, Lane::Mid);
        minion.row = minion_row;
        minion.col = minion_col;

        let scan_range = (10, 10); // 10x10 range

        // Case 1: Empty board
        let target_none = minion.get_potential_target(&board, scan_range);
        assert!(
            target_none.is_none(),
            "Should return None when board is empty"
        );

        // Case 2: Only allies in range
        let ally_team = minion_team;
        let ally_row = minion_row + 1; // Within 10x10 range
        let ally_col = minion_col + 1;
        board.place_cell(
            CellContent::Champion(99, ally_team),
            ally_row as usize,
            ally_col as usize,
        );
        let target_ally = minion.get_potential_target(&board, scan_range);
        assert!(
            target_ally.is_none(),
            "Should return None when only allies are in range"
        );

        // Case 3: Non-entity content or BaseTerrain in range
        board.clear_cell(ally_row as usize, ally_col as usize); // Remove ally
        board.change_base(
            BaseTerrain::Wall,
            (minion_row + 1) as usize,
            minion_col as usize,
        ); // Wall in range
        let target_wall = minion.get_potential_target(&board, scan_range);
        assert!(
            target_wall.is_none(),
            "Should return None when only impassable terrain is in range"
        );
    }

    #[test]
    fn test_get_potential_target_single_enemy() {
        let mut board = create_dummy_board(50, 50);
        let minion_id: MinionId = 1;
        let minion_team = Team::Blue;
        let minion_row = 25; // Center minion
        let minion_col = 25;
        let mut minion = Minion::new(minion_id, minion_team, Lane::Mid);
        minion.row = minion_row;
        minion.col = minion_col;

        let scan_range = (10, 10); // 10x10 range
        let enemy_team = Team::Red; // Opposite team

        // Place an enemy champion in range
        let enemy_row = minion_row + 2; // Within 10x10 range
        let enemy_col = minion_col + 3; // Within 10x10 range
        let enemy_content = CellContent::Champion(99, enemy_team);
        board.place_cell(
            enemy_content.clone(),
            enemy_row as usize,
            enemy_col as usize,
        );

        let target_cell_option = minion.get_potential_target(&board, scan_range);

        assert!(
            target_cell_option.is_some(),
            "Should return Some when an enemy is in range"
        );
        let target_cell = target_cell_option.unwrap();
        assert_eq!(
            target_cell.content,
            Some(enemy_content),
            "The returned cell should contain the enemy champion"
        );
        assert_eq!(
            target_cell.position,
            (enemy_row, enemy_col),
            "The returned cell should be at the enemy's position"
        );

        // Place an enemy minion in range (clear previous enemy)
        board.clear_cell(enemy_row as usize, enemy_col as usize);
        let enemy_minion_row = minion_row - 4; // Within 10x10 range
        let enemy_minion_col = minion_col - 2; // Within 10x10 range
        let enemy_minion_content = CellContent::Minion(2, enemy_team);
        board.place_cell(
            enemy_minion_content.clone(),
            enemy_minion_row as usize,
            enemy_minion_col as usize,
        );

        let target_minion_cell_option = minion.get_potential_target(&board, scan_range);
        assert!(
            target_minion_cell_option.is_some(),
            "Should return Some when an enemy minion is in range"
        );
        let target_minion_cell = target_minion_cell_option.unwrap();
        assert_eq!(
            target_minion_cell.content,
            Some(enemy_minion_content),
            "The returned cell should contain the enemy minion"
        );
        assert_eq!(
            target_minion_cell.position,
            (enemy_minion_row, enemy_minion_col),
            "The returned cell should be at the enemy minion's position"
        );

        // Place an enemy tower in range (clear previous enemy)
        board.clear_cell(enemy_minion_row as usize, enemy_minion_col as usize);
        let enemy_tower_row = minion_row + 1; // Within 10x10 range
        let enemy_tower_col = minion_col - 3; // Within 10x10 range
        let enemy_tower_content = CellContent::Tower(1, enemy_team);
        board.place_cell(
            enemy_tower_content.clone(),
            enemy_tower_row as usize,
            enemy_tower_col as usize,
        );

        let target_tower_cell_option = minion.get_potential_target(&board, scan_range);
        assert!(
            target_tower_cell_option.is_some(),
            "Should return Some when an enemy tower is in range"
        );
        let target_tower_cell = target_tower_cell_option.unwrap();
        assert_eq!(
            target_tower_cell.content,
            Some(enemy_tower_content),
            "The returned cell should contain the enemy tower"
        );
        assert_eq!(
            target_tower_cell.position,
            (enemy_tower_row, enemy_tower_col),
            "The returned cell should be at the enemy tower's position"
        );
    }

    #[test]
    fn test_get_potential_target_multiple_enemies_closest() {
        let mut board = create_dummy_board(50, 50);
        let minion_id: MinionId = 1;
        let minion_team = Team::Blue;
        let minion_row = 25; // Center minion
        let minion_col = 25;
        let mut minion = Minion::new(minion_id, minion_team, Lane::Mid);
        minion.row = minion_row;
        minion.col = minion_col;

        let scan_range = (10, 10); // 10x10 range centered at (25,25)
        let enemy_team = Team::Red; // Opposite team

        // Place multiple enemies at different distances within the 10x10 range
        // Center of the 10x10 view relative to minion is effectively (minion_row, minion_col)
        // Distances are calculated from the center of the *view*, which aligns with minion's position.

        // Closest enemy (Manhattan distance 1 from 25,25)
        let closest_enemy_row = minion_row;
        let closest_enemy_col = minion_col + 1;
        let closest_enemy_content = CellContent::Champion(1, enemy_team);
        board.place_cell(
            closest_enemy_content.clone(),
            closest_enemy_row as usize,
            closest_enemy_col as usize,
        );
        let dist_closest =
            manhattan_distance(minion_row, minion_col, closest_enemy_row, closest_enemy_col);

        // Further enemy (Manhattan distance 3 from 25,25)
        let further_enemy_row_1 = minion_row + 1;
        let further_enemy_col_1 = minion_col + 2;
        let further_enemy_content_1 = CellContent::Minion(1, enemy_team);
        board.place_cell(
            further_enemy_content_1.clone(),
            further_enemy_row_1 as usize,
            further_enemy_col_1 as usize,
        );
        let dist_further1 = manhattan_distance(
            minion_row,
            minion_col,
            further_enemy_row_1,
            further_enemy_col_1,
        );

        // Even further enemy (Manhattan distance 4 from 25,25)
        let further_enemy_row_2 = minion_row - 2;
        let further_enemy_col_2 = minion_col - 2;
        let further_enemy_content_2 = CellContent::Tower(1, enemy_team);
        board.place_cell(
            further_enemy_content_2.clone(),
            further_enemy_row_2 as usize,
            further_enemy_col_2 as usize,
        );
        let dist_further2 = manhattan_distance(
            minion_row,
            minion_col,
            further_enemy_row_2,
            further_enemy_col_2,
        );

        println!(
            "Distances from minion ({},{}): Closest ({},{}) dist {}, Further1 ({},{}) dist {}, Further2 ({},{}) dist {}",
            minion_row,
            minion_col,
            closest_enemy_row,
            closest_enemy_col,
            dist_closest,
            further_enemy_row_1,
            further_enemy_col_1,
            dist_further1,
            further_enemy_row_2,
            further_enemy_col_2,
            dist_further2
        );
        assert!(
            dist_closest < dist_further1 && dist_closest < dist_further2,
            "Closest enemy distance calculation error"
        );

        let target_cell_option = minion.get_potential_target(&board, scan_range);

        assert!(
            target_cell_option.is_some(),
            "Should return Some when multiple enemies are in range"
        );
        let target_cell = target_cell_option.unwrap();
        // Verify that the returned cell contains the closest enemy
        assert_eq!(
            target_cell.content,
            Some(closest_enemy_content),
            "Should return the closest enemy champion"
        );
        assert_eq!(
            target_cell.position,
            (closest_enemy_row, closest_enemy_col),
            "Should return the cell at the closest enemy's position"
        );
    }

    #[test]
    fn test_get_potential_target_enemy_outside_range() {
        let mut board = create_dummy_board(50, 50);
        let minion_id: MinionId = 1;
        let minion_team = Team::Blue;
        let minion_row = 25; // Center minion
        let minion_col = 25;
        let mut minion = Minion::new(minion_id, minion_team, Lane::Mid);
        minion.row = minion_row;
        minion.col = minion_col;

        let scan_range = (5, 5); // 11x11 range centered at (25,25)
        let enemy_team = Team::Red; // Opposite team

        // Place an enemy champion just outside the x10 range
        // Range rows [25-5, 25+5] = [20, 30], cols [25-5, 25+5] = [20, 30]
        // Place enemy at row 19 or 30, or col 19 or 30.
        let enemy_row_outside = minion_row + 5; // row 30, outside range
        let enemy_col_outside = minion_col + 1; // col 26, within range
        board.place_cell(
            CellContent::Champion(99, enemy_team),
            enemy_row_outside as usize,
            enemy_col_outside as usize,
        );

        let target_cell_option = minion.get_potential_target(&board, scan_range);
        assert!(
            target_cell_option.is_none(),
            "Should return None when enemies are outside the specified range (row)"
        );

        // Place an enemy minion just outside the range
        board.clear_cell(enemy_row_outside as usize, enemy_col_outside as usize);
        let enemy_minion_row_outside = minion_row + 1; // row 26, within range
        let enemy_minion_col_outside = minion_col - 6; // col 19, outside range
        board.place_cell(
            CellContent::Minion(2, enemy_team),
            enemy_minion_row_outside as usize,
            enemy_minion_col_outside as usize,
        );

        let target_minion_cell_option = minion.get_potential_target(&board, scan_range);
        assert!(
            target_minion_cell_option.is_none(),
            "Should return None when enemies are outside the specified range (col)"
        );
    }
}
