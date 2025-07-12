use super::{cell::{CellAnimation, Team}, entities::{projectile::GameplayEffect, Target}};

pub mod aoe;
pub mod freeze_wall;

pub struct ProjectileBlueprint {
    pub projectile_type: ProjectileType,
    pub owner_id: u64,
    pub team_id: Team,
    pub target_id: Option<Target>,
    pub start_pos: (u16, u16),
    pub end_pos: (u16, u16),
    pub speed: u32,
    pub payload: GameplayEffect,
    pub visual_cell_type: CellAnimation,
}

pub enum ProjectileType {
    LockOn,
    SkillShot,
}

