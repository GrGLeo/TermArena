use std::collections::HashMap;
use std::fs;
use std::time::Duration;

use serde::Deserialize;

#[derive(Debug, Deserialize, Clone)]
pub struct BaseStats {
    pub health: u16,
    pub armor: u16,
}

#[derive(Debug, Deserialize, Clone)]
pub struct ChampionStats {
    pub attack_damage: u16,
    pub attack_speed_ms: u64,
    pub health: u16,
    pub armor: u16,
    pub xp_per_level: Vec<u32>,
    pub level_up_health_increase: u16,
    pub level_up_attack_damage_increase: u16,
    pub level_up_armor_increase: u16,
    pub attack_range_row: u16,
    pub attack_range_col: u16,
}

#[derive(Debug, Deserialize, Clone)]
pub struct MinionStats {
    pub attack_damage: u16,
    pub attack_speed_ms: u64,
    pub health: u16,
    pub armor: u16,
    pub aggro_range_row: u16,
    pub aggro_range_col: u16,
    pub attack_range_row: u16,
    pub attack_range_col: u16,
}

#[derive(Debug, Deserialize, Clone)]
pub struct TowerStats {
    pub attack_damage: u16,
    pub attack_speed_secs: u64,
    pub health: u16,
    pub armor: u16,
    pub attack_range_row: u16,
    pub attack_range_col: u16,
}

#[derive(Debug, Deserialize, Clone)]
pub struct GameConfig {
    pub base: BaseStats,
    pub champion: ChampionStats,
    pub minion: MinionStats,
    pub tower: TowerStats,
}

impl GameConfig {
    pub fn load(path: &str) -> Result<Self, Box<dyn std::error::Error>> {
        let content = fs::read_to_string(path)?;
        let config: GameConfig = toml::from_str(&content)?;
        Ok(config)
    }
}
