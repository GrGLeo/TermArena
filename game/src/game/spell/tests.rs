use crate::game::entities::projectile::PathingLogic;
use std::collections::HashMap;

use crate::{
    config::{ChampionStats, SpellStats},
    game::{
        cell::Team,
        entities::{
            champion::{Champion, Direction},
            projectile::GameplayEffect,
        },
        projectile_manager::ProjectileManager,
        spell::{Spell, fireball::FireballSpell, freeze_wall::FreezeWallSpell},
    },
};

fn mock_champion_stats() -> ChampionStats {
    ChampionStats {
        attack_damage: 50,
        attack_speed_ms: 1000,
        health: 500,
        mana: 500,
        armor: 10,
        xp_per_level: vec![100, 200],
        level_up_health_increase: 50,
        level_up_attack_damage_increase: 5,
        level_up_armor_increase: 2,
        attack_range_row: 3,
        attack_range_col: 3,
    }
}

fn mock_fireball_spell_stats() -> SpellStats {
    SpellStats {
        id: 1,
        mana_cost: 50,
        damage_ratio: 1.2,
        base_damage: 60,
        range: 5,
        cooldown_secs: 10,
        speed: 1,
        width: 1,
        stun_duration: None,
        is_heal: Some(false),
    }
}

fn mock_freezewall_spell_stats() -> SpellStats {
    SpellStats {
        id: 0,
        mana_cost: 100,
        damage_ratio: 0.8,
        base_damage: 40,
        range: 3,
        cooldown_secs: 20,
        speed: 1,
        width: 3,
        stun_duration: Some(2),
        is_heal: Some(false),
    }
}

#[test]
fn test_fireball_cast_creates_projectile() {
    let mut champion = Champion::new(1, Team::Blue, 10, 10, mock_champion_stats(), HashMap::new());
    champion.direction = Direction::Right;
    let mut fireball_spell = FireballSpell::new(mock_fireball_spell_stats());
    let mut projectile_manager = ProjectileManager::new();

    fireball_spell.cast(&mut champion, 50, &mut projectile_manager);

    assert_eq!(projectile_manager.projectiles.len(), 1);
    let projectile = projectile_manager.projectiles.values().next().unwrap();
    assert_eq!(projectile.owner_id, 1);
    assert_eq!(projectile.team_id, Team::Blue);

    // Verify the projectile's path
    if let PathingLogic::Straight { path, .. } = &projectile.pathing {
        // Starts one cell to the right of the champion
        assert_eq!(path[0], (10, 11), "Fireball start position is incorrect");
        // Ends `range` cells away
        assert_eq!(
            *path.last().unwrap(),
            (10, 15),
            "Fireball end position is incorrect"
        );
    } else {
        panic!("Fireball should create a Straight path projectile");
    }

    assert_eq!(
        projectile.payloads,
        vec![GameplayEffect::Damage((50.0 * 1.2 + 60.0) as u16)]
    );
}

#[test]
fn test_fireball_cast_respects_cooldown() {
    let mut champion = Champion::new(1, Team::Blue, 10, 10, mock_champion_stats(), HashMap::new());
    let mut fireball_spell = FireballSpell::new(mock_fireball_spell_stats());
    let mut projectile_manager = ProjectileManager::new();

    // First cast
    fireball_spell.cast(&mut champion, 50, &mut projectile_manager);
    assert_eq!(projectile_manager.projectiles.len(), 1);

    // Second cast, should be on cooldown
    fireball_spell.cast(&mut champion, 50, &mut projectile_manager);
    assert_eq!(projectile_manager.projectiles.len(), 1);
}

#[test]
fn test_fireball_cast_checks_mana() {
    let mut champion = Champion::new(1, Team::Blue, 10, 10, mock_champion_stats(), HashMap::new());
    champion.stats.mana = 20; // Not enough mana
    let mut fireball_spell = FireballSpell::new(mock_fireball_spell_stats());
    let mut projectile_manager = ProjectileManager::new();

    fireball_spell.cast(&mut champion, 50, &mut projectile_manager);

    assert_eq!(projectile_manager.projectiles.len(), 0);
}

#[test]
fn test_freezewall_cast_creates_multiple_projectiles() {
    let mut champion = Champion::new(1, Team::Blue, 10, 10, mock_champion_stats(), HashMap::new());
    champion.direction = Direction::Up;
    let mut freezewall_spell = FreezeWallSpell::new(mock_freezewall_spell_stats());
    let mut projectile_manager = ProjectileManager::new();

    freezewall_spell.cast(&mut champion, 50, &mut projectile_manager);

    assert_eq!(projectile_manager.projectiles.len(), 3);
    let mut projectiles: Vec<_> = projectile_manager.projectiles.values().collect();

    // Sort projectiles by their column for deterministic testing
    projectiles.sort_by_key(|p| {
        if let PathingLogic::Straight { path, .. } = &p.pathing {
            path[0].1
        } else {
            0
        }
    });

    // Test payloads
    assert_eq!(projectiles[0].payloads.len(), 2);
    assert_eq!(projectiles[1].payloads.len(), 2);
    assert_eq!(projectiles[2].payloads.len(), 2);

    // Test paths
    // Wall center is (9, 10), width is 3, so projectiles start at cols 9, 10, 11
    if let PathingLogic::Straight { path, .. } = &projectiles[0].pathing {
        assert_eq!(path[0], (9, 9), "Left projectile start position");
        assert_eq!(
            *path.last().unwrap(),
            (6, 9),
            "Left projectile end position"
        );
    } else {
        panic!("FreezeWall should create a Straight path projectile");
    }
    if let PathingLogic::Straight { path, .. } = &projectiles[1].pathing {
        assert_eq!(path[0], (9, 10), "Center projectile start position");
        assert_eq!(
            *path.last().unwrap(),
            (6, 10),
            "Center projectile end position"
        );
    } else {
        panic!("FreezeWall should create a Straight path projectile");
    }
    if let PathingLogic::Straight { path, .. } = &projectiles[2].pathing {
        assert_eq!(path[0], (9, 11), "Right projectile start position");
        assert_eq!(
            *path.last().unwrap(),
            (6, 11),
            "Right projectile end position"
        );
    } else {
        panic!("FreezeWall should create a Straight path projectile");
    }
}

#[test]
fn test_freezewall_cast_respects_cooldown() {
    let mut champion = Champion::new(1, Team::Blue, 10, 10, mock_champion_stats(), HashMap::new());
    let mut freezewall_spell = FreezeWallSpell::new(mock_freezewall_spell_stats());
    let mut projectile_manager = ProjectileManager::new();

    // First cast
    freezewall_spell.cast(&mut champion, 50, &mut projectile_manager);
    assert_eq!(projectile_manager.projectiles.len(), 3);

    // Second cast, should be on cooldown
    freezewall_spell.cast(&mut champion, 50, &mut projectile_manager);
    assert_eq!(projectile_manager.projectiles.len(), 3);
}

#[test]
fn test_freezewall_cast_checks_mana() {
    let mut champion = Champion::new(1, Team::Blue, 10, 10, mock_champion_stats(), HashMap::new());
    champion.stats.mana = 50; // Not enough mana
    let mut freezewall_spell = FreezeWallSpell::new(mock_freezewall_spell_stats());
    let mut projectile_manager = ProjectileManager::new();

    freezewall_spell.cast(&mut champion, 50, &mut projectile_manager);

    assert_eq!(projectile_manager.projectiles.len(), 0);
}
