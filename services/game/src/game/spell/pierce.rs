use std::time::{Duration, Instant};

use crate::config::SpellStats;
use crate::game::buffs::health_buff::DoTBuff;
use crate::game::projectile_manager::ProjectileManager;
use crate::game::{Champion, entities::projectile::GameplayEffect};

use super::Spell;

#[derive(Debug, Clone)]
pub struct PierceSpell {
    last_casted: Option<Instant>,
    stats: SpellStats,
}

impl PierceSpell {
    pub fn new(spell_stats: SpellStats) -> PierceSpell {
        PierceSpell {
            last_casted: None,
            stats: spell_stats,
        }
    }
}

impl Spell for PierceSpell {
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
        _: u16,
        _projectile_manager: &mut ProjectileManager,
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

        let damage_per_sec = (caster_damage as f32 * self.stats.damage_ratio
            + self.stats.base_attack_damage as f32) as u16;
        let dot_buff = DoTBuff::new(
            self.stats.effect_duration.unwrap_or(0) as u64,
            damage_per_sec as u8,
        );
        let effect = GameplayEffect::Buff(Box::new(dot_buff));
        caster.reset_aa();
        caster.add_effects(effect);
    }
}
