use std::time::{Duration, Instant};

use crate::game::{cell::Team, MinionId};

use super::Stats;

enum Lane {
    Top,
    Mid,
    Bottom,
}

pub struct Minion {
    pub minion_id: MinionId,
    pub team_id: Team,
    lane: Lane,
    stats: Stats,
    last_attacked: Instant,
    pub row: u16,
    pub col: u16,
}

impl Minion {
    pub fn new(minion_id: MinionId, team_id: Team, lane: Lane, row: u16, col: u16) -> Self {
        let stats = Stats {
            attack_damage: 6,
            attack_speed: Duration::from_millis(2500),
            health: 40,
            armor: 8,
        };

        // TODO: calculate row and bol based on team_id and Lane

        Self {
            minion_id,
            team_id,
            lane,
            stats,
            last_attacked: Instant::now(),
            row,
            col,
        }
    }
}
