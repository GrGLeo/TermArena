use std::time::Duration;

use super::{animation::{melee::MeleeAnimation, AnimationTrait}, Board, Cell, MinionId, PlayerId, TowerId};

pub mod champion;
pub mod tower;

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
    fn scan_range<'a>(&self, board: &'a Board) -> Option<&'a Cell>;
}
