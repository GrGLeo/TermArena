use crate::game::cell::Team;
use crate::game::entities::{Fighter, Stats};
use crate::game::Board;
use crate::game::Cell;
use crate::game::animation::AnimationTrait;

pub struct Base {
    pub team: Team,
    pub stats: Stats,
    pub position: (i32, i32),
}

impl Base {
    pub fn new(team: Team, position: (i32, i32)) -> Self {
        let stats = Stats {
            attack_damage: 0,
            attack_speed: std::time::Duration::from_secs(999),
            health: 5000,
            max_health: 5000,
            armor: 20,
        };

        Self {
            team,
            stats,
            position,
        }
    }
}

impl Fighter for Base {
    fn take_damage(&mut self, damage: u16) {
        self.stats.health = self.stats.health.saturating_sub(damage);
    }

    fn can_attack(&mut self) -> Option<(u16, Box<dyn AnimationTrait>)> {
        None
    }

    fn get_potential_target<'a>(&self, _board: &'a Board, _range: (u16, u16)) -> Option<&'a Cell> {
        None
    }
}