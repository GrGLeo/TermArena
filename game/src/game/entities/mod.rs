use std::time::Duration;

use super::{animation::Animation, Board, Cell, MinionId, PlayerId, TowerId};

pub mod champion;
pub mod tower;

pub enum Target {
    Tower(TowerId),
    Minion(MinionId),
    Champion(PlayerId),
}


#[derive(Debug)]
pub struct Stats {
    attack_damage: u8,
    attack_speed: Duration,
    pub health: u16,
    armor: u8,
}

pub trait Fighter {
    fn take_damage(&mut self, damage: u8);
    fn can_attack(&mut self) -> Option<(u8, Box<dyn Animation + '_>)>;
    fn scan_range<'a>(&self, board: &'a Board) -> Option<&'a Cell>;
}
