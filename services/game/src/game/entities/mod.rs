use std::time::Duration;

use projectile::GameplayEffect;

use super::{
    Board, Cell, MinionId, PlayerId, TowerId, animation::AnimationTrait, cell::CellAnimation,
};
use crate::game::cell::Team;

pub mod base;
pub mod champion;
pub mod item;
pub mod minion;
pub mod monster;
pub mod projectile;
pub mod tower;

pub enum AttackAction {
    Melee {
        animation: Box<dyn AnimationTrait>,
        effects: Vec<GameplayEffect>,
    },
    Projectile {
        effects: Vec<GameplayEffect>,
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
    pub attack_damage: u16,
    attack_speed: Duration,
    pub magic_power: u16,
    pub health: u16,
    pub max_health: u16,
    pub hp_per_sec: f32,
    pub health_regen_acc: f32,
    pub mana_regen_acc: f32,
    pub mana: u16,
    pub max_mana: u16,
    pub mp_per_sec: f32,
    pub armor: u16,
    pub magic_resistance: u16,
}

pub trait Fighter {
    fn take_effect(&mut self, effects: Vec<GameplayEffect>);
    fn can_attack(&mut self) -> Option<AttackAction>;
    fn get_potential_target<'a>(&self, board: &'a Board) -> Option<Cell>;
}

pub fn reduced_damage(damage: u16, armor: u16, magic_resistance: u16, is_magic: bool) -> u16 {
    if !is_magic{
        damage / (1 + (armor / 100))
    } else {
        damage / (1 + (magic_resistance / 100))
    }
}
