use std::time::Duration;

use super::{animation::AnimationTrait, Board, Cell, MinionId, PlayerId, TowerId};

pub mod champion;
pub mod tower;
pub mod minion;

pub enum Target {
    Tower(TowerId),
    Minion(MinionId),
    Champion(PlayerId),
}


#[derive(Debug)]
pub struct Stats {
    attack_damage: u16,
    attack_speed: Duration,
    pub health: u16,
    armor: u16,
}

pub trait Fighter {
    fn take_damage(&mut self, damage: u16);
    fn can_attack(&mut self) -> Option<(u16, Box<dyn AnimationTrait>)>;
    fn get_potential_target<'a>(&self, board: &'a Board, range: (u16, u16)) -> Option<&'a Cell>;
}

pub fn reduced_damage(damage: u16, armor: u16) -> u16 {
    damage / (1 + (armor / 100))
}



