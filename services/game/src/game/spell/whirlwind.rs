use std::time::{Duration, Instant};

use crate::{
    config::SpellStats,
    game::{
        Champion, cell::CellAnimation, entities::projectile::GameplayEffect,
        projectile_manager::ProjectileManager,
    },
};

use super::{ProjectileBlueprint, ProjectileType, Spell};

#[derive(Debug, Clone)]
pub struct WhirlwindSpell {
    last_casted: Option<Instant>,
    stats: SpellStats,
}

impl WhirlwindSpell {
    pub fn new(spell_stats: SpellStats) -> WhirlwindSpell {
        WhirlwindSpell {
            last_casted: None,
            stats: spell_stats,
        }
    }
}

impl Spell for WhirlwindSpell {
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

        let spell_damage =
            (caster_damage as f32 * self.stats.damage_ratio + self.stats.base_attack_damage as f32) as u16;

        let blueprint = ProjectileBlueprint {
            projectile_type: ProjectileType::Rotationnary,
            owner_id: caster.player_id as u64,
            team_id: caster.team_id,
            target_id: None,
            start_pos: (0, 0),
            end_pos: (0, 0),
            radius: Some(1),
            total_iteration: Some(8),
            speed: self.stats.speed,
            payloads: vec![GameplayEffect::PhysicalDamage(spell_damage)],
            visual_cell_type: CellAnimation::FireBall,
        };
        projectile_manager.create_from_blueprint(blueprint);
    }
}
