use std::time::{Duration, Instant};

use crate::config::SpellStats;
use crate::game::projectile_manager::ProjectileManager;
use crate::game::{
    Champion,
    cell::CellAnimation,
    entities::{champion::Direction, projectile::GameplayEffect},
};

use super::{ProjectileBlueprint, ProjectileType, Spell};

#[derive(Debug, Clone)]
pub struct FireballSpell {
    last_casted: Option<Instant>,
    stats: SpellStats,
}

impl FireballSpell {
    pub fn new(spell_stats: SpellStats) -> FireballSpell {
        FireballSpell {
            last_casted: None,
            stats: spell_stats,
        }
    }
}

impl Spell for FireballSpell {
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

        let (proj_start_row, proj_start_col) = match caster.direction {
            Direction::Up => (caster.row.saturating_sub(1), caster.col),
            Direction::Down => (caster.row.saturating_add(1), caster.col),
            Direction::Left => (caster.row, caster.col.saturating_sub(1)),
            Direction::Right => (caster.row, caster.col.saturating_add(1)),
        };

        let (proj_end_row, proj_end_col) = match caster.direction {
            Direction::Up => (caster.row.saturating_sub(self.stats.range), caster.col),
            Direction::Down => (caster.row.saturating_add(self.stats.range), caster.col),
            Direction::Left => (caster.row, caster.col.saturating_sub(self.stats.range)),
            Direction::Right => (caster.row, caster.col.saturating_add(self.stats.range)),
        };

        let blueprint = ProjectileBlueprint {
            projectile_type: ProjectileType::SkillShot,
            owner_id: caster.player_id as u64,
            team_id: caster.team_id,
            target_id: None,
            start_pos: (proj_start_row, proj_start_col),
            end_pos: (proj_end_row, proj_end_col),
            speed: self.stats.speed,
            payloads: vec![GameplayEffect::Damage(spell_damage)],
            visual_cell_type: CellAnimation::FireBall,
        };
        projectile_manager.create_from_blueprint(blueprint);
    }
}
