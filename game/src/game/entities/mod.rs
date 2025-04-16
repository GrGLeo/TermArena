use std::time::Duration;

use super::{Board, Cell};

pub mod champion;
pub mod tower;

#[derive(Debug)]
pub struct Stats {
    attack_damage: u8,
    attack_speed: Duration,
    health: u16,
    armor: u8,
}

pub trait Fighter {
    fn take_damage(&mut self, damage: u8);
    fn attack(&self, target: &mut dyn Fighter);
    fn scan_range<'a>(&self, board: &'a Board) -> Vec<&'a Cell>;
}
