use std::time::{Duration, Instant};

use crate::config::SpellStats;
use crate::game::projectile_manager::ProjectileManager;
use crate::game::{
    Champion,
    cell::CellAnimation,
    entities::projectile::GameplayEffect,
};

use super::{ProjectileBlueprint, ProjectileType, Spell};

#[derive(Debug, Clone)]
pub struct ElectricPulseSpell {
    last_casted: Option<Instant>,
    stats: SpellStats,
}

impl ElectricPulseSpell {
    pub fn new(spell_stats: SpellStats) -> ElectricPulseSpell {
        ElectricPulseSpell {
            last_casted: None,
            stats: spell_stats,
        }
    }
}

impl Spell for ElectricPulseSpell {
    fn id(&self) -> u8 {
        self.stats.id
    }

    fn mana_cost(&self) -> &u16 {
        &self.stats.mana_cost
    }

    fn cast_time_ms(&self) -> u16 {
        self.stats.cast_time_ms
    }

    fn clone_box(&self) -> Box<dyn Spell> {
        Box::new(self.clone())
    }

    fn cast(
        &mut self,
        caster: &mut Champion,
        _caster_damage: u16,
        _caster_magic_power: u16,
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

        // Check if target exists
        if let Some(target) = &caster.target {
            let blueprint = ProjectileBlueprint {
                projectile_type: ProjectileType::LockOn,
                owner_id: caster.player_id as u64,
                team_id: caster.team_id,
                target_id: Some(target.clone()),
                start_pos: (caster.row, caster.col),
                end_pos: (0, 0), // Not used for LockOn
                radius: None,
                total_iteration: None,
                speed: self.stats.speed,
                payloads: vec![GameplayEffect::Buff(Box::new(crate::game::buffs::stun_buff::StunBuff::new(Duration::from_millis(500))))],
                visual_cell_type: CellAnimation::TowerHit,
            };
            projectile_manager.create_from_blueprint(blueprint);
        }
    }
}