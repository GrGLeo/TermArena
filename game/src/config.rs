use std::collections::HashMap;
use std::fs;

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
    pub mana: u16,
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
pub struct MonsterStats {
    pub id: String,
    pub spawn_row:  u16,
    pub spawn_col: u16, 
    pub attack_damage: u16,
    pub attack_speed_ms: u64,
    pub health: u16,
    pub armor: u16,
    pub aggro_range_row: u8,
    pub aggro_range_col: u8,
    pub attack_range_row: u8,
    pub attack_range_col: u8,
    pub leash_range: u8,
    pub xp_reward: u8,
    pub respawn_timer_secs: u16,
}

#[derive(Debug, Deserialize, Clone)]
pub struct SpellStats {
    pub id: u8,
    pub mana_cost: u16,
    pub cooldown_secs: u8,
    pub range: u16,
    pub speed: u32,
    pub width: u8,
    pub damage_ratio: f32,
    pub base_damage: u16,
    #[serde(default)]
    pub stun_duration: Option<u8>,
    #[serde(default)]
    pub is_heal: Option<bool>,
}

#[derive(Debug, Deserialize, Clone)]
pub struct SpellFile {
    spell: Vec<SpellStats>,
}

#[derive(Debug, Deserialize, Clone)]
pub struct GameConfig {
    pub base: BaseStats,
    pub champion: ChampionStats,
    pub minion: MinionStats,
    pub tower: TowerStats,
    pub neutral_monsters: Vec<MonsterStats>,
    #[serde(skip)]
    pub spells: HashMap<u8, SpellStats>,
}

impl GameConfig {
    pub fn load(config_path: &str, spell_path: &str) -> Result<Self, Box<dyn std::error::Error>> {
        let content = fs::read_to_string(config_path)?;
        let mut config: GameConfig = toml::from_str(&content)?;

        let spell_content = fs::read_to_string(spell_path)?;
        let spells_file: SpellFile = toml::from_str(&spell_content)?;

        config.spells = spells_file
            .spell
            .into_iter()
            .map(|spell_conf| (spell_conf.id, spell_conf))
            .collect();

        Ok(config)
    }
}
