use std::time::{Duration, Instant};

use rand::seq::IndexedRandom;

use crate::errors::GameError;
use crate::game::entities::reduced_damage;
use crate::game::BaseTerrain;
use crate::game::animation::AnimationTrait;
use crate::game::animation::tower::TowerHitAnimation;
use crate::game::board::Board;
use crate::game::cell::{Cell, CellContent, Team, TowerId};

use super::{Fighter, Stats};

#[derive(Debug)]
pub struct Tower {
    pub tower_id: TowerId,
    pub team_id: Team,
    stats: Stats,
    destroyed: bool,
    last_attacked: Instant,
    pub row: u16,
    pub col: u16,
}

impl Tower {
    pub fn new(tower_id: TowerId, team_id: Team, row: u16, col: u16) -> Self {
        let stats = Stats {
            attack_damage: 40,
            attack_speed: Duration::from_secs(3),
            health: 400,
            armor: 8,
        };

        Tower {
            tower_id,
            team_id,
            stats,
            destroyed: false,
            last_attacked: Instant::now(),
            row,
            col,
        }
    }

    pub fn place_tower(&self, board: &mut Board) {
        board.place_cell(
            CellContent::Tower(self.tower_id, self.team_id),
            self.row as usize,
            self.col as usize,
        );
        board.place_cell(
            CellContent::Tower(self.tower_id, self.team_id),
            self.row as usize - 1,
            self.col as usize,
        );
        board.place_cell(
            CellContent::Tower(self.tower_id, self.team_id),
            self.row as usize,
            self.col as usize + 1,
        );
        board.place_cell(
            CellContent::Tower(self.tower_id, self.team_id),
            self.row as usize - 1,
            self.col as usize + 1,
        );
    }

    pub fn is_destroyed(&self) -> bool {
        self.destroyed
    }

    pub fn destroy_tower(&self, board: &mut Board) {
        // Clear cell
        board.clear_cell(self.row as usize, self.col as usize);
        board.clear_cell(self.row as usize - 1, self.col as usize);
        board.clear_cell(self.row as usize, self.col as usize + 1);
        board.clear_cell(self.row as usize - 1, self.col as usize + 1);

        board.change_base(
            BaseTerrain::TowerDestroyed,
            self.row as usize,
            self.col as usize,
        );
        board.change_base(
            BaseTerrain::TowerDestroyed,
            self.row as usize,
            self.col as usize + 1,
        );
    }
}

impl Fighter for Tower {
    fn take_damage(&mut self, damage: u16) {
        let reduced_damage = reduced_damage(damage, self.stats.armor);
        self.stats.health = self.stats.health.saturating_sub(reduced_damage as u16);
        println!("Tower health: {}", self.stats.health);
        if self.stats.health == 0 {
            println!("Tower got 0 health");
            self.destroyed = true;
        }
    }

    fn can_attack(&mut self) -> Option<(u16, Box<dyn AnimationTrait>)> {
        if self.last_attacked + self.stats.attack_speed < Instant::now() {
            self.last_attacked = Instant::now();
            Some((
                self.stats.attack_damage,
                Box::new(TowerHitAnimation::new(self.row, self.col)),
            ))
        } else {
            None
        }
    }

    fn get_potential_target<'a>(&self, board: &'a Board, range: (u16, u16)) -> Option<&'a Cell> {
        // range is implied here with: 6, 8
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
                        CellContent::Champion(_, team_id) | CellContent::Minion(_, team_id) => {
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

pub fn generate_tower_id() -> Result<TowerId, GameError> {
    let mut rng = rand::rng();
    let nums: Vec<usize> = (1..99999).collect();
    if let Some(id) = nums.choose(&mut rng) {
        Ok(*id)
    } else {
        Err(GameError::GenerateIdError)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::game::{BaseTerrain, Board, cell::CellContent}; // Import necessary types

    // Helper function to create a dummy board
    fn create_dummy_board(rows: usize, cols: usize) -> Board {
        Board::new(rows, cols)
    }

    #[test]
    fn test_new_tower() {
        let tower_id = 1;
        let team_id = Team::Red;
        let row = 10;
        let col = 20;
        let tower = Tower::new(tower_id, team_id, row, col);

        assert_eq!(tower.tower_id, tower_id);
        assert_eq!(tower.team_id, team_id);
        assert_eq!(tower.row, row);
        assert_eq!(tower.col, col);
        // Check initial stats (assuming default values from new())
        assert_eq!(tower.stats.attack_damage, 40);
        assert_eq!(tower.stats.health, 400);
        assert_eq!(tower.stats.armor, 8);
        assert!(
            !tower.destroyed,
            "Newly created tower should not be destroyed"
        );
        // last_attacked will be Instant::now(), difficult to assert exact value
    }

    #[test]
    fn test_is_destroyed() {
        let tower = Tower::new(1, Team::Red, 10, 20);
        assert!(!tower.is_destroyed(), "New tower should not be destroyed");

        let mut destroyed_tower = Tower::new(2, Team::Red, 10, 20);
        destroyed_tower.destroyed = true;
        assert!(
            destroyed_tower.is_destroyed(),
            "Tower marked as destroyed should report true"
        );
    }

    #[test]
    fn test_take_damage() {
        let mut tower = Tower::new(1, Team::Red, 10, 20);
        let initial_health = tower.stats.health;
        let damage = 50;
        let armor = tower.stats.armor as u16;

        tower.take_damage(damage);

        // Calculate expected health after damage reduction by armor
        let reduced_damage = reduced_damage(damage, armor);
        let expected_health = initial_health.saturating_sub(reduced_damage);
        assert_eq!(
            tower.stats.health, expected_health,
            "Health should be reduced after taking damage"
        );
        assert!(
            !tower.is_destroyed(),
            "Tower should not be destroyed after taking partial damage"
        );

        // Test taking enough damage to be destroyed
        let mut tower_to_destroy = Tower::new(2, Team::Red, 10, 20);
        let lethal_damage = 500; // Damage exceeding health + armor

        tower_to_destroy.take_damage(lethal_damage);

        assert_eq!(
            tower_to_destroy.stats.health, 0,
            "Health should be 0 after taking lethal damage"
        );
        assert!(
            tower_to_destroy.is_destroyed(),
            "Tower should be destroyed after taking lethal damage"
        );

        // Test taking damage when already at 0 health (should not go below 0)
        let mut tower_already_destroyed = Tower::new(3, Team::Red, 10, 20);
        tower_already_destroyed.stats.health = 0;
        tower_already_destroyed.destroyed = true;
        let additional_damage = 10;

        tower_already_destroyed.take_damage(additional_damage);
        assert_eq!(
            tower_already_destroyed.stats.health, 0,
            "Health should remain at 0 if already destroyed"
        );
    }

    #[test]
    fn test_place_tower() {
        // Tower::place_tower places content in a 2x2 area starting from (row - 1, col)
        let mut board = create_dummy_board(200, 200); // Board large enough
        let tower_id = 1;
        let team_id = Team::Red;
        let row = 100; // Center row for placing
        let col = 100; // Center col for placing

        let tower = Tower::new(tower_id, team_id, row, col);
        let tower_content = CellContent::Tower(tower_id, team_id);

        tower.place_tower(&mut board);

        // Verify the 2x2 area where the tower should be placed
        // Cells should have the tower's content and default base terrain
        let placed_cells = [
            (row - 1, col),
            (row - 1, col + 1),
            (row, col),
            (row, col + 1),
        ];

        for (r, c) in &placed_cells {
            let cell = board
                .get_cell(*r as usize, *c as usize)
                .expect(&format!("Cell at ({}, {}) should exist", r, c));
            assert_eq!(
                cell.content,
                Some(tower_content.clone()),
                "Cell at ({}, {}) should have tower content",
                r,
                c
            );
            // Assuming default board is BaseTerrain::Floor
            assert_eq!(
                cell.base,
                BaseTerrain::Floor,
                "Cell at ({}, {}) should have default base terrain",
                r,
                c
            );
            assert!(
                cell.animation.is_none(),
                "Cell at ({}, {}) should have no animation",
                r,
                c
            );
        }

        // Verify cells just outside the placed area do NOT have tower content
        let outside_cells = [
            (row - 2, col),
            (row - 1, col - 1),
            (row - 2, col + 1),
            (row - 1, col + 2),
            (row + 1, col),
            (row, col - 1),
            (row + 1, col + 1),
            (row, col + 2),
        ];

        for (r, c) in &outside_cells {
            // Check boundaries before getting cell
            if *r < board.rows as u16 && *c < board.cols as u16 {
                let cell = board.get_cell(*r as usize, *c as usize).expect(&format!(
                    "Cell at ({}, {}) should exist for outside check",
                    r, c
                ));
                assert!(
                    cell.content.is_none() || cell.content != Some(tower_content.clone()),
                    "Cell at ({}, {}) should not have tower content",
                    r,
                    c
                );
            }
        }
    }

    #[test]
    fn test_destroy_tower() {
        // Tower::destroy_tower clears content and changes base in a 2x2 area
        let mut board = create_dummy_board(200, 200); // Board large enough
        let tower_id = 1;
        let team_id = Team::Red;
        let row = 100; // Center row for placing
        let col = 100; // Center col for placing

        let tower = Tower::new(tower_id, team_id, row, col);
        let tower_content = CellContent::Tower(tower_id, team_id);

        // First, place the tower
        tower.place_tower(&mut board);

        // Verify tower content is present before destroying
        let placed_cells = [
            (row - 1, col),
            (row - 1, col + 1),
            (row, col),
            (row, col + 1),
        ];
        for (r, c) in &placed_cells {
            let cell = board.get_cell(*r as usize, *c as usize).expect(&format!(
                "Cell at ({}, {}) should exist before destroying",
                r, c
            ));
            assert_eq!(
                cell.content,
                Some(tower_content.clone()),
                "Cell at ({}, {}) should have tower content before destroying",
                r,
                c
            );
        }

        // Now, destroy the tower
        tower.destroy_tower(&mut board);

        // Verify the 2x2 area after destruction
        // Content should be None, BaseTerrain should be TowerDestroyed for (row, col) and (row, col + 1)
        // Note: Based on the destroy_tower code, it seems to only change base for (row, col) and (row, col + 1),
        // while clearing content for all four cells. We'll test based on the code's implementation.
        let cleared_content_cells = [
            (row - 1, col),
            (row - 1, col + 1),
            (row, col),
            (row, col + 1),
        ];
        for (r, c) in &cleared_content_cells {
            let cell = board.get_cell(*r as usize, *c as usize).expect(&format!(
                "Cell at ({}, {}) should exist after destroying for content check",
                r, c
            ));
            assert!(
                cell.content.is_none(),
                "Cell at ({}, {}) should have no content after destroying",
                r,
                c
            );
        }

        let destroyed_base_cells = [(row, col), (row, col + 1)];
        for (r, c) in &destroyed_base_cells {
            let cell = board.get_cell(*r as usize, *c as usize).expect(&format!(
                "Cell at ({}, {}) should exist after destroying for base check",
                r, c
            ));
            assert_eq!(
                cell.base,
                BaseTerrain::TowerDestroyed,
                "Cell at ({}, {}) should have TowerDestroyed base after destroying",
                r,
                c
            );
        }

        // Verify the other two cells in the 2x2 area still have their original base (Floor)
        let original_base_cells = [(row - 1, col), (row - 1, col + 1)];
        for (r, c) in &original_base_cells {
            let cell = board.get_cell(*r as usize, *c as usize).expect(&format!(
                "Cell at ({}, {}) should exist after destroying for original base check",
                r, c
            ));
            assert_eq!(
                cell.base,
                BaseTerrain::Floor,
                "Cell at ({}, {}) should retain original base after destroying",
                r,
                c
            );
        }
    }

    #[test]
    fn test_tower_scan_range_no_enemy_in_range() {
        let mut board = create_dummy_board(20, 20); // Board large enough for 7x9 range
        let tower_row = 10; // Center row for tower
        let tower_col = 10; // Center col for tower
        let tower_id = 1;
        let tower_team = Team::Red;

        let tower = Tower::new(tower_id, tower_team, tower_row, tower_col);
        // We don't need to place the tower content for scan_range test itself


        // Case 1: No other entities on the board
        let target_none = tower.get_potential_target(&board, (7, 9));
        assert!(target_none.is_none(), "Tower scan_range should return None when no other entities are present");

        // Case 2: Ally champion in range
        let ally_id = 1;
        let ally_team = tower_team;
        let ally_row = tower_row - 2; // Within 7x9 range
        let ally_col = tower_col - 3; // Within 7x9 range
        board.place_cell(CellContent::Champion(ally_id, ally_team), ally_row as usize, ally_col as usize);
        let target_ally_champ = tower.get_potential_target(&board, (7, 9));
        assert!(target_ally_champ.is_none(), "Tower scan_range should return None when only allied champions are in range");
        board.clear_cell(ally_row as usize, ally_col as usize);


        // Case 3: Ally minion in range
         let ally_minion_id = 1;
        let ally_minion_team = tower_team;
        let ally_minion_row = tower_row + 1; // Within 7x9 range
        let ally_minion_col = tower_col + 2; // Within 7x9 range
         board.place_cell(CellContent::Minion(ally_minion_id, ally_minion_team), ally_minion_row as usize, ally_minion_col as usize);
        let target_ally_minion = tower.get_potential_target(&board, (7, 9));
        assert!(target_ally_minion.is_none(), "Tower scan_range should return None when only allied minions are in range");
         board.clear_cell(ally_minion_row as usize, ally_minion_col as usize);


        // Case 4: Enemy tower in range (towers don't target other towers)
        let enemy_tower_id = 2;
        let enemy_tower_team = Team::Blue;
        let enemy_tower_row = tower_row - 1;
        let enemy_tower_col = tower_col + 1;
         board.place_cell(CellContent::Tower(enemy_tower_id, enemy_tower_team), enemy_tower_row as usize, enemy_tower_col as usize);
        let target_enemy_tower = tower.get_potential_target(&board, (7, 9));
        assert!(target_enemy_tower.is_none(), "Tower scan_range should return None when only enemy towers are in range");
        board.clear_cell(enemy_tower_row as usize, enemy_tower_col as usize);
    }

    #[test]
    fn test_tower_scan_range_enemy_in_range() {
        let mut board = create_dummy_board(20, 20); // Board large enough for 7x9 range
        let tower_row = 10; // Center row for tower
        let tower_col = 10; // Center col for tower
        let tower_id = 1;
        let tower_team = Team::Red;

        let tower = Tower::new(tower_id, tower_team, tower_row, tower_col);

        let enemy_team = Team::Blue; // Different team

        // Case 1: Enemy champion in range
        let enemy_champ_id = 1;
        let enemy_champ_row = tower_row + 2; // Within 7x9 range
        let enemy_champ_col = tower_col + 3; // Within 7x9 range
        let enemy_champ_content = CellContent::Champion(enemy_champ_id, enemy_team);
        board.place_cell(enemy_champ_content.clone(), enemy_champ_row as usize, enemy_champ_col as usize);

        let target_champ = tower.get_potential_target(&board, (7, 9));
        assert!(target_champ.is_some(), "Tower scan_range should return Some when an enemy champion is in range");
        let target_champ_cell = target_champ.unwrap();
        assert_eq!(target_champ_cell.content, Some(enemy_champ_content), "The returned cell should contain the enemy champion");
        board.clear_cell(enemy_champ_row as usize, enemy_champ_col as usize);


        // Case 2: Enemy minion in range
        let enemy_minion_id = 1;
        let enemy_minion_row = tower_row - 3; // Within 7x9 range
        let enemy_minion_col = tower_col - 4; // Within 7x9 range
        let enemy_minion_content = CellContent::Minion(enemy_minion_id, enemy_team);
        board.place_cell(enemy_minion_content.clone(), enemy_minion_row as usize, enemy_minion_col as usize);

        let target_minion = tower.get_potential_target(&board, (7, 9));
        assert!(target_minion.is_some(), "Tower scan_range should return Some when an enemy minion is in range");
        let target_minion_cell = target_minion.unwrap();
        assert_eq!(target_minion_cell.content, Some(enemy_minion_content), "The returned cell should contain the enemy minion");
        board.clear_cell(enemy_minion_row as usize, enemy_minion_col as usize);
    }

     #[test]
    fn test_tower_scan_range_multiple_enemies_in_range() {
        let mut board = create_dummy_board(20, 20); // Board large enough for 7x9 range
        let tower_row = 10; // Center row for tower
        let tower_col = 10; // Center col for tower
        let tower_id = 1;
        let tower_team = Team::Red;

        let tower = Tower::new(tower_id, tower_team, tower_row, tower_col);

        let enemy_team = Team::Blue; // Different team

        // Place multiple enemies at different distances within the 7x9 range
        // Tower's reference point is (tower_row, tower_col).
        // The center of the 7x9 view is effectively also (tower_row, tower_col).
        // Manhattan distance is calculated from this center point.

        // Closest enemy (Manhattan distance 2 from 10,10 -> e.g., 9,11)
        let closest_enemy_row = tower_row - 1;
        let closest_enemy_col = tower_col + 1;
        let closest_enemy_content = CellContent::Champion(1, enemy_team);
        board.place_cell(closest_enemy_content.clone(), closest_enemy_row as usize, closest_enemy_col as usize);


        // Further enemy (Manhattan distance 3 from 10,10 -> e.g., 11,11 or 9,12)
        let further_enemy_row_1 = tower_row + 1;
        let further_enemy_col_1 = tower_col + 1;
        let further_enemy_content_1 = CellContent::Minion(1, enemy_team);
        board.place_cell(further_enemy_content_1.clone(), further_enemy_row_1 as usize, further_enemy_col_1 as usize);

         let further_enemy_row_2 = tower_row - 1;
        let further_enemy_col_2 = tower_col + 2;
        let further_enemy_content_2 = CellContent::Champion(2, enemy_team);
        board.place_cell(further_enemy_content_2.clone(), further_enemy_row_2 as usize, further_enemy_col_2 as usize);


        let target = tower.get_potential_target(&board, (7, 9));

        assert!(target.is_some(), "Tower scan_range should return Some when multiple enemies are in range");
        let target_cell = target.unwrap();
        // Verify that the returned cell contains the closest enemy (the one at 9,11)
        assert_eq!(target_cell.content, Some(closest_enemy_content), "Tower scan_range should return the closest enemy");
    }

    #[test]
    fn test_tower_scan_range_enemies_outside_range() {
        let mut board = create_dummy_board(20, 20); // Board large enough
        let tower_row = 10; // Center row for tower
        let tower_col = 10; // Center col for tower
        let tower_id = 1;
        let tower_team = Team::Red;

        let tower = Tower::new(tower_id, tower_team, tower_row, tower_col);

        let enemy_team = Team::Blue; // Different team

        // Place an enemy champion just outside the 7x9 range
        // 7x9 range centered at 10,10 means rows [10-3, 10+3] = [7, 13], cols [10-4, 10+4] = [6, 14]
        // An enemy at row 6 or 14, or col 5 or 15 would be outside.
        let enemy_row_outside = tower_row + 4; // row 14, outside the [7, 13] range
        let enemy_col_outside = tower_col; // col 10, within the [6, 14] range
        board.place_cell(CellContent::Champion(1, enemy_team), enemy_row_outside as usize, enemy_col_outside as usize);


        let target = tower.get_potential_target(&board, (7, 9));

        assert!(target.is_none(), "Tower scan_range should return None when enemies are outside the 7x9 range");

        // Place an enemy minion just outside the range
        let enemy_minion_row_outside = tower_row; // row 10, within range
        let enemy_minion_col_outside = tower_col - 5; // col 5, outside range
         board.place_cell(CellContent::Minion(1, enemy_team), enemy_minion_row_outside as usize, enemy_minion_col_outside as usize);

         let target_minion_outside = tower.get_potential_target(&board, (7, 9));
         assert!(target_minion_outside.is_none(), "Tower scan_range should return None when enemies are outside the 7x9 range");
    }
}
