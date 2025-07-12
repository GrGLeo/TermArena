use std::collections::HashMap;
use std::time::{Duration, Instant};
use std::usize;

use crate::errors::GameError;
use crate::game::Cell;
use crate::game::animation::melee::MeleeAnimation;
use crate::game::buffs::{Buff, HasBuff};
use crate::game::cell::{CellContent, Team};
use crate::game::projectile_manager::ProjectileManager;
use crate::game::spell::freeze_wall::cast_freeze_wall;
use crate::game::{Board, cell::PlayerId};

use super::projectile::GameplayEffect;
use super::{AttackAction, Fighter, Stats, reduced_damage};
use crate::config::{ChampionStats, SpellStats};

#[derive(Debug, Clone, Copy)]
pub enum Direction {
    Up,
    Down,
    Left,
    Right,
}

#[derive(Debug, Clone, Copy)]
pub enum Action {
    MoveUp,
    MoveDown,
    MoveLeft,
    MoveRight,
    Action1,
    Action2,
    InvalidAction,
}

#[derive(Debug)]
pub struct Champion {
    pub player_id: PlayerId,
    pub team_id: Team,
    pub xp: u32,
    pub level: u8,
    pub stats: Stats,
    champion_stats: ChampionStats,
    pub spells: HashMap<String, SpellStats>,
    pub active_buffs: HashMap<String, Box<dyn Buff>>,
    death_counter: u8,
    death_timer: Instant,
    last_attacked: Instant,
    stun_timer: Option<Instant>,
    pub row: u16,
    pub col: u16,
    pub direction: Direction,
}

impl Champion {
    pub fn new(
        player_id: PlayerId,
        team_id: Team,
        row: u16,
        col: u16,
        champion_stats: ChampionStats,
        spells: HashMap<String, SpellStats>,
    ) -> Self {
        let stats = Stats {
            attack_damage: champion_stats.attack_damage,
            attack_speed: Duration::from_millis(champion_stats.attack_speed_ms),
            health: champion_stats.health,
            max_health: champion_stats.health,
            armor: champion_stats.armor,
        };

        Champion {
            player_id,
            stats,
            champion_stats,
            spells,
            xp: 0,
            level: 1,
            death_counter: 0,
            death_timer: Instant::now(),
            last_attacked: Instant::now(),
            stun_timer: None,
            active_buffs: HashMap::new(),
            team_id,
            row,
            col,
            direction: Direction::Up,
        }
    }

    pub fn add_xp(&mut self, xp: u32) {
        self.xp += xp;
        while let Some(xp_needed) = self.xp_for_next_level() {
            if self.xp >= xp_needed {
                self.xp -= xp_needed;
                self.level_up();
            } else {
                break;
            }
        }
    }

    pub fn xp_for_next_level(&self) -> Option<u32> {
        if (self.level as usize - 1) < self.champion_stats.xp_per_level.len() {
            Some(self.champion_stats.xp_per_level[self.level as usize - 1])
        } else {
            None
        }
    }

    fn level_up(&mut self) {
        self.level += 1;
        self.stats.max_health += self.champion_stats.level_up_health_increase;
        self.stats.health += self.champion_stats.level_up_health_increase;
        self.stats.attack_damage += self.champion_stats.level_up_attack_damage_increase;
        self.stats.armor += self.champion_stats.level_up_armor_increase;
    }

    pub fn take_action(
        &mut self,
        action: &Action,
        board: &mut Board,
        projectile_manager: &mut ProjectileManager,
    ) -> Result<(), GameError> {
        // Check if stunned before taking any action
        if self.is_stunned() {
            return Ok(())
        }
        let res = match action {
            Action::MoveUp => {
                self.direction = Direction::Up;
                return self.move_champion(board, -1, 0);
            }
            Action::MoveDown => {
                self.direction = Direction::Down;
                return self.move_champion(board, 1, 0);
            }
            Action::MoveLeft => {
                self.direction = Direction::Left;
                return self.move_champion(board, 0, -1);
            }
            Action::MoveRight => {
                self.direction = Direction::Right;
                return self.move_champion(board, 0, 1);
            }
            Action::Action1 => {
                if let Some(spell_stats) = self.spells.get("freeze_wall") {
                    for blueprint in cast_freeze_wall(&self, self.stats.attack_damage, spell_stats)
                    {
                        projectile_manager.create_from_blueprint(blueprint);
                    }
                }
                Ok(())
            }
            Action::Action2 => Ok(()),
            Action::InvalidAction => {
                Err(GameError::InvalidInput("InvalidAction found".to_string()))
            }
        };
        res
    }

    fn move_champion(
        &mut self,
        board: &mut Board,
        d_row: isize,
        d_col: isize,
    ) -> Result<(), GameError> {
        let new_row = if d_row < 0 {
            self.row.saturating_sub(d_row.unsigned_abs() as u16)
        } else {
            self.row.saturating_add(d_row as u16)
        };

        let new_col = if d_col < 0 {
            self.col.saturating_sub(d_col.unsigned_abs() as u16)
        } else {
            self.col.saturating_add(d_col as u16)
        };

        if new_row >= board.rows as u16 || new_col >= board.cols as u16 {
            return Err(GameError::CannotMoveHere(self.player_id));
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
                return Err(GameError::NotFoundCell);
            }
        } else {
            return Err(GameError::NotFoundCell);
        }
    }

    pub fn place_at_base(&mut self, board: &mut Board) {
        let old_row = self.row;
        let old_col = self.col;
        self.row = 197;
        self.col = 2;
        board.move_cell(
            old_row as usize,
            old_col as usize,
            self.row as usize,
            self.col as usize,
        );
    }

    pub fn is_dead(&self) -> bool {
        if Instant::now() > self.death_timer {
            return false;
        } else {
            true
        }
    }

    pub fn get_health(&self) -> (u16, u16) {
        (self.stats.health, self.stats.max_health)
    }

    pub fn put_at_max_health(&mut self) {
        self.stats.health = self.stats.max_health;
    }
}

impl Fighter for Champion {
    fn take_effect(&mut self, effects: Vec<GameplayEffect>) {
        for effect in effects.into_iter() {
            match effect {
                GameplayEffect::Damage(damage) => {
                    let reduced_damage = reduced_damage(damage, self.stats.armor);
                    self.stats.health = self.stats.health.saturating_sub(reduced_damage as u16);
                    // Check if champion get killed
                    if self.stats.health == 0 {
                        self.death_counter += 1;
                        let timer = ((self.death_counter as f32).sqrt() * 10.) as u64;
                        self.death_timer = Instant::now() + Duration::from_secs(timer);
                    }
                }
                GameplayEffect::Buff(mut buff) => {
                    buff.on_apply(self);
                    self.active_buffs.insert(buff.id().to_string(), buff);
                }
            };
        }
    }

    fn can_attack(&mut self) -> Option<AttackAction> {
        // Cannot attack while stun
        if self.is_stunned() {
            return None;
        }
        if self.last_attacked + self.stats.attack_speed < Instant::now() {
            self.last_attacked = Instant::now();
            let animation = MeleeAnimation::new(self.player_id);
            Some(AttackAction::Melee {
                damage: self.stats.attack_damage,
                animation: Box::new(animation),
            })
        } else {
            None
        }
    }

    fn get_potential_target<'a>(&self, board: &'a Board) -> Option<&'a Cell> {
        let (row_range, col_range) = (
            self.champion_stats.attack_range_row,
            self.champion_stats.attack_range_col,
        );
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
                        | CellContent::Minion(_, team_id)
                        | CellContent::Base(team_id) => {
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

impl HasBuff for Champion {
    fn is_stunned(&self) -> bool {
        self.stun_timer
            .map_or(false, |timer_end| Instant::now() < timer_end)
    }

    fn set_stunned(&mut self, stunned: bool, duration: Option<Duration>) {
        if stunned {
            if let Some(dur) = duration {
                self.stun_timer = Some(Instant::now() + dur);
            } else {
                self.stun_timer = Some(Instant::now() + Duration::from_secs(1));
            }
        } else {
            self.stun_timer = None;
        }
    }
}

#[cfg(test)]
mod tests {

    use super::*;
    use crate::config::ChampionStats;
    use crate::game::BaseTerrain; // Assuming BaseTerrain is needed for Board creation
    use crate::game::Board;

    // Helper function to create a dummy board for tests that require one
    fn create_dummy_board(rows: usize, cols: usize) -> Board {
        Board::new(rows, cols)
    }

    fn create_default_champion_stats() -> ChampionStats {
        ChampionStats {
            attack_damage: 20,
            attack_speed_ms: 2500,
            health: 200,
            armor: 5,
            xp_per_level: vec![
                35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 100, 105, 110, 115,
            ],
            level_up_health_increase: 20,
            level_up_attack_damage_increase: 5,
            level_up_armor_increase: 2,
            attack_range_row: 3,
            attack_range_col: 3,
        }
    }

    #[test]
    fn test_new_champion() {
        let player_id = 1;
        let team_id = Team::Red;
        let row = 10;
        let col = 20;
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let champion = Champion::new(player_id, team_id, row, col, champion_stats, spell_stats);

        assert_eq!(champion.player_id, player_id);
        assert_eq!(champion.team_id, team_id);
        assert_eq!(champion.row, row);
        assert_eq!(champion.col, col);
        // Check initial stats (assuming default values from new())
        assert_eq!(champion.stats.attack_damage, 20);
        assert_eq!(champion.stats.health, 200);
        assert_eq!(champion.stats.armor, 5);
        assert_eq!(champion.death_counter, 0);
        // death_timer and last_attacked will be Instant::now(), difficult to assert exact value
        assert!(
            champion.is_dead() == false,
            "Newly created champion should not be dead"
        );
    }

    #[test]
    fn test_take_damage() {
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 2, 2, champion_stats, spell_stats);
        let initial_health = champion.stats.health;
        let damage = 30;
        let armor = champion.stats.armor as u16;

        champion.take_effect(vec![GameplayEffect::Damage(damage)]);

        // Calculate expected health after damage reduction by armor
        let reduced_damage = reduced_damage(damage, armor);
        let expected_health = initial_health.saturating_sub(reduced_damage);
        assert_eq!(
            champion.stats.health, expected_health,
            "Health should be reduced after taking damage"
        );
        assert!(
            !champion.is_dead(),
            "Champion should not be dead after taking some damage"
        );

        // Test taking enough damage to be defeated
        let champion_stats_defeat = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion_to_defeat =
            Champion::new(2, Team::Red, 10, 20, champion_stats_defeat, spell_stats);
        let lethal_damage = 250; // Damage exceeding health + armor

        // Use a specific instant for death timer check
        let start_time = Instant::now();
        // We'll need to mock or control time for precise testing of death timer,
        // but for now, we can at least check if it's set to *sometime in the future*
        // and that is_dead returns true immediately after taking lethal damage.

        champion_to_defeat.take_effect(vec![GameplayEffect::Damage(lethal_damage)]);

        assert_eq!(
            champion_to_defeat.stats.health, 0,
            "Health should be 0 after taking lethal damage"
        );
        assert!(
            champion_to_defeat.is_dead(),
            "Champion should be dead after taking lethal damage"
        );
        // Simple check that death timer was set to a future time
        assert!(
            champion_to_defeat.death_timer > start_time,
            "Death timer should be set to a future time"
        );
        assert_eq!(
            champion_to_defeat.death_counter, 1,
            "Death counter should increment after first defeat"
        );

        // Test taking damage when already at 0 health (should not go below 0)
        let champion_stats_already_defeated = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion_already_defeated = Champion::new(
            3,
            Team::Red,
            10,
            20,
            champion_stats_already_defeated,
            spell_stats,
        );
        champion_already_defeated.stats.health = 0;
        let additional_damage = 10;

        champion_already_defeated.take_effect(vec![GameplayEffect::Damage(additional_damage)]);
        assert_eq!(
            champion_already_defeated.stats.health, 0,
            "Health should remain at 0 if already defeated"
        );
    }

    #[test]
    fn test_take_action_move() {
        let mut board = create_dummy_board(5, 5);
        let mut pm = ProjectileManager::new();
        let spell_stats = HashMap::new();
        let initial_row = 2;
        let initial_col = 2;
        let player_id = 1;

        // Place the champion on the board
        let champion_stats = create_default_champion_stats();
        let mut champion = Champion::new(
            player_id,
            Team::Red,
            initial_row,
            initial_col,
            champion_stats.clone(),
            spell_stats,
        );
        board.place_cell(
            CellContent::Champion(player_id, Team::Red),
            initial_row as usize,
            initial_col as usize,
        );

        // Test moving up
        let action_up = Action::MoveUp;
        let result_up = champion.take_action(&action_up, &mut board, &mut pm);
        assert!(result_up.is_ok(), "Moving up should be successful");
        assert_eq!(
            champion.row,
            initial_row - 1,
            "Champion row should decrease after moving up"
        );
        assert_eq!(
            champion.col, initial_col,
            "Champion col should remain the same after moving up"
        );
        // Verify board state: old cell is empty, new cell has champion content
        let old_cell_up = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Old cell should exist");
        assert!(
            old_cell_up.content.is_none(),
            "Old cell should be empty after moving up"
        );
        let new_cell_up = board
            .get_cell((initial_row - 1) as usize, initial_col as usize)
            .expect("New cell should exist");
        assert_eq!(
            new_cell_up.content,
            Some(CellContent::Champion(player_id, Team::Red)),
            "New cell should have champion content after moving up"
        );

        // Reset champion position and board for next move test
        let current_row = champion.row; // Current row is now initial_row - 1
        let current_col = champion.col; // Current col is now initial_col
        board.clear_cell(current_row as usize, current_col as usize);
        champion.row = initial_row;
        champion.col = initial_col;
        board.place_cell(
            CellContent::Champion(player_id, Team::Red),
            initial_row as usize,
            initial_col as usize,
        );

        // Test moving right
        let action_right = Action::MoveRight;
        let result_right = champion.take_action(&action_right, &mut board, &mut pm);
        assert!(result_right.is_ok(), "Moving right should be successful");
        assert_eq!(
            champion.row, initial_row,
            "Champion row should remain the same after moving right"
        );
        assert_eq!(
            champion.col,
            initial_col + 1,
            "Champion col should increase after moving right"
        );
        // Verify board state
        let old_cell_right = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Old cell should exist");
        assert!(
            old_cell_right.content.is_none(),
            "Old cell should be empty after moving right"
        );
        let new_cell_right = board
            .get_cell(initial_row as usize, (initial_col + 1) as usize)
            .expect("New cell should exist");
        assert_eq!(
            new_cell_right.content,
            Some(CellContent::Champion(player_id, Team::Red)),
            "New cell should have champion content after moving right"
        );

        // Add tests for MoveDown and MoveLeft similarly...
        // Reset
        let current_row = champion.row; // Current row is now initial_row
        let current_col = champion.col; // Current col is now initial_col + 1
        board.clear_cell(current_row as usize, current_col as usize);
        champion.row = initial_row;
        champion.col = initial_col;
        board.place_cell(
            CellContent::Champion(player_id, Team::Red),
            initial_row as usize,
            initial_col as usize,
        );

        // Test moving down
        let action_down = Action::MoveDown;
        let result_down = champion.take_action(&action_down, &mut board, &mut pm);
        assert!(result_down.is_ok(), "Moving down should be successful");
        assert_eq!(
            champion.row,
            initial_row + 1,
            "Champion row should increase after moving down"
        );
        assert_eq!(
            champion.col, initial_col,
            "Champion col should remain the same after moving down"
        );
        // Verify board state
        let old_cell_down = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Old cell should exist");
        assert!(
            old_cell_down.content.is_none(),
            "Old cell should be empty after moving down"
        );
        let new_cell_down = board
            .get_cell((initial_row + 1) as usize, initial_col as usize)
            .expect("New cell should exist");
        assert_eq!(
            new_cell_down.content,
            Some(CellContent::Champion(player_id, Team::Red)),
            "New cell should have champion content after moving down"
        );

        // Reset
        let current_row = champion.row; // Current row is now initial_row + 1
        let current_col = champion.col; // Current col is now initial_col
        board.clear_cell(current_row as usize, current_col as usize);
        champion.row = initial_row;
        champion.col = initial_col;
        board.place_cell(
            CellContent::Champion(player_id, Team::Red),
            initial_row as usize,
            initial_col as usize,
        );

        // Test moving left
        let action_left = Action::MoveLeft;
        let result_left = champion.take_action(&action_left, &mut board, &mut pm);
        assert!(result_left.is_ok(), "Moving left should be successful");
        assert_eq!(
            champion.row, initial_row,
            "Champion row should remain the same after moving left"
        );
        assert_eq!(
            champion.col,
            initial_col - 1,
            "Champion col should decrease after moving left"
        );
        // Verify board state
        let old_cell_left = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Old cell should exist");
        assert!(
            old_cell_left.content.is_none(),
            "Old cell should be empty after moving left"
        );
        let new_cell_left = board
            .get_cell(initial_row as usize, (initial_col - 1) as usize)
            .expect("New cell should exist");
        assert_eq!(
            new_cell_left.content,
            Some(CellContent::Champion(player_id, Team::Red)),
            "New cell should have champion content after moving left"
        );
    }

    #[test]
    fn test_take_action_move_into_impassable() {
        let mut board = create_dummy_board(5, 5);
        let mut pm = ProjectileManager::new();
        let spell_stats = HashMap::new();
        let initial_row = 2;
        let initial_col = 2;
        let player_id = 1;

        // Place the champion on the board
        let champion_stats = create_default_champion_stats();
        let mut champion = Champion::new(
            player_id,
            Team::Red,
            initial_row,
            initial_col,
            champion_stats.clone(),
            spell_stats,
        );
        board.place_cell(
            CellContent::Champion(player_id, Team::Red),
            initial_row as usize,
            initial_col as usize,
        );

        // Place a wall next to the champion
        let wall_row = initial_row - 1;
        let wall_col = initial_col;
        board.change_base(BaseTerrain::Wall, wall_row as usize, wall_col as usize);

        // Attempt to move into the wall
        let action_up = Action::MoveUp;
        let result_up = champion.take_action(&action_up, &mut board, &mut pm);

        assert!(
            result_up.is_err(),
            "Moving into a wall should return an error"
        );
        assert_eq!(
            champion.row, initial_row,
            "Champion row should not change after failing to move"
        );
        assert_eq!(
            champion.col, initial_col,
            "Champion col should not change after failing to move"
        );
        // Verify board state: champion should still be in the original cell
        let initial_cell = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Initial cell should exist");
        assert_eq!(
            initial_cell.content,
            Some(CellContent::Champion(player_id, Team::Red)),
            "Champion should remain in the initial cell"
        );

        // Place content in a cell next to the champion
        let content_row = initial_row;
        let content_col = initial_col + 1;
        board.place_cell(
            CellContent::Minion(1, Team::Blue),
            content_row as usize,
            content_col as usize,
        );

        // Attempt to move into the cell with content
        let action_right = Action::MoveRight;
        let result_right = champion.take_action(&action_right, &mut board, &mut pm);

        assert!(
            result_right.is_err(),
            "Moving into a cell with content should return an error"
        );
        assert_eq!(
            champion.row, initial_row,
            "Champion row should not change after failing to move"
        );
        assert_eq!(
            champion.col, initial_col,
            "Champion col should not change after failing to move"
        );
        // Verify board state: champion should still be in the original cell, content still in target cell
        let initial_cell_after_fail = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Initial cell should exist");
        assert_eq!(
            initial_cell_after_fail.content,
            Some(CellContent::Champion(player_id, Team::Red)),
            "Champion should remain in the initial cell"
        );
        let target_cell_after_fail = board
            .get_cell(content_row as usize, content_col as usize)
            .expect("Target cell should exist");
        assert_eq!(
            target_cell_after_fail.content,
            Some(CellContent::Minion(1, Team::Blue)),
            "Content should remain in the target cell"
        );
    }

    #[test]
    fn test_take_action_one() {
        let mut board = create_dummy_board(5, 5);
        let mut pm = ProjectileManager::new();
        let champion_stats = create_default_champion_stats();
        let spell_stat = SpellStats {
            range: 10,
            width: 5,
            speed: 1,
            base_damage: 20,
            damage_ratio: 0.8,
            stun_duration: 5,
        };
        let mut spell_stats: HashMap<String, SpellStats> = HashMap::new();
        spell_stats.insert("freeze_wall".to_string(), spell_stat);

        let mut champion = Champion::new(1, Team::Red, 2, 2, champion_stats, spell_stats);

        // Test Action1 (currently does nothing, should not error)
        let action1 = Action::Action1;
        let result1 = champion.take_action(&action1, &mut board, &mut pm);
        assert!(result1.is_ok(), "Action1 should not return an error");

        // Test Action1 correctly created 5 projectiles
        assert_eq!(pm.projectiles.len(), 5);
    }

    #[test]
    fn test_take_action_other_actions() {
        let mut board = create_dummy_board(5, 5);
        let mut pm = ProjectileManager::new();
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 2, 2, champion_stats, spell_stats);

        // Test Action1 (currently does nothing, should not error)
        let action1 = Action::Action1;
        let result1 = champion.take_action(&action1, &mut board, &mut pm);
        assert!(result1.is_ok(), "Action1 should not return an error");

        // Test Action2 (currently does nothing, should not error)
        let action2 = Action::Action2;
        let result2 = champion.take_action(&action2, &mut board, &mut pm);
        assert!(result2.is_ok(), "Action2 should not return an error");
    }

    #[test]
    fn test_take_action_invalid_action() {
        let mut board = create_dummy_board(5, 5);
        let mut pm = ProjectileManager::new();
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 2, 2, champion_stats.clone(), spell_stats);

        // Test InvalidAction
        let invalid_action = Action::InvalidAction;
        let result = champion.take_action(&invalid_action, &mut board, &mut pm);
        println!("{:?}", result);

        assert!(result.is_err(), "InvalidAction should return an error");
        // Optionally, check the specific error type if needed, but checking for an error is sufficient for now.
    }

    #[test]
    fn test_place_at_base() {
        let mut board = create_dummy_board(200, 200); // Use a board large enough for base position
        let initial_row = 10;
        let initial_col = 10;
        let player_id = 1;
        let base_row = 197;
        let base_col = 2;

        // Place champion at initial position
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(
            player_id,
            Team::Red,
            initial_row,
            initial_col,
            champion_stats.clone(),
            spell_stats,
        );
        board.place_cell(
            CellContent::Champion(player_id, Team::Red),
            initial_row as usize,
            initial_col as usize,
        );

        // Place the champion at base
        champion.place_at_base(&mut board);

        // Check if champion's position updated
        assert_eq!(
            champion.row, base_row,
            "Champion's row should update to base row"
        );
        assert_eq!(
            champion.col, base_col,
            "Champion's col should update to base col"
        );

        // Verify board state: old position is empty, base position has champion content
        let old_cell = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Old cell should exist");
        assert!(
            old_cell.content.is_none(),
            "Old position should be empty after placing at base"
        );
        let base_cell = board
            .get_cell(base_row as usize, base_col as usize)
            .expect("Base cell should exist");
        assert_eq!(
            base_cell.content,
            Some(CellContent::Champion(player_id, Team::Red)),
            "Base position should have champion content"
        );
    }

    #[test]
    fn test_scan_range_no_enemy_in_range() {
        let mut board = create_dummy_board(10, 10);
        let champion_row = 5;
        let champion_col = 5;
        let player_id = 1;
        let champion_team = Team::Red;

        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let champion = Champion::new(
            player_id,
            champion_team,
            champion_row,
            champion_col,
            champion_stats,
            spell_stats,
        );
        board.place_cell(
            CellContent::Champion(player_id, champion_team),
            champion_row as usize,
            champion_col as usize,
        );

        // Case 1: No other entities on the board
        let target_none = champion.get_potential_target(&board);
        assert!(
            target_none.is_none(),
            "scan_range should return None when no other entities are present"
        );

        // Case 2: Ally champion in range
        let ally_id = 2;
        let ally_row = champion_row - 1; // Within 3x3 range
        let ally_col = champion_col;
        board.place_cell(
            CellContent::Champion(ally_id, champion_team),
            ally_row as usize,
            ally_col as usize,
        );
        let target_ally = champion.get_potential_target(&board);
        assert!(
            target_ally.is_none(),
            "scan_range should return None when only allies are in range"
        );

        // Case 3: Non-entity content in range (e.g., Flag - although Flag is CellContent, it might not be a "target")
        // Based on scan_range implementation, Flag and Champion/Minion/Tower on the same team are filtered out.
        // Let's explicitly place a Flag of the same team to be sure.
        let flag_id = 1;
        let flag_row = champion_row;
        let flag_col = champion_col + 1;
        board.place_cell(
            CellContent::Flag(flag_id, champion_team),
            flag_row as usize,
            flag_col as usize,
        );
        let target_flag_ally = champion.get_potential_target(&board);
        assert!(
            target_flag_ally.is_none(),
            "scan_range should return None when only allied flags are in range"
        );

        // Clean up the board for next tests (optional in unit tests, but good practice)
        board.clear_cell(ally_row as usize, ally_col as usize);
        board.clear_cell(flag_row as usize, flag_col as usize);
    }

    #[test]
    fn test_scan_range_enemy_in_range() {
        let mut board = create_dummy_board(10, 10);
        let champion_row = 5;
        let champion_col = 5;
        let player_id = 1;
        let champion_team = Team::Red;

        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let champion = Champion::new(
            player_id,
            champion_team,
            champion_row,
            champion_col,
            champion_stats,
            spell_stats,
        );
        board.place_cell(
            CellContent::Champion(player_id, champion_team),
            champion_row as usize,
            champion_col as usize,
        );

        // Place an enemy champion in range
        let enemy_id = 2;
        let enemy_team = Team::Blue; // Different team
        let enemy_row = champion_row + 1; // Within 3x3 range
        let enemy_col = champion_col + 1; // Within 3x3 range
        let enemy_cell_content = CellContent::Champion(enemy_id, enemy_team);
        board.place_cell(
            enemy_cell_content.clone(),
            enemy_row as usize,
            enemy_col as usize,
        );

        let target = champion.get_potential_target(&board);

        assert!(
            target.is_some(),
            "scan_range should return Some when an enemy is in range"
        );
        let target_cell = target.unwrap();
        assert_eq!(
            target_cell.content,
            Some(enemy_cell_content),
            "The returned cell should contain the enemy champion"
        );

        // Check another enemy type (Tower)
        let tower_id = 1;
        let tower_team = Team::Blue;
        let tower_row = champion_row - 1;
        let tower_col = champion_col;
        let tower_cell_content = CellContent::Tower(tower_id, tower_team);
        board.clear_cell(enemy_row as usize, enemy_col as usize); // Remove previous enemy
        board.place_cell(
            tower_cell_content.clone(),
            tower_row as usize,
            tower_col as usize,
        );

        let target_tower = champion.get_potential_target(&board);
        assert!(
            target_tower.is_some(),
            "scan_range should return Some when an enemy tower is in range"
        );
        let target_tower_cell = target_tower.unwrap();
        assert_eq!(
            target_tower_cell.content,
            Some(tower_cell_content),
            "The returned cell should contain the enemy tower"
        );
    }

    #[test]
    fn test_scan_range_multiple_enemies_in_range() {
        let mut board = create_dummy_board(10, 10);
        let champion_row = 5;
        let champion_col = 5;
        let player_id = 1;
        let champion_team = Team::Red;

        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let champion = Champion::new(
            player_id,
            champion_team,
            champion_row,
            champion_col,
            champion_stats,
            spell_stats,
        );
        board.place_cell(
            CellContent::Champion(player_id, champion_team),
            champion_row as usize,
            champion_col as usize,
        );

        // Place multiple enemies at different distances within range
        let enemy_team = Team::Blue;

        // Closest enemy (Manhattan distance 1)
        let closest_enemy_row = champion_row;
        let closest_enemy_col = champion_col + 1;
        let closest_enemy_content = CellContent::Champion(2, enemy_team);
        board.place_cell(
            closest_enemy_content.clone(),
            closest_enemy_row as usize,
            closest_enemy_col as usize,
        );

        // Further enemy (Manhattan distance 2)
        let further_enemy_row = champion_row + 1;
        let further_enemy_col = champion_col + 1;
        let further_enemy_content = CellContent::Minion(1, enemy_team);
        board.place_cell(
            further_enemy_content.clone(),
            further_enemy_row as usize,
            further_enemy_col as usize,
        );

        // Even further enemy (Manhattan distance 2)
        let even_further_enemy_row = champion_row - 1;
        let even_further_enemy_col = champion_col - 1;
        let even_further_enemy_content = CellContent::Tower(1, enemy_team);
        board.place_cell(
            even_further_enemy_content.clone(),
            even_further_enemy_row as usize,
            even_further_enemy_col as usize,
        );

        let target = champion.get_potential_target(&board);

        assert!(
            target.is_some(),
            "scan_range should return Some when multiple enemies are in range"
        );
        let target_cell = target.unwrap();
        // Verify that the returned cell contains the closest enemy
        assert_eq!(
            target_cell.content,
            Some(closest_enemy_content),
            "scan_range should return the closest enemy"
        );
    }

    #[test]
    fn test_scan_range_enemies_outside_range() {
        let mut board = create_dummy_board(10, 10);
        let champion_row = 5;
        let champion_col = 5;
        let player_id = 1;
        let champion_team = Team::Red;

        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let champion = Champion::new(
            player_id,
            champion_team,
            champion_row,
            champion_col,
            champion_stats,
            spell_stats,
        );
        board.place_cell(
            CellContent::Champion(player_id, champion_team),
            champion_row as usize,
            champion_col as usize,
        );

        // Place an enemy champion outside the 3x3 range
        let enemy_id = 2;
        let enemy_team = Team::Blue; // Different team
        let enemy_row_outside = champion_row + 2; // Outside 3x3 range (center is 1 tile away, edge is 1 tile away, 2 is outside)
        let enemy_col_outside = champion_col + 2; // Outside 3x3 range
        board.place_cell(
            CellContent::Champion(enemy_id, enemy_team),
            enemy_row_outside as usize,
            enemy_col_outside as usize,
        );

        let target = champion.get_potential_target(&board);

        assert!(
            target.is_none(),
            "scan_range should return None when enemies are outside the 3x3 range"
        );
    }

    #[test]
    fn test_champion_stun_application() {
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 0, 0, champion_stats, spell_stats);
        let mut board = create_dummy_board(10, 10);
        let mut pm = ProjectileManager::new();

        // Apply a stun buff
        let stun_duration_secs = 5;
        let stun_effect = GameplayEffect::Buff(Box::new(StunBuff::new(stun_duration_secs)));
        champion.take_effect(vec![stun_effect]);

        // Assert champion is stunned
        assert!(champion.is_stunned(), "Champion should be stunned after applying stun buff");

        // Assert stunned champion cannot move
        let move_action = Action::MoveUp;
        let move_result = champion.take_action(&move_action, &mut board, &mut pm);
        assert!(move_result.is_err(), "Stunned champion should not be able to move");
        assert_eq!(move_result.unwrap_err(), GameError::IsStunned, "Stunned champion move error should be GameError::IsStunned");

        // Assert stunned champion cannot attack
        assert!(champion.can_attack().is_none(), "Stunned champion should not be able to attack");
    }

    #[test]
    fn test_champion_stun_expiration() {
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 0, 0, champion_stats, spell_stats);

        // Apply a very short stun buff
        let stun_effect = GameplayEffect::Buff(Box::new(StunBuff::new(0))); // Duration 0 for immediate expiration
        champion.take_effect(vec![stun_effect]);

        // Manually process buffs to trigger expiration
        let current_buffs = std::mem::take(&mut champion.active_buffs);
        let mut kept_buffs = HashMap::new();
        for (id, mut buff) in current_buffs.into_iter() {
            if buff.on_tick(&mut champion) {
                buff.on_remove(&mut champion);
            } else {
                kept_buffs.insert(id, buff);
            }
        }
        champion.active_buffs = kept_buffs;

        // Assert champion is no longer stunned
        assert!(!champion.is_stunned(), "Champion should not be stunned after buff expiration");

        // Assert champion can now move (assuming board and pm are set up for a valid move)
        let mut board = create_dummy_board(10, 10);
        let mut pm = ProjectileManager::new();
        // Place champion on board for movement test
        board.place_cell(CellContent::Champion(champion.player_id, champion.team_id), champion.row as usize, champion.col as usize);
        let move_action = Action::MoveDown;
        let move_result = champion.take_action(&move_action, &mut board, &mut pm);
        assert!(move_result.is_ok(), "Unstunned champion should be able to move");

        // Assert champion can now attack
        // For can_attack to return Some, last_attacked needs to be old enough.
        // In a real test, you might mock Instant::now() or set last_attacked explicitly.
        // For simplicity here, we'll just check if it's not None.
        // Note: This test might be flaky if run too quickly after champion creation due to Instant::now()
        // A more robust test would involve setting champion.last_attacked to a past time.
        champion.last_attacked = Instant::now() - champion.stats.attack_speed - Duration::from_secs(1);
        assert!(champion.can_attack().is_some(), "Unstunned champion should be able to attack");
    }

    fn test_level_up() {
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 0, 0, champion_stats, spell_stats);
        assert_eq!(champion.level, 1);
        assert_eq!(champion.stats.max_health, 200);
        assert_eq!(champion.stats.attack_damage, 20);
        assert_eq!(champion.stats.armor, 5);

        champion.add_xp(35);
        assert_eq!(champion.level, 2);
        assert_eq!(champion.xp, 0);
        assert_eq!(champion.stats.max_health, 220);
        assert_eq!(champion.stats.attack_damage, 25);
        assert_eq!(champion.stats.armor, 7);

        champion.add_xp(40);
        assert_eq!(champion.level, 3);
        assert_eq!(champion.xp, 0);
        assert_eq!(champion.stats.max_health, 240);
        assert_eq!(champion.stats.attack_damage, 30);
        assert_eq!(champion.stats.armor, 9);
    }
}
