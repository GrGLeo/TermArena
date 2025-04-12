use crate::entities::common::{Stats, Position};
use std::time::Instant;

pub type PlayerId = String;

#[derive(Debug)]
pub struct Player {
    player_id: PlayerId,
    position: Position,
    stats: Stats,
    last_attack: Instant,
}

impl Player {
    pub fn new(player_id: String, pos: Position) -> Self {
        return Player {
            player_id,
            position: pos,
            stats: Stats::default_player(),
            last_attack: Instant::now(),
        }
    }

    pub fn take_damage(&mut self, incoming_damage: u8) {
        let reduced_damage = if incoming_damage > self.stats.armor {
            incoming_damage - self.stats.armor
        } else {
            0
        };

        self.stats.health -= reduced_damage;
    }

    pub fn is_attack_ready(&self) -> bool {
        let now = Instant::now();
        if now.duration_since(self.last_attack) >= self.stats.attack_speed {
            true
        } else {
            false
        }
    }

}
