use crate::config::SpellStats;
use crate::game::buffs::stun_buff::StunBuff;
use crate::game::{
    Champion,
    cell::CellAnimation,
    entities::{champion::Direction, projectile::GameplayEffect},
};

use super::{ProjectileBlueprint, ProjectileType};

pub fn cast_freeze_wall(
    caster: &Champion,
    caster_damage: u16,
    spell_stats: &SpellStats,
) -> Vec<ProjectileBlueprint> {
    let mut blueprints = Vec::new();
    let spell_damage =
        (caster_damage as f32 * spell_stats.damage_ratio + spell_stats.base_damage as f32) as u16;

    let (wall_center_row, wall_center_col) = match caster.direction {
        Direction::Up => (caster.row.saturating_sub(1), caster.col),
        Direction::Down => (caster.row.saturating_add(1), caster.col),
        Direction::Left => (caster.row, caster.col.saturating_sub(1)),
        Direction::Right => (caster.row, caster.col.saturating_add(1)),
    };

    for i in 0..spell_stats.width {
        let offset = i as i16 - (spell_stats.width / 2) as i16;
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
                proj_start_row.saturating_sub(spell_stats.range),
                proj_start_col,
            ),
            Direction::Down => (
                proj_start_row.saturating_add(spell_stats.range),
                proj_start_col,
            ),
            Direction::Left => (
                proj_start_row,
                proj_start_col.saturating_sub(spell_stats.range),
            ),
            Direction::Right => (
                proj_start_row,
                proj_start_col.saturating_add(spell_stats.range),
            ),
        };
        let payloads = vec![
            GameplayEffect::Damage(spell_damage),
            GameplayEffect::Buff(Box::new(StunBuff::new(spell_stats.stun_duration as u64))),
        ];

        let blueprint = ProjectileBlueprint {
            projectile_type: ProjectileType::SkillShot,
            owner_id: caster.player_id as u64,
            team_id: caster.team_id,
            target_id: None,
            start_pos: (proj_start_row, proj_start_col),
            end_pos: (proj_end_row, proj_end_col),
            speed: spell_stats.speed,
            payloads,
            visual_cell_type: CellAnimation::FreezeWall,
        };
        blueprints.push(blueprint)
    }
    return blueprints;
}

#[cfg(test)]
mod tests {
    use std::collections::HashMap;

    use super::*;
    use crate::config::ChampionStats;
    use crate::game::{Champion, cell::Team};

    fn create_spell_stats() -> SpellStats {
        SpellStats {
            range: 10,
            speed: 1,
            width: 5,
            damage_ratio: 0.8,
            base_damage: 20,
            stun_duration: 1,
        }
    }

    fn create_test_champion(
        row: u16,
        col: u16,
        direction: Direction,
        attack_damage: u16,
    ) -> Champion {
        let champion_stats = ChampionStats {
            attack_damage,
            attack_speed_ms: 1000,
            health: 500,
            armor: 10,
            xp_per_level: vec![100],
            level_up_health_increase: 50,
            level_up_attack_damage_increase: 10,
            level_up_armor_increase: 5,
            attack_range_row: 1,
            attack_range_col: 1,
        };
        let mut champion = Champion::new(1, Team::Red, row, col, champion_stats, HashMap::new());
        champion.direction = direction;
        champion
    }

    #[test]
    fn test_cast_freeze_wall_up() {
        let caster = create_test_champion(20, 20, Direction::Up, 100);
        let spell_stats = create_spell_stats();
        let blueprints = cast_freeze_wall(&caster, 100, &spell_stats);

        assert_eq!(blueprints.len(), 5);

        // Wall should be centered at (10, 20), extending horizontally
        let expected_center_row = 19 as u16;
        let expected_center_col = 20 as u16;

        for (i, blueprint) in blueprints.iter().enumerate() {
            let offset = i as i16 - 2;
            let expected_col = expected_center_col.saturating_add_signed(offset);
            assert_eq!(blueprint.start_pos, (expected_center_row, expected_col));
            assert_eq!(
                blueprint.end_pos,
                (expected_center_row.saturating_sub(10), expected_col)
            );
            assert_eq!(blueprint.payloads.len(), 2);
            assert!(matches!(blueprint.payloads[0], GameplayEffect::Damage(_)));
            assert!(matches!(blueprint.payloads[1], GameplayEffect::Buff(_)));
        }
    }

    #[test]
    fn test_cast_freeze_wall_down() {
        let caster = create_test_champion(20, 20, Direction::Down, 100);
        let spell_stats = create_spell_stats();
        let blueprints = cast_freeze_wall(&caster, 100, &spell_stats);

        assert_eq!(blueprints.len(), 5);

        // Wall should be centered at (30, 20), extending horizontally
        let expected_center_row = 21 as u16;
        let expected_center_col = 20 as u16;

        for (i, blueprint) in blueprints.iter().enumerate() {
            let offset = i as i16 - 2;
            let expected_col = expected_center_col.saturating_add_signed(offset);
            assert_eq!(blueprint.start_pos, (expected_center_row, expected_col));
            assert_eq!(
                blueprint.end_pos,
                (expected_center_row.saturating_add(10), expected_col)
            );
        }
    }

    #[test]
    fn test_cast_freeze_wall_left() {
        let caster = create_test_champion(20, 20, Direction::Left, 100);
        let spell_stats = create_spell_stats();
        let blueprints = cast_freeze_wall(&caster, 100, &spell_stats);

        assert_eq!(blueprints.len(), 5);

        // Wall should be centered at (20, 10), extending vertically
        let expected_center_row = 20 as u16;
        let expected_center_col = 19 as u16;

        for (i, blueprint) in blueprints.iter().enumerate() {
            let offset = i as i16 - 2;
            let expected_row = expected_center_row.saturating_add_signed(offset);
            assert_eq!(blueprint.start_pos, (expected_row, expected_center_col));
            assert_eq!(
                blueprint.end_pos,
                (expected_row, expected_center_col.saturating_sub(10))
            );
            assert_eq!(blueprint.payloads.len(), 2);
            assert!(matches!(blueprint.payloads[0], GameplayEffect::Damage(_)));
            assert!(matches!(blueprint.payloads[1], GameplayEffect::Buff(_)));
        }
    }

    #[test]
    fn test_cast_freeze_wall_right() {
        let caster = create_test_champion(20, 20, Direction::Right, 100);
        let spell_stats = create_spell_stats();
        let blueprints = cast_freeze_wall(&caster, 100, &spell_stats);

        assert_eq!(blueprints.len(), 5);

        // Wall should be centered at (20, 30), extending vertically
        let expected_center_row = 20 as u16;
        let expected_center_col = 21 as u16;

        for (i, blueprint) in blueprints.iter().enumerate() {
            let offset = i as i16 - 2;
            let expected_row = expected_center_row.saturating_add_signed(offset);
            assert_eq!(blueprint.start_pos, (expected_row, expected_center_col));
            assert_eq!(
                blueprint.end_pos,
                (expected_row, expected_center_col.saturating_add(10))
            );
            assert_eq!(blueprint.payloads.len(), 2);
            assert!(matches!(blueprint.payloads[0], GameplayEffect::Damage(_)));
            assert!(matches!(blueprint.payloads[1], GameplayEffect::Buff(_)));
        }
    }

    #[test]
    fn test_cast_freeze_wall_edge_case_top_left() {
        // Caster is at (1, 1), casting Up. Wall should be at (0, 5)
        let caster = create_test_champion(5, 5, Direction::Up, 100);
        let spell_stats = create_spell_stats();
        let blueprints = cast_freeze_wall(&caster, 100, &spell_stats);

        assert_eq!(blueprints.len(), 5);
        let expected_row = 4; // 5 - 1 saturates at 4
        for (i, blueprint) in blueprints.iter().enumerate() {
            let offset = i as i16 - 2;
            let expected_col = 5i16.saturating_add(offset) as u16;
            assert_eq!(blueprint.start_pos, (expected_row, expected_col));
            assert_eq!(blueprint.payloads.len(), 2);
            assert!(matches!(blueprint.payloads[0], GameplayEffect::Damage(_)));
            assert!(matches!(blueprint.payloads[1], GameplayEffect::Buff(_)));
        }

        // Caster is at (1, 1), casting Left. Wall should be at (5, 0)
        let caster = create_test_champion(5, 5, Direction::Left, 100);
        let blueprints = cast_freeze_wall(&caster, 100, &spell_stats);

        assert_eq!(blueprints.len(), 5);
        let expected_col = 4; // 5 - 1 saturates at 4
        for (i, blueprint) in blueprints.iter().enumerate() {
            let offset = i as i16 - 2;
            let expected_row = 5i16.saturating_add(offset) as u16;
            assert_eq!(blueprint.start_pos, (expected_row, expected_col));
            assert_eq!(blueprint.payloads.len(), 2);
            assert!(matches!(blueprint.payloads[0], GameplayEffect::Damage(_)));
            assert!(matches!(blueprint.payloads[1], GameplayEffect::Buff(_)));
        }
    }
}
