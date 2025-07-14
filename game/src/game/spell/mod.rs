use std::fmt::Debug;

use super::{
    Champion,
    cell::{CellAnimation, Team},
    entities::{Target, projectile::GameplayEffect},
    projectile_manager::ProjectileManager,
};
use crate::config::SpellStats;

pub mod fireball;
pub mod freeze_wall;

pub struct ProjectileBlueprint {
    pub projectile_type: ProjectileType,
    pub owner_id: u64,
    pub team_id: Team,
    pub target_id: Option<Target>,
    pub start_pos: (u16, u16),
    pub end_pos: (u16, u16),
    pub speed: u32,
    pub payloads: Vec<GameplayEffect>,
    pub visual_cell_type: CellAnimation,
}

pub enum ProjectileType {
    LockOn,
    SkillShot,
}

pub trait Spell: Send + Sync + Debug + 'static {
    fn id(&self) -> u8;
    fn mana_cost(&self) -> &u16;
    fn cast(
        &mut self,
        caster: &mut Champion,
        caster_damage: u16,
        projectile_manager: &mut ProjectileManager,
    );
    fn clone_box(&self) -> Box<dyn Spell>;
}

pub fn create_spell_from_id(id: u8, stats: SpellStats) -> Box<dyn Spell> {
    match id {
        0 => Box::new(freeze_wall::FreezeWallSpell::new(stats)),
        1 => Box::new(fireball::FireballSpell::new(stats)),
        _ => panic!("Unknown spell ID: {}", id),
    }
}

#[cfg(test)]
mod tests;
