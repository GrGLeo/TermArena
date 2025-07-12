use crate::config::BaseStats;
use crate::game::Board;
use crate::game::Cell;
use crate::game::cell::Team;
use crate::game::entities::{Fighter, Stats};

use super::projectile::GameplayEffect;
use super::AttackAction;

pub struct Base {
    pub team: Team,
    pub stats: Stats,
    pub position: (i32, i32),
}

impl Base {
    pub fn new(team: Team, position: (i32, i32), base_stats: BaseStats) -> Self {
        let stats = Stats {
            attack_damage: 0,
            attack_speed: std::time::Duration::from_secs(999),
            health: base_stats.health,
            max_health: base_stats.health,
            armor: base_stats.armor,
        };

        Base {
            team,
            stats,
            position,
        }
    }
}

impl Fighter for Base {
    fn take_effect(&mut self, effect: GameplayEffect) {
        match effect {
            GameplayEffect::Damage(damage) => self.stats.health = self.stats.health.saturating_sub(damage),
            GameplayEffect::Stun(damage, .. ) => self.stats.health = self.stats.health.saturating_sub(damage),
        }
    }

    fn can_attack(&mut self) -> Option<AttackAction> {
        // Base can't attack
        None
    }

    fn get_potential_target<'a>(&self, _board: &'a Board) -> Option<&'a Cell> {
        // Base can't get potential target
        None
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::BaseStats;
    use crate::game::cell::Team;
    use crate::game::entities::projectile::GameplayEffect;

    fn create_default_base_stats() -> BaseStats {
        BaseStats {
            health: 5000,
            armor: 10,
        }
    }

    #[test]
    fn test_new_base() {
        let base_stats = create_default_base_stats();
        let base = Base::new(Team::Red, (10, 10), base_stats);
        assert_eq!(base.team, Team::Red);
        assert_eq!(base.stats.health, 5000);
        assert_eq!(base.position, (10, 10));
    }

    #[test]
    fn test_take_damage() {
        let base_stats = create_default_base_stats();
        let mut base = Base::new(Team::Red, (10, 10), base_stats);
        base.take_effect(GameplayEffect::Damage(100));
        assert_eq!(base.stats.health, 4900);

        base.take_effect(GameplayEffect::Damage(5000));
        assert_eq!(base.stats.health, 0);

        base.take_effect(GameplayEffect::Damage(100));
        assert_eq!(base.stats.health, 0);
    }
}
