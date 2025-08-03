use crate::config::MonsterStats;
use crate::game::entities::monster::Monster;
use std::collections::HashMap;

use super::algorithms::pathfinding::{find_path_on_board, is_adjacent_to_goal};
use super::animation::AnimationTrait;
use super::cell::MonsterId;
use super::entities::monster::MonsterState;
use super::entities::projectile::GameplayEffect;
use super::entities::{AttackAction, Fighter, Target};
use super::{Board, CellContent, Champion, PlayerId};

pub struct MonsterManager {
    pub monster_definitions: HashMap<String, MonsterStats>,

    pub active_monsters: HashMap<usize, Monster>,
    next_instance_id: MonsterId,
}

impl MonsterManager {
    pub fn new(monsters: Vec<MonsterStats>) -> MonsterManager {
        let monster_definitions = monsters
            .into_iter()
            .map(|monster| (monster.id.clone(), monster))
            .collect();
        MonsterManager {
            monster_definitions,
            active_monsters: HashMap::new(),
            next_instance_id: 1,
        }
    }

    pub fn spawn_monster(&mut self, name_id: &str, board: &mut Board) {
        if let Some(monster_def) = self.monster_definitions.get(name_id) {
            let monster = Monster::new(self.next_instance_id, monster_def.clone());
            board.place_cell(
                CellContent::Monster(monster.id),
                monster.row as usize,
                monster.col as usize,
            );
            self.active_monsters.insert(self.next_instance_id, monster);
            self.next_instance_id += 1;
        } else {
            ()
        }
    }

    pub fn spawn_initial_monsters(&mut self, board: &mut Board) {
        let definitions_to_spawn: Vec<_> = self.monster_definitions.keys().cloned().collect();
        for def_id in definitions_to_spawn {
            self.spawn_monster(&def_id, board);
        }
    }

    pub fn apply_effects_to_monster(
        &mut self,
        monster_id: &MonsterId,
        effects: Vec<GameplayEffect>,
        player_id: PlayerId,
    ) -> Option<(PlayerId, u8, u16)> {
        if let Some(monster) = self.active_monsters.get_mut(monster_id) {
            monster.take_effect(effects);
            monster.attach_target(player_id);
            if monster.stats.health == 0 {
                let monster_def = self.monster_definitions.get(&monster.monster_id).unwrap();
                return Some((player_id, monster_def.xp_reward, monster_def.gold_reward));
            }
        }
        None
    }

    pub fn update(
        &mut self,
        board: &mut Board,
        champions: &HashMap<PlayerId, Champion>,
    ) -> (
        Vec<(Target, Vec<GameplayEffect>)>,
        Vec<Box<dyn AnimationTrait>>,
    ) {
        let mut pending_damages: Vec<(Target, Vec<GameplayEffect>)> = Vec::new();
        let mut new_animations: Vec<Box<dyn AnimationTrait>> = Vec::new();
        let mut dead_monster: Vec<(MonsterId, String)> = Vec::new();
        for monster in self.active_monsters.values_mut() {
            match monster.state {
                MonsterState::Idle => {}
                MonsterState::Aggro => {
                    //  First we ensure the monster has a valid target champion.
                    if let Some(champion_id) = monster.target_champion_id {
                        if let Some(champion) = champions.get(&champion_id) {
                            // 1. We check leash range
                            let delta_row = monster.row.abs_diff(monster.spawn_row);
                            let delta_col = monster.col.abs_diff(monster.spawn_col);
                            if delta_row >= monster.leash_range as u16
                                || delta_col >= monster.leash_range as u16
                            {
                                monster.start_returning(board);
                                continue; // We stop processing for this monster
                            }
                            // 2. We check enemy is in range
                            if is_adjacent_to_goal(
                                (monster.row, monster.col),
                                (champion.row, champion.col),
                            ) {
                                if let Some(attack_action) = monster.can_attack() {
                                    if let AttackAction::Melee { damage, animation } = attack_action
                                    {
                                        new_animations.push(animation);
                                        pending_damages.push((
                                            Target::Champion(champion_id),
                                            vec![GameplayEffect::Damage(damage)],
                                        ));
                                    }
                                }
                            } else {
                                if monster.path.is_none() {
                                    monster.path = find_path_on_board(
                                        board,
                                        (monster.row, monster.col),
                                        (champion.row, champion.col),
                                    );
                                }
                                if let Some(path) = &mut monster.path {
                                    if let Some(next_path) = path.pop_front() {
                                        let old_row = monster.row;
                                        let old_col = monster.col;
                                        monster.row = next_path.0;
                                        monster.col = next_path.1;
                                        board.move_cell(
                                            old_row as usize,
                                            old_col as usize,
                                            monster.row as usize,
                                            monster.col as usize,
                                        );
                                    } else {
                                        monster.path = None;
                                    }
                                }
                            }
                        } else {
                            // Target champion doesn't exist anymore
                            monster.start_returning(board);
                        }
                    }
                }
                MonsterState::Returning => {
                    if let Some(path) = &mut monster.path {
                        if let Some(p) = path.pop_front() {
                            let old_row = monster.row;
                            let old_col = monster.col;
                            monster.row = p.0;
                            monster.col = p.1;
                            board.move_cell(
                                old_row as usize,
                                old_col as usize,
                                monster.row as usize,
                                monster.col as usize,
                            );
                            if monster.row == monster.spawn_row && monster.col == monster.spawn_col
                            {
                                monster.reset();
                            }
                        } else {
                            if monster.row == monster.spawn_row && monster.col == monster.spawn_col
                            {
                                monster.reset();
                            }
                        }
                    } else {
                        if monster.row == monster.spawn_row && monster.col == monster.spawn_col {
                            monster.reset();
                        }
                    }
                }
                MonsterState::Dead => {
                    board.clear_cell(monster.row as usize, monster.col as usize);
                    if monster.can_respawn() {
                        dead_monster.push((monster.id, monster.monster_id.clone()));
                    }
                }
            }
        }
        for (id, monster_id) in dead_monster.iter() {
            self.active_monsters.remove(id);
            self.spawn_monster(monster_id, board);
        }
        return (pending_damages, new_animations);
    }
}

#[cfg(test)]
mod tests {
    use crate::{
        config::ChampionStats,
        game::{Board, Champion, cell::Team, entities::monster::MonsterState},
    };

    use super::*;

    // Helper to create monster stats for testing
    fn create_test_monster_stats(id: &str, spawn_row: u16, spawn_col: u16) -> MonsterStats {
        MonsterStats {
            id: id.to_string(),
            spawn_row,
            spawn_col,
            health: 100,
            armor: 5,
            attack_damage: 10,
            attack_range_row: 1,
            attack_range_col: 1,
            aggro_range_row: 8,
            aggro_range_col: 8,
            leash_range: 10,
            xp_reward: 30,
            gold_reward: 50,
            respawn_timer_secs: 60,
            attack_speed_ms: 1000,
        }
    }

    fn create_default_champion_stats() -> ChampionStats {
        ChampionStats {
            attack_damage: 20,
            attack_speed_ms: 2500,
            health: 200,
            mana: 100,
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

    fn create_champion(row: u16, col: u16) -> Champion {
        let player_id = 1;
        let team_id = Team::Red;
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        Champion::new(player_id, team_id, row, col, champion_stats, spell_stats)
    }

    #[test]
    fn test_new_monster_manager_initializes_correctly() {
        let monster_defs = vec![
            create_test_monster_stats("wolf_red", 10, 10),
            create_test_monster_stats("wolf_blue", 15, 15),
        ];

        let manager = MonsterManager::new(monster_defs);

        // Check that definitions are stored correctly
        assert_eq!(manager.monster_definitions.len(), 2);
        assert!(manager.monster_definitions.contains_key("wolf_red"));
        assert!(manager.monster_definitions.contains_key("wolf_blue"));
        assert_eq!(
            manager
                .monster_definitions
                .get("wolf_red")
                .unwrap()
                .spawn_row,
            10
        );

        // Check that active monsters list is empty initially
        assert!(manager.active_monsters.is_empty());

        // Check that the instance ID counter is initialized
        assert_eq!(manager.next_instance_id, 1);
    }

    #[test]
    fn test_spawn_monster_creates_and_adds_monster() {
        let monster_defs = vec![create_test_monster_stats("wolf_red", 10, 10)];
        let mut manager = MonsterManager::new(monster_defs);
        let mut board = Board::new(100, 100);

        // Spawn the monster
        manager.spawn_monster("wolf_red", &mut board);

        // Check that there is one active monster
        assert_eq!(
            manager.active_monsters.len(),
            1,
            "Should be one active monster"
        );
        // Check that the cellcontent got correctly set
        if let Some(cell) = board.get_cell(10, 10) {
            if let Some(content) = &cell.content {
                assert_eq!(CellContent::Monster(1), *content)
            }
        }

        // Check that the next ID has been incremented
        assert_eq!(manager.next_instance_id, 2, "Next instance ID should be 2");

        // Get the monster and verify its properties
        let monster = manager
            .active_monsters
            .get(&1)
            .expect("Monster with ID 1 should exist");
        assert_eq!(monster.id, 1);
        assert_eq!(monster.monster_id, "wolf_red");
        assert_eq!(
            monster.spawn_row, 10,
            "Monster should spawn at the definition's coordinates"
        );
        assert_eq!(monster.spawn_col, 10);
        assert_eq!(manager.next_instance_id, 2, "Next instance ID should be 2");

        // Get the monster and verify its properties
        let monster = manager
            .active_monsters
            .get(&1)
            .expect("Monster with ID 1 should exist");
        assert_eq!(monster.id, 1);
        assert_eq!(monster.monster_id, "wolf_red");
        assert_eq!(
            monster.spawn_row, 10,
            "Monster should spawn at the definition's coordinates"
        );
        assert_eq!(monster.spawn_col, 10);
        assert_eq!(monster.row, 10);
        assert_eq!(monster.col, 10);
    }

    #[test]
    fn test_apply_effects_sets_aggro_on_idle_monster() {
        let monster_defs = vec![create_test_monster_stats("wolf_red", 10, 10)];
        let mut manager = MonsterManager::new(monster_defs);
        let mut board = Board::new(100, 100);
        manager.spawn_monster("wolf_red", &mut board);

        let monster_id = 1;
        let attacker_id = 42; // Player's ID

        // Apply damage effect
        let effects = vec![GameplayEffect::Damage(30)];
        manager.apply_effects_to_monster(&monster_id, effects, attacker_id);

        // Get the monster to check its new state
        let monster = manager.active_monsters.get(&monster_id).unwrap();

        // Verify health, state, and target
        assert_eq!(monster.stats.health, 70);
        assert_eq!(monster.state, MonsterState::Aggro);
        assert_eq!(monster.target_champion_id, Some(attacker_id));
    }

    #[test]
    fn test_apply_effects_does_not_change_target_on_aggro_monster() {
        let monster_defs = vec![create_test_monster_stats("wolf_red", 10, 10)];
        let mut manager = MonsterManager::new(monster_defs);
        let mut board = Board::new(100, 100);
        manager.spawn_monster("wolf_red", &mut board);

        let monster_id = 1;
        let attacker_1 = 42; // First attacker
        let attacker_2 = 99; // Second attacker

        // First attack sets the aggro
        manager.apply_effects_to_monster(&monster_id, vec![GameplayEffect::Damage(10)], attacker_1);
        let monster = manager.active_monsters.get(&monster_id).unwrap();
        assert_eq!(
            monster.target_champion_id,
            Some(attacker_1),
            "Target should be the first attacker"
        );
        assert_eq!(monster.stats.health, 90);

        // Second attack from a different champion
        manager.apply_effects_to_monster(&monster_id, vec![GameplayEffect::Damage(10)], attacker_2);
        let monster = manager.active_monsters.get(&monster_id).unwrap();

        // Verify health is reduced, but target remains unchanged
        assert_eq!(monster.stats.health, 80, "Health should be further reduced");
        assert_eq!(
            monster.target_champion_id,
            Some(attacker_1),
            "Target should NOT change to the second attacker"
        );
    }

    #[test]
    fn test_update_leashes_monster_when_far_from_spawn() {
        // Leash range in test stats is 10. Spawn is (10, 10).
        let monster_defs = vec![create_test_monster_stats("wolf_red", 10, 10)];
        let mut manager = MonsterManager::new(monster_defs);
        let mut board = Board::new(100, 100);
        manager.spawn_monster("wolf_red", &mut board);
        let monster_id = 1;
        let attacker_id = 42;

        let mut champions = HashMap::new();
        champions.insert(attacker_id, create_champion(15, 15)); // Champion position is irrelevant for the leash calculation itself

        // Make the monster aggro
        manager.apply_effects_to_monster(&monster_id, vec![], attacker_id);

        // Manually move the monster far from its spawn point to simulate it being kited
        let monster = manager.active_monsters.get_mut(&monster_id).unwrap();
        monster.row = 21; // This is 11 units away from spawn row 10, exceeding leash range of 10
        monster.col = 10;
        assert_eq!(
            monster.state,
            MonsterState::Aggro,
            "Monster should be aggro initially"
        );

        // Call the update loop
        manager.update(&mut board, &champions);

        // Verify the monster is now returning because it's too far from its spawn
        let monster = manager.active_monsters.get(&monster_id).unwrap();
        assert_eq!(
            monster.state,
            MonsterState::Returning,
            "Monster should be returning after being leashed"
        );
        assert!(
            monster.target_champion_id.is_none(),
            "Monster target should be cleared when returning"
        );
    }

    #[test]
    fn test_update_moves_aggro_monster_towards_target() {
        // Attack range is 1, Leash range is 10. Spawn is (10, 10)
        let monster_defs = vec![create_test_monster_stats("wolf_red", 10, 10)];
        let mut manager = MonsterManager::new(monster_defs);
        let mut board = Board::new(100, 100);
        manager.spawn_monster("wolf_red", &mut board);
        let monster_id = 1;
        let attacker_id = 42;

        let mut champions = HashMap::new();
        // Place champion within leash range but outside attack range
        champions.insert(attacker_id, create_champion(15, 10));

        // Make monster aggro
        manager.apply_effects_to_monster(&monster_id, vec![], attacker_id);
        let monster = manager.active_monsters.get(&monster_id).unwrap();
        let initial_pos = (monster.row, monster.col);
        assert_eq!(initial_pos, (10, 10));

        // Call the update loop
        manager.update(&mut board, &champions);

        // Verify the monster has moved one step towards the champion
        let monster = manager.active_monsters.get(&monster_id).unwrap();
        let new_pos = (monster.row, monster.col);
        assert_ne!(new_pos, initial_pos, "Monster should have moved");
        // The path should be straight down in this case
        assert_eq!(
            new_pos,
            (11, 9),
            "Monster should move one step along the path to the target"
        );
    }

    #[test]
    fn test_update_attacks_champion_in_range() {
        // Attack range is 1. Spawn is (10, 10).
        let monster_defs = vec![create_test_monster_stats("wolf_red", 10, 10)];
        let mut manager = MonsterManager::new(monster_defs);
        let mut board = Board::new(100, 100);
        manager.spawn_monster("wolf_red", &mut board);
        let monster_id = 1;
        let attacker_id = 42;

        let mut champions = HashMap::new();
        // Place champion right next to the monster
        champions.insert(attacker_id, create_champion(10, 11));

        // Make monster aggro and expire its attack cooldown so it can attack immediately
        manager.apply_effects_to_monster(&monster_id, vec![], attacker_id);
        let monster = manager.active_monsters.get_mut(&monster_id).unwrap();
        monster.last_attacked = std::time::Instant::now() - std::time::Duration::from_secs(5);
        let initial_pos = (monster.row, monster.col);

        // Call the update loop
        let (pending_effects, animation) = manager.update(&mut board, &mut champions);

        // Verify the monster did NOT move
        let monster = manager.active_monsters.get(&monster_id).unwrap();
        let new_pos = (monster.row, monster.col);
        assert_eq!(
            new_pos, initial_pos,
            "Monster should not move when in attack range"
        );

        assert_eq!(
            pending_effects.len(),
            1,
            "An attack should generate 1 effect"
        );
        assert_eq!(animation.len(), 1, "An attack should generate 1 animation");

        let (target, effect) = &pending_effects[0];
        assert_eq!(*target, Target::Champion(attacker_id));
        assert_eq!(effect.len(), 1);
        assert_eq!(effect[0], GameplayEffect::Damage(10))
    }

    #[test]
    fn test_update_moves_returning_monster_towards_spawn() {
        let monster_defs = vec![create_test_monster_stats("wolf_red", 10, 10)];
        let mut manager = MonsterManager::new(monster_defs);
        let mut board = Board::new(100, 100);
        manager.spawn_monster("wolf_red", &mut board);
        let monster_id = 1;

        let mut champions = HashMap::new();

        // Manually put the monster in a returning state from a different position
        let monster = manager.active_monsters.get_mut(&monster_id).unwrap();
        monster.row = 15;
        monster.col = 15;
        monster.start_returning(&board);
        assert_eq!(monster.state, MonsterState::Returning);
        let initial_pos = (monster.row, monster.col);

        // Call the update loop
        manager.update(&mut board, &mut champions);

        // Verify the monster has moved one step towards its spawn diagonally
        let monster = manager.active_monsters.get(&monster_id).unwrap();
        let new_pos = (monster.row, monster.col);
        assert_ne!(new_pos, initial_pos, "Monster should have moved");
        assert_eq!(
            new_pos,
            (14, 14),
            "Monster should move one step diagonally along the path to its spawn"
        );
    }

    #[test]
    fn test_update_resets_monster_when_it_reaches_spawn() {
        let monster_defs = vec![create_test_monster_stats("wolf_red", 10, 10)];
        let mut manager = MonsterManager::new(monster_defs);
        let mut board = Board::new(100, 100);
        manager.spawn_monster("wolf_red", &mut board);
        let monster_id = 1;

        let mut champions = HashMap::new();

        // Manually put the monster in a returning state, right next to its spawn
        // We create a scope for the mutable borrow
        {
            let monster = manager.active_monsters.get_mut(&monster_id).unwrap();
            monster.row = 11;
            monster.col = 11;
            monster.stats.health = 50; // Make sure it needs healing
            monster.start_returning(&board);
        }
        manager.update(&mut board, &champions);
        let monster = manager.active_monsters.get_mut(&monster_id).unwrap();
        assert_eq!(monster.state, MonsterState::Returning);

        // Call the update loop
        manager.update(&mut board, &mut champions);

        // Verify the monster has been reset
        let monster = manager.active_monsters.get(&monster_id).unwrap();
        assert_eq!(
            (monster.row, monster.col),
            (10, 10),
            "Monster should be at its spawn point"
        );
        assert_eq!(
            monster.state,
            MonsterState::Idle,
            "Monster should be Idle after returning"
        );
        assert_eq!(
            monster.stats.health, monster.stats.max_health,
            "Monster health should be restored"
        );
        assert!(
            monster.path.is_none(),
            "Path should be cleared after returning"
        );
    }

    #[test]
    fn test_update_respawns_monster_when_ready() {
        let monster_defs = vec![create_test_monster_stats("wolf_red", 10, 10)];
        let mut manager = MonsterManager::new(monster_defs);
        let mut board = Board::new(100, 100);
        manager.spawn_monster("wolf_red", &mut board);
        let monster_id = 1;

        let champions = HashMap::new();

        // Manually kill the monster and set its death time to be in the past
        // to ensure its `can_respawn()` method will return true.
        let monster = manager.active_monsters.get_mut(&monster_id).unwrap();
        monster.state = MonsterState::Dead;
        let respawn_duration = monster.respawn_timer;
        monster.death_time =
            Some(std::time::Instant::now() - respawn_duration - std::time::Duration::from_secs(1));

        let next_id = manager.next_instance_id;

        // Call the update loop
        manager.update(&mut board, &champions);

        // The old monster should be gone, and a new one should exist.
        assert!(
            manager.active_monsters.get(&monster_id).is_none(),
            "Old monster should be removed"
        );
        assert_eq!(
            manager.active_monsters.len(),
            1,
            "There should be one active monster after respawn"
        );
        assert_eq!(
            manager.next_instance_id,
            next_id + 1,
            "Next instance ID should be incremented"
        );

        // The new monster should exist with the next ID.
        let new_monster = manager
            .active_monsters
            .get(&next_id)
            .expect("New monster should exist with the next ID");
        assert_eq!(new_monster.state, MonsterState::Idle);
        assert_eq!(new_monster.stats.health, new_monster.stats.max_health);
    }
}
