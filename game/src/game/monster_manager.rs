use crate::config::MonsterStats;
use crate::game::entities::monster::Monster;
use std::collections::HashMap;

use super::cell::MonsterId;
use super::entities::projectile::GameplayEffect;
use super::entities::Fighter;
use super::PlayerId;

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

    pub fn spawn_monster(&mut self, name_id: &str) {
        if let Some(monster_def) = self.monster_definitions.get(name_id) {
            let monster = Monster::new(self.next_instance_id, monster_def.clone());
            self.active_monsters.insert(self.next_instance_id, monster);
            self.next_instance_id += 1;
        } else {
            ()
        }
    }

    pub fn apply_effects_to_monster(&mut self,monster_id: &MonsterId, effects: Vec<GameplayEffect>, player_id: PlayerId) {
        if let Some(monster) = self.active_monsters.get_mut(monster_id) {
            monster.take_effect(effects);
            monster.attach_target(player_id);
        } else {
            // For now we do nothing
            ()
        }
    }

}

#[cfg(test)]
mod tests {
    use crate::{config::ChampionStats, game::{cell::Team, entities::monster::MonsterState, Board, Champion}};

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

        // Spawn the monster
        manager.spawn_monster("wolf_red");

        // Check that there is one active monster
        assert_eq!(manager.active_monsters.len(), 1, "Should be one active monster");

        // Check that the next ID has been incremented
        assert_eq!(manager.next_instance_id, 2, "Next instance ID should be 2");

        // Get the monster and verify its properties
        let monster = manager.active_monsters.get(&1).expect("Monster with ID 1 should exist");
        assert_eq!(monster.id, 1);
        assert_eq!(monster.monster_id, "wolf_red");
        assert_eq!(monster.spawn_row, 10, "Monster should spawn at the definition's coordinates");
        assert_eq!(monster.spawn_col, 10);
        assert_eq!(manager.next_instance_id, 2, "Next instance ID should be 2");

        // Get the monster and verify its properties
        let monster = manager.active_monsters.get(&1).expect("Monster with ID 1 should exist");
        assert_eq!(monster.id, 1);
        assert_eq!(monster.monster_id, "wolf_red");
        assert_eq!(monster.spawn_row, 10, "Monster should spawn at the definition's coordinates");
        assert_eq!(monster.spawn_col, 10);
        assert_eq!(monster.row, 10);
        assert_eq!(monster.col, 10);
    }

    #[test]
    fn test_apply_effects_sets_aggro_on_idle_monster() {
        let monster_defs = vec![create_test_monster_stats("wolf_red", 10, 10)];
        let mut manager = MonsterManager::new(monster_defs);
        manager.spawn_monster("wolf_red");

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
        manager.spawn_monster("wolf_red");

        let monster_id = 1;
        let attacker_1 = 42; // First attacker
        let attacker_2 = 99; // Second attacker

        // First attack sets the aggro
        manager.apply_effects_to_monster(&monster_id, vec![GameplayEffect::Damage(10)], attacker_1);
        let monster = manager.active_monsters.get(&monster_id).unwrap();
        assert_eq!(monster.target_champion_id, Some(attacker_1), "Target should be the first attacker");
        assert_eq!(monster.stats.health, 90);

        // Second attack from a different champion
        manager.apply_effects_to_monster(&monster_id, vec![GameplayEffect::Damage(10)], attacker_2);
        let monster = manager.active_monsters.get(&monster_id).unwrap();

        // Verify health is reduced, but target remains unchanged
        assert_eq!(monster.stats.health, 80, "Health should be further reduced");
        assert_eq!(monster.target_champion_id, Some(attacker_1), "Target should NOT change to the second attacker");
    }

    #[test]
    fn test_update_leashes_monster_when_far_from_spawn() {
        // Leash range in test stats is 10. Spawn is (10, 10).
        let monster_defs = vec![create_test_monster_stats("wolf_red", 10, 10)];
        let mut manager = MonsterManager::new(monster_defs);
        manager.spawn_monster("wolf_red");
        let monster_id = 1;
        let attacker_id = 42;

        let board = Board::new(100, 100);
        let mut champions = HashMap::new();
        champions.insert(attacker_id, create_champion(15, 15)); // Champion position is irrelevant for the leash calculation itself

        // Make the monster aggro
        manager.apply_effects_to_monster(&monster_id, vec![], attacker_id);
        
        // Manually move the monster far from its spawn point to simulate it being kited
        let monster = manager.active_monsters.get_mut(&monster_id).unwrap();
        monster.row = 21; // This is 11 units away from spawn row 10, exceeding leash range of 10
        monster.col = 10;
        assert_eq!(monster.state, MonsterState::Aggro, "Monster should be aggro initially");

        // Call the update loop
        manager.update(&board, &champions);

        // Verify the monster is now returning because it's too far from its spawn
        let monster = manager.active_monsters.get(&monster_id).unwrap();
        assert_eq!(monster.state, MonsterState::Returning, "Monster should be returning after being leashed");
        assert!(monster.target_champion_id.is_none(), "Monster target should be cleared when returning");
    }
}

