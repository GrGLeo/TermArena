use std::time::{Duration, Instant};

use crate::config::SpellStats;
use crate::game::buffs::stun_buff::StunBuff;
use crate::game::projectile_manager::ProjectileManager;
use crate::game::{
    Champion,
    cell::CellAnimation,
    entities::{champion::Direction, projectile::GameplayEffect},
};

use super::{ProjectileBlueprint, ProjectileType, Spell};

#[derive(Debug, Clone)]
pub struct FreezeWallSpell {
    last_casted: Option<Instant>,
    stats: SpellStats,
}

impl FreezeWallSpell {
    pub fn new(spell_stats: SpellStats) -> FreezeWallSpell {
        FreezeWallSpell {
            last_casted: None,
            stats: spell_stats,
        }
    }
}

impl Spell for FreezeWallSpell {
    fn id(&self) -> u8 {
        self.stats.id
    }

    fn mana_cost(&self) -> &u16 {
        &self.stats.mana_cost
    }

    fn clone_box(&self) -> Box<dyn Spell> {
        Box::new(self.clone())
    }

    fn cast(
        &mut self,
        caster: &mut Champion,
        caster_damage: u16,
        projectile_manager: &mut ProjectileManager,
    ) {
        // TODO: return Err maybe instead of empty Vec
        // Cooldown check
        if let Some(last_casted) = self.last_casted {
            if last_casted.elapsed() < Duration::from_secs(self.stats.cooldown_secs as u64) {
                return ();
            }
        }
        // Mana check
        if caster.stats.mana < self.stats.mana_cost {
            return ();
        } else {
            caster.stats.mana -= self.stats.mana_cost;
        }

        self.last_casted = Some(Instant::now());

        let spell_damage =
            (caster_damage as f32 * self.stats.damage_ratio + self.stats.base_damage as f32) as u16;

        let (wall_center_row, wall_center_col) = match caster.direction {
            Direction::Up => (caster.row.saturating_sub(1), caster.col),
            Direction::Down => (caster.row.saturating_add(1), caster.col),
            Direction::Left => (caster.row, caster.col.saturating_sub(1)),
            Direction::Right => (caster.row, caster.col.saturating_add(1)),
        };

        for i in 0..self.stats.width {
            let offset = i as i16 - (self.stats.width / 2) as i16;
            let (proj_start_row, proj_start_col) = match caster.direction {
                Direction::Up | Direction::Down => (
                    wall_center_row,
                    wall_center_col.saturating_add_signed(offset),
                ),
                Direction::Left | Direction::Right => (
                    wall_center_row.saturating_add_signed(offset),
                    wall_center_col,
                ),
            };

            let (proj_end_row, proj_end_col) = match caster.direction {
                Direction::Up => (
                    proj_start_row.saturating_sub(self.stats.range),
                    proj_start_col,
                ),
                Direction::Down => (
                    proj_start_row.saturating_add(self.stats.range),
                    proj_start_col,
                ),
                Direction::Left => (
                    proj_start_row,
                    proj_start_col.saturating_sub(self.stats.range),
                ),
                Direction::Right => (
                    proj_start_row,
                    proj_start_col.saturating_add(self.stats.range),
                ),
            };
            // Once build.rs is done we would check that stun duration is initalized
            // We will then always be sure to have Some(duration)
            let mut payloads: Vec<GameplayEffect> = Vec::new();
            if let Some(duration) = self.stats.stun_duration {
                payloads = vec![
                    GameplayEffect::Damage(spell_damage),
                    GameplayEffect::Buff(Box::new(StunBuff::new(duration as u64))),
                ];
            };

            let blueprint = ProjectileBlueprint {
                projectile_type: ProjectileType::SkillShot,
                owner_id: caster.player_id as u64,
                team_id: caster.team_id,
                target_id: None,
                start_pos: (proj_start_row, proj_start_col),
                end_pos: (proj_end_row, proj_end_col),
                speed: self.stats.speed,
                payloads,
                visual_cell_type: CellAnimation::FreezeWall,
            };
            projectile_manager.create_from_blueprint(blueprint);
        }
    }
}
