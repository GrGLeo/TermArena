use std::time::Duration;

use super::{animation::AnimationTrait, cell::CellAnimation, Board, Cell, MinionId, PlayerId, TowerId};
use crate::game::cell::Team;

pub mod base;
pub mod champion;
pub mod minion;
pub mod tower;
pub mod projectile;

pub enum AttackAction {
    Melee {
        damage: u16,
        animation: Box<dyn AnimationTrait>
    },
    Projectile {
        damage: u16,
        speed: u32,
        visual: CellAnimation
    }
}

#[derive(Debug, Clone)]
pub enum Target {
    Tower(TowerId),
    Minion(MinionId),
    Champion(PlayerId),
    Base(Team),
}

#[derive(Debug)]
pub struct Stats {
    attack_damage: u16,
    attack_speed: Duration,
    pub health: u16,
    pub max_health: u16,
    armor: u16,
}

pub trait Fighter {
    fn take_damage(&mut self, damage: u16);
    fn can_attack(&mut self) -> Option<AttackAction>;
    fn get_potential_target<'a>(&self, board: &'a Board) -> Option<&'a Cell>;
}

pub fn reduced_damage(damage: u16, armor: u16) -> u16 {
    damage / (1 + (armor / 100))
}
