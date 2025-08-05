use std::{
    collections::{HashMap, VecDeque},
    time::{Duration, Instant},
};

use crate::{
    config::MonsterStats,
    game::{
        Board, PlayerId, algorithms::pathfinding::find_path_on_board,
        animation::melee::MeleeAnimation, buffs::Buff, cell::MonsterId, entities::AttackAction,
    },
};

use super::{Fighter, Stats, projectile::GameplayEffect, reduced_damage};

#[derive(PartialEq, Debug)]
pub enum MonsterState {
    Aggro,
    Idle,
    Returning,
    Dead,
}

#[derive(Debug)]
pub struct Monster {
    pub id: MonsterId,
    pub monster_id: String,
    pub state: MonsterState,
    pub target_champion_id: Option<PlayerId>,
    pub path: Option<VecDeque<(u16, u16)>>,
    pub stats: Stats,
    pub last_attacked: Instant,
    stun_timer: Option<Instant>,
    pub active_buffs: HashMap<String, Box<dyn Buff>>,
    pub respawn_timer: Duration,
    pub death_time: Option<Instant>,
    pub row: u16,
    pub col: u16,
    pub spawn_row: u16,
    pub spawn_col: u16,
    pub leash_range: u8,
}

impl Monster {
    pub fn new(id: MonsterId, monster_stats: MonsterStats) -> Monster {
        let stats = Stats {
            attack_damage: monster_stats.attack_damage,
            attack_speed: Duration::from_millis(monster_stats.attack_speed_ms),
            health: monster_stats.health,
            max_health: monster_stats.health,
            mana: 0,
            max_mana: 0,
            armor: monster_stats.armor,
        };

        Monster {
            id,
            monster_id: monster_stats.id,
            state: MonsterState::Idle,
            target_champion_id: None,
            path: None,
            stats,
            last_attacked: Instant::now(),
            stun_timer: None,
            active_buffs: HashMap::new(),
            respawn_timer: Duration::from_secs(monster_stats.respawn_timer_secs as u64),
            death_time: None,
            row: monster_stats.spawn_row,
            col: monster_stats.spawn_col,
            spawn_row: monster_stats.spawn_row,
            spawn_col: monster_stats.spawn_col,
            leash_range: monster_stats.leash_range,
        }
    }

    pub fn attach_target(&mut self, player_id: PlayerId) {
        // A dead monster cannot aggro or acquire a target
        if self.state == MonsterState::Dead {
            return;
        }
        self.state = MonsterState::Aggro;
        match self.target_champion_id {
            Some(_) => {}
            None => self.target_champion_id = Some(player_id),
        }
    }

    pub fn start_returning(&mut self, board: &Board) {
        self.state = MonsterState::Returning;
        self.target_champion_id = None;
        let mut path = find_path_on_board(
            board,
            (self.row, self.col),
            (self.spawn_row, self.spawn_col),
        );
        if let Some(ref mut p) = path {
            p.push_back((self.spawn_row, self.spawn_col));
        }
        self.path = path;
    }

    pub fn reset(&mut self) {
        self.state = MonsterState::Idle;
        self.stats.health = self.stats.max_health;
        self.path = None;
    }

    pub fn can_respawn(&self) -> bool {
        if let Some(death_timer) = self.death_time {
            if death_timer.elapsed() > self.respawn_timer {
                return true;
            } else {
                return false;
            }
        } else {
            // TODO: We need to return an error here, can timer should always be set.
            return false;
        }
    }
}

impl Fighter for Monster {
    fn take_effect(&mut self, effects: Vec<GameplayEffect>) {
        for effect in effects.into_iter() {
            match effect {
                GameplayEffect::Damage(damage) => {
                    let reduced_damage = reduced_damage(damage, self.stats.armor);
                    self.stats.health = self.stats.health.saturating_sub(reduced_damage as u16);
                    if self.stats.health == 0 {
                        self.state = MonsterState::Dead;
                        self.target_champion_id = None;
                        self.death_time = Some(Instant::now());
                    }
                }
                GameplayEffect::Heal(..) => {}
                GameplayEffect::Buff(..) => {}
            };
        }
    }

    fn can_attack(&mut self) -> Option<super::AttackAction> {
        if self.last_attacked + self.stats.attack_speed < Instant::now() {
            self.last_attacked = Instant::now();
            let animation = MeleeAnimation::new(self.id);
            Some(AttackAction::Melee {
                damage: self.stats.attack_damage,
                animation: Box::new(animation),
            })
        } else {
            None
        }
    }

    fn get_potential_target<'a>(
        &self,
        _board: &'a crate::game::Board,
    ) -> Option<&'a crate::game::Cell> {
        // No need for monster
        unimplemented!()
    }
}

#[cfg(test)]
mod tests {
    use crate::{
        config::MonsterStats,
        game::{Board, entities::AttackAction},
    };

    use super::*;
    use std::time::Duration;

    // Helper function to create a default monster definition for tests
    fn create_test_monster_def() -> MonsterStats {
        MonsterStats {
            id: "wolf_test".to_string(),
            spawn_row: 1,
            spawn_col: 1,
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
            attack_speed_ms: 1,
        }
    }

    #[test]
    fn test_new_monster_initial_state() {
        let monster_def = create_test_monster_def();
        let monster = Monster::new(1, monster_def.clone());

        assert_eq!(monster.id, 1);
        assert_eq!(monster.monster_id, "wolf_test");
        assert_eq!(monster.stats.health, 100);
        assert_eq!(monster.row, 1);
        assert_eq!(monster.col, 1);
        assert_eq!(monster.spawn_row, 1);
        assert_eq!(monster.spawn_col, 1);
        assert_eq!(monster.state, MonsterState::Idle);
        assert!(monster.target_champion_id.is_none());

        // Check that last_attack_time is recent, allowing for a small delay
        assert!(monster.last_attacked.elapsed() < Duration::from_secs(1));
    }

    #[test]
    fn test_take_effect_reduces_health() {
        let monster_def = create_test_monster_def();
        let mut monster = Monster::new(1, monster_def);

        monster.take_effect(vec![GameplayEffect::Damage(40)]);

        assert_eq!(monster.stats.health, 60);
        // State should NOT change, as per the new design
        assert_eq!(monster.state, MonsterState::Idle);
        assert!(monster.target_champion_id.is_none());
    }

    #[test]
    fn test_attach_target_sets_aggro() {
        let monster_def = create_test_monster_def();
        let mut monster = Monster::new(1, monster_def);
        let target_id = 25;

        monster.attach_target(target_id);

        assert_eq!(monster.state, MonsterState::Aggro);
        assert_eq!(monster.target_champion_id, Some(target_id));
    }

    #[test]
    fn test_take_effect_handles_death() {
        let monster_def = create_test_monster_def();
        let mut monster = Monster::new(1, monster_def);
        let attacker_id = 42;

        // Set the monster to be aggressive towards a target
        monster.attach_target(attacker_id);
        assert_eq!(monster.state, MonsterState::Aggro);

        // Apply lethal damage (more than its health)
        monster.take_effect(vec![GameplayEffect::Damage(150)]);

        // Verify the monster is dead
        assert_eq!(monster.stats.health, 0);
        assert_eq!(monster.state, MonsterState::Dead);
        assert!(monster.death_time.is_some(), "death_time should be set");

        // Verify the target is cleared upon death
        assert!(
            monster.target_champion_id.is_none(),
            "target should be cleared on death"
        );
    }

    #[test]
    fn test_can_attack_respects_cooldown() {
        let monster_def = create_test_monster_def();
        let mut monster = Monster::new(1, monster_def);

        // 1. Manually expire the cooldown.
        let cooldown = monster.stats.attack_speed;
        monster.last_attacked = Instant::now() - (cooldown + Duration::from_millis(100));

        let attack_action = monster.can_attack();
        assert!(
            attack_action.is_some(),
            "Should be able to attack after cooldown"
        );

        if let Some(AttackAction::Melee { damage, .. }) = attack_action {
            assert_eq!(damage, 10); // From create_test_monster_def
        } else {
            panic!("Expected a Melee attack action");
        }
    }

    #[test]
    fn test_start_returning_calculates_path_to_spawn() {
        let monster_def = create_test_monster_def(); // Spawns at (1, 1)
        let mut monster = Monster::new(1, monster_def);
        let board = Board::new(20, 20); // A clear board for pathfinding
        let target_id = 25;

        // Move the monster away from its spawn
        monster.row = 10;
        monster.col = 10;

        // Make the monster aggressive first
        monster.attach_target(target_id);
        assert_eq!(monster.state, MonsterState::Aggro);
        assert!(monster.path.is_none());

        // Tell the monster to return, providing the board context
        monster.start_returning(&board);

        // Verify state change and target clearing
        assert_eq!(monster.state, MonsterState::Returning);
        assert!(
            monster.target_champion_id.is_none(),
            "Target should be cleared"
        );

        // Verify a path has been calculated
        assert!(
            monster.path.is_some(),
            "Path should be calculated on return"
        );
        let path = monster.path.as_ref().unwrap();
        assert!(!path.is_empty(), "Path should not be empty");

        // The path should lead back to the spawn point
        let final_destination = path.back().unwrap();
        assert_eq!(*final_destination, (monster.spawn_row, monster.spawn_col));
    }

    #[test]
    fn test_reset_monster_restores_state_and_health() {
        let monster_def = create_test_monster_def();
        let mut monster = Monster::new(1, monster_def);
        let board = Board::new(20, 20);

        // Damage the monster and make it return
        monster.take_effect(vec![GameplayEffect::Damage(50)]);
        monster.start_returning(&board);
        assert_eq!(monster.stats.health, 50);
        assert_eq!(monster.state, MonsterState::Returning);
        assert!(monster.path.is_some());

        // Reset the monster
        monster.reset();

        // Verify it's back to a pristine Idle state
        assert_eq!(monster.state, MonsterState::Idle);
        assert!(monster.path.is_none(), "Path should be cleared on reset");
        assert_eq!(
            monster.stats.health, monster.stats.max_health,
            "Health should be fully restored"
        );
    }

    #[test]
    fn test_can_respawn_respects_timer() {
        let monster_def = create_test_monster_def();
        let mut monster = Monster::new(1, monster_def);

        // Kill the monster
        monster.state = MonsterState::Dead;
        monster.death_time = Some(Instant::now());

        // Immediately after death, it should not be able to respawn
        assert!(!monster.can_respawn(), "Should not respawn immediately");

        // Manually set the death time to be far in the past
        let respawn_duration = monster.respawn_timer;
        monster.death_time = Some(Instant::now() - respawn_duration - Duration::from_secs(1));

        // Now it should be able to respawn
        assert!(
            monster.can_respawn(),
            "Should be able to respawn after timer expires"
        );
    }
}
