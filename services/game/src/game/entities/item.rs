use serde::Deserialize;

#[derive(Debug, Clone, Deserialize, PartialEq)]
pub struct Item {
    pub id: u32,
    pub name: String,
    pub cost: u32,
    pub crafting_cost: Option<u32>,
    pub required: Option<Vec<u32>>,
    pub stats: ItemStats,
}

#[derive(Debug, Clone, Deserialize, PartialEq)]
pub struct ItemStats {
    pub attack_damage: Option<u32>,
    pub attack_speed: Option<u64>,
    pub magic_power: Option<u32>,
    pub health: Option<u32>,
    pub mana: Option<u32>,
    pub armor: Option<u32>,
    pub health_regen: Option<f32>,
}
