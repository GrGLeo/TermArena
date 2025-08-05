use std::time::Duration;

use projectile::GameplayEffect;

use super::{
    Board, Cell, MinionId, PlayerId, TowerId, animation::AnimationTrait, cell::CellAnimation,
};
use crate::game::cell::Team;

pub mod base;
pub mod champion;
pub mod minion;
pub mod monster;
pub mod projectile;
pub mod tower;
pub mod item;

pub enum AttackAction {
    Melee {
        damage: u16,
        animation: Box<dyn AnimationTrait>,
    },
    Projectile {
        damage: u16,
        speed: u32,
        visual: CellAnimation,
    },
}

#[derive(Debug, Clone, PartialEq)]
pub enum Target {
    Tower(TowerId),
    Minion(MinionId),
    Champion(PlayerId),
    Base(Team),
    Monster(MinionId),
}

#[derive(Debug)]
pub struct Stats {
    attack_damage: u16,
    attack_speed: Duration,
    pub health: u16,
    pub max_health: u16,
    pub mana: u16,
    pub max_mana: u16,
    armor: u16,
}

pub trait Fighter {
    fn take_effect(&mut self, effects: Vec<GameplayEffect>);
    fn can_attack(&mut self) -> Option<AttackAction>;
    fn get_potential_target<'a>(&self, board: &'a Board) -> Option<&'a Cell>;
}

pub fn reduced_damage(damage: u16, armor: u16) -> u16 {
    damage / (1 + (armor / 100))
}
