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
pub struct HealingWaveSpell {
    last_casted: Option<Instant>,
    stats: SpellStats,
}

impl HealingWaveSpell {
    pub fn new(spell_stats: SpellStats) -> HealingWaveSpell {
        HealingWaveSpell {
            last_casted: None,
            stats: spell_stats,
        }
    }
}

impl Spell for HealingWaveSpell {
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
        if let Some(last_casted) = self.last_casted {
            if last_casted.elapsed() < Duration::from_secs(self.stats.cooldown_secs as u64) {
                return;
            }
        }
        if caster.stats.mana < self.stats.mana_cost {
            return;
        } else {
            caster.stats.mana -= self.stats.mana_cost;
        }

        self.last_casted = Some(Instant::now());

        let heal_amount =
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
            payloads: vec![GameplayEffect::Heal(heal_amount)],
            visual_cell_type: CellAnimation::Heal, // Placeholder visual
        };
        projectile_manager.create_from_blueprint(blueprint);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::{ChampionStats, SpellStats};
    use crate::game::cell::Team;
    use crate::game::entities::champion::Champion;
    use crate::game::projectile_manager::ProjectileManager;
    use std::collections::HashMap;

    fn create_default_champion_stats() -> ChampionStats {
        ChampionStats {
            attack_damage: 20,
            attack_speed_ms: 2500,
            health: 200,
            mana: 100,
            armor: 5,
            xp_per_level: vec![
                35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 100, 105, 110, 115,
            ],
            level_up_health_increase: 20,
            level_up_attack_damage_increase: 5,
            level_up_armor_increase: 2,
            attack_range_row: 3,
            attack_range_col: 3,
        }
    }

    #[test]
    fn test_healing_wave_spell() {
        let mut projectile_manager = ProjectileManager::new();
        let champion_stats = create_default_champion_stats();
        let spell_stats = SpellStats {
            id: 2,
            mana_cost: 30,
            cooldown_secs: 15,
            range: 8,
            speed: 0,
            width: 3,
            damage_ratio: 0.0,
            base_damage: 20,
            stun_duration: None,
            is_heal: Some(true),
        };

        let mut champion = Champion::new(1, Team::Red, 5, 5, champion_stats, HashMap::new());
        champion.stats.health = 100;
        champion.stats.mana = 100;

        let mut spell = HealingWaveSpell::new(spell_stats);

        spell.cast(&mut champion, 0, &mut projectile_manager);

        assert_eq!(projectile_manager.projectiles.len(), 1);
        let projectile = &projectile_manager.projectiles[&0];
        assert_eq!(projectile.payloads.len(), 1);
        assert_eq!(projectile.payloads[0], GameplayEffect::Heal(20));
    }
}
