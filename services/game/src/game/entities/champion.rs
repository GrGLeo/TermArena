use std::collections::HashMap;
use std::ops::Add;
use std::time::{Duration, Instant};
use std::usize;

use rayon::slice::ParallelSliceMut;

use crate::errors::GameError;
use crate::game::Cell;
use crate::game::animation::melee::MeleeAnimation;
use crate::game::buffs::{Buff, HasBuff};
use crate::game::cell::{CellContent, Team};
use crate::game::projectile_manager::ProjectileManager;
use crate::game::spell::Spell;
use crate::game::{Board, cell::PlayerId};

use super::item::Item;
use super::projectile::GameplayEffect;
use super::{AttackAction, Fighter, Stats, reduced_damage};
use crate::config::ChampionStats;

#[derive(Debug, Clone, Copy)]
pub enum Direction {
    Up,
    Down,
    Left,
    Right,
}

// Casting mechanism
#[derive(Debug, Clone)]
pub enum Ability {
    Recall,
}

pub enum Castable {
    Spell(Box<dyn Spell>),
    Ability(Ability),
}

impl Clone for Castable {
    fn clone(&self) -> Self {
        match self {
            Castable::Spell(spell) => Castable::Spell(spell.clone_box()),
            Castable::Ability(ability) => Castable::Ability(ability.clone()),
        }
    }
}

impl std::fmt::Debug for Castable {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Castable::Spell(spell) => write!(f, "Castable::Spell(id: {})", spell.id()),
            Castable::Ability(ability) => write!(f, "Castable::Ability({:?})", ability),
        }
    }
}

#[derive(Debug, Clone)]
pub struct Cast {
    pub start_time: Instant,
    pub cast_time: Duration,
    pub action: Castable,
}

#[derive(Debug, Clone, Copy)]
pub enum Action {
    MoveUp,
    MoveDown,
    MoveLeft,
    MoveRight,
    Action1,
    Action2,
    AttackMode,
    Recall,
    InvalidAction,
}

#[derive(Debug)]
pub struct Champion {
    pub player_id: PlayerId,
    pub team_id: Team,
    pub xp: u16,
    pub gold: u16,
    pub level: u8,
    pub stats: Stats,
    champion_stats: ChampionStats,
    pub spells: HashMap<u8, Box<dyn Spell>>,
    pub current_cast: Option<Cast>,
    pub active_buffs: HashMap<String, Box<dyn Buff>>,
    on_hit_effects: Vec<GameplayEffect>,
    last_regen: Instant,
    death_counter: u8,
    death_timer: Instant,
    last_attacked: Instant,
    attack_mode: bool,
    stun_timer: Option<Instant>,
    inventory: [Option<Item>; 6],
    pub row: u16,
    pub col: u16,
    pub direction: Direction,
}

impl Champion {
    pub fn new(
        player_id: PlayerId,
        team_id: Team,
        row: u16,
        col: u16,
        champion_stats: ChampionStats,
        spells: HashMap<u8, Box<dyn Spell>>,
    ) -> Self {
        // Calculate % regen to point regen
        let hp_per_sec = champion_stats.health as f32 * champion_stats.health_per_sec;
        let mp_per_sec = champion_stats.mana as f32 * champion_stats.mana_per_sec;

        let stats = Stats {
            attack_damage: champion_stats.attack_damage,
            attack_speed: Duration::from_millis(champion_stats.attack_speed_ms),
            magic_power: champion_stats.magic_power,
            health: champion_stats.health,
            max_health: champion_stats.health,
            hp_per_sec,
            health_regen_acc: 0.0,
            mana: champion_stats.mana,
            max_mana: champion_stats.mana,
            mana_regen_acc: 0.0,
            mp_per_sec,
            armor: champion_stats.armor,
        };

        Champion {
            player_id,
            stats,
            champion_stats,
            spells,
            current_cast: None,
            xp: 0,
            gold: 350,
            level: 1,
            death_counter: 0,
            death_timer: Instant::now(),
            last_attacked: Instant::now(),
            attack_mode: false,
            stun_timer: None,
            inventory: [None, None, None, None, None, None],
            active_buffs: HashMap::new(),
            on_hit_effects: Vec::new(),
            last_regen: Instant::now(),
            team_id,
            row,
            col,
            direction: Direction::Up,
        }
    }

    pub fn stats(&self) -> (u16, u16, u16, u16, u16, u16) {
        (
            self.stats.max_health,
            self.stats.max_mana,
            self.stats.attack_damage,
            self.stats.magic_power,
            self.stats.armor,
            self.gold,
        )
    }

    pub fn add_effects(&mut self, effect: GameplayEffect) {
        self.on_hit_effects.push(effect);
    }

    pub fn add_gold(&mut self, gold: u16) {
        self.gold += gold
    }

    pub fn add_health(&mut self, hp: u8) {
        let health = &mut self.stats.health;
        *health = health.add(hp as u16).min(self.stats.max_health);
    }

    pub fn add_xp(&mut self, xp: u16) {
        self.xp += xp;
        while let Some(xp_needed) = self.xp_for_next_level() {
            if self.xp >= xp_needed {
                self.xp -= xp_needed;
                self.add_level();
            } else {
                break;
            }
        }
    }

    fn add_level(&mut self) {
        self.level += 1;
        self.recalculate_stats();
    }

    pub fn xp_for_next_level(&self) -> Option<u16> {
        if (self.level as usize - 1) < self.champion_stats.xp_per_level.len() {
            Some(self.champion_stats.xp_per_level[self.level as usize - 1])
        } else {
            None
        }
    }

    pub fn add_item(&mut self, item: Item) -> Result<(), GameError> {
        println!("Player inventory: {:?}", self.inventory);

        // Case 1: Item has crafting requirements
        if let Some(required_ids) = &item.required {
            // Ensure crafting_cost is present for craftable items
            let crafting_cost = if let Some(cost) = item.crafting_cost {
                cost as u16
            } else {
                // This should not be reached when an item with requirement is added
                // crafting cost should be included.
                // TODO: Validate config when loaded ?
                return Err(GameError::InvalidInput(
                    "Craftable item missing crafting_cost".to_string(),
                ));
            };

            // Check if player has enough gold for crafting
            if self.gold < crafting_cost {
                return Err(GameError::NotEnoughGold);
            }

            let mut available_inventory_slots: Vec<(usize, u32)> = self
                .inventory
                .iter()
                .enumerate()
                .filter_map(|(idx, slot)| slot.as_ref().map(|item| (idx, item.id)))
                .collect();

            let mut indices_to_remove: Vec<usize> = Vec::new();
            let mut all_requirements_met = true;

            for &req_id_u8 in required_ids {
                let req_id = req_id_u8 as u32;
                if let Some(pos) = available_inventory_slots
                    .iter()
                    .position(|&(_, id)| id == req_id)
                {
                    let (original_idx, _) = available_inventory_slots.remove(pos);
                    indices_to_remove.push(original_idx);
                } else {
                    all_requirements_met = false;
                    break;
                }
            }

            if all_requirements_met {
                // Crafting is possible.
                self.gold -= crafting_cost;

                // Remove required items from the actual inventory.
                for &idx in &indices_to_remove {
                    self.inventory[idx] = None;
                }

                // Add the new item to an empty slot.
                for slot in &mut self.inventory {
                    if slot.is_none() {
                        *slot = Some(item);
                        self.recalculate_stats();
                        return Ok(());
                    }
                }
                // This should ideally not be reached if we removed items and created space.
                return Err(GameError::InventoryFull); // Should be unreachable if logic is sound
            } else {
                // Check if player has enough gold for direct purchase (buyout cost)
                if self.gold < item.cost as u16 {
                    return Err(GameError::NotEnoughGold);
                }

                // Find an empty slot for direct purchase
                for slot in &mut self.inventory {
                    if slot.is_none() {
                        self.gold -= item.cost as u16;
                        *slot = Some(item);
                        self.recalculate_stats();
                        return Ok(());
                    }
                }
                return Err(GameError::InventoryFull);
            }
        } else {
            if self.gold < item.cost as u16 {
                return Err(GameError::NotEnoughGold);
            }
            for slot in &mut self.inventory {
                if slot.is_none() {
                    self.gold -= item.cost as u16;
                    *slot = Some(item);
                    self.recalculate_stats();
                    return Ok(());
                }
            }
            Err(GameError::InventoryFull)
        }
    }

    pub fn get_inventory(&self) -> Vec<u16> {
        let inventory: Vec<u16> = self
            .inventory
            .iter()
            .map(|opt_item| opt_item.as_ref().map_or(0, |item| item.id as u16))
            .collect();
        inventory
    }

    pub fn recalculate_stats(&mut self) {
        let old_max_health = self.stats.max_health;
        let old_max_mana = self.stats.max_mana;

        // Reset to base stats for current level
        let mut max_health = self.champion_stats.health;
        let mut max_mana = self.champion_stats.mana;
        let mut attack_damage = self.champion_stats.attack_damage;
        let mut attack_speed_ms = self.champion_stats.attack_speed_ms;
        let mut magic_power = self.champion_stats.magic_power;
        let mut armor = self.champion_stats.armor;
        let mut health_regen_bonus = 0.0;

        if self.level > 1 {
            let level_ups = (self.level - 1) as u16;
            max_health += self.champion_stats.level_up_health_increase * level_ups;
            attack_damage += self.champion_stats.level_up_attack_damage_increase * level_ups;
            armor += self.champion_stats.level_up_armor_increase * level_ups;
        }

        // Add item stats
        for item in self.inventory.iter().flatten() {
            if let Some(ad) = item.stats.attack_damage {
                attack_damage += ad as u16;
            }
            if let Some(mp) = item.stats.magic_power {
                magic_power += mp as u16;
            }
            if let Some(h) = item.stats.health {
                max_health += h as u16;
            }
            if let Some(m) = item.stats.mana {
                max_mana += m as u16;
            }
            if let Some(a) = item.stats.armor {
                armor += a as u16;
            }
            if let Some(as_) = item.stats.attack_speed {
                attack_speed_ms -= as_;
            }
            if let Some(hr) = item.stats.health_regen {
                health_regen_bonus += hr;
            }
        }

        self.stats.attack_damage = attack_damage;
        self.stats.attack_speed = Duration::from_millis(attack_speed_ms);
        self.stats.magic_power = magic_power;
        self.stats.armor = armor;
        self.stats.max_health = max_health;
        self.stats.max_mana = max_mana;

        let base_hp_per_sec = self.stats.max_health as f32 * self.champion_stats.health_per_sec;
        self.stats.hp_per_sec = base_hp_per_sec * (1.0 + health_regen_bonus);

        let max_health_diff = self.stats.max_health as i32 - old_max_health as i32;
        if max_health_diff > 0 {
            self.stats.health = (self.stats.health as u32 + max_health_diff as u32) as u16;
        }
        self.stats.health = self.stats.health.min(self.stats.max_health);

        let max_mana_diff = self.stats.max_mana as i32 - old_max_mana as i32;
        if max_mana_diff > 0 {
            self.stats.mana = (self.stats.mana as u32 + max_mana_diff as u32) as u16;
        }
        self.stats.mana = self.stats.mana.min(self.stats.max_mana);
    }

    pub fn take_action(
        &mut self,
        action: &Action,
        board: &mut Board,
        projectile_manager: &mut ProjectileManager,
    ) -> Result<(), GameError> {
        // Check if stunned before taking any action
        if self.is_stunned() {
            return Ok(());
        }
        let res = match action {
            Action::MoveUp => {
                self.direction = Direction::Up;
                return self.move_champion(board, -1, 0);
            }
            Action::MoveDown => {
                self.direction = Direction::Down;
                return self.move_champion(board, 1, 0);
            }
            Action::MoveLeft => {
                self.direction = Direction::Left;
                return self.move_champion(board, 0, -1);
            }
            Action::MoveRight => {
                self.direction = Direction::Right;
                return self.move_champion(board, 0, 1);
            }
            Action::Action1 => {
                if self.current_cast.is_some() {
                    return Err(GameError::ChampionBusy);
                }
                if let Some(mut spell) = self.spells.remove(&0) {
                    spell.cast(self, self.stats.attack_damage, self.stats.magic_power, projectile_manager);
                    self.spells.insert(0, spell);
                    return Ok(());
                }
                return Ok(());
            }
            Action::Action2 => {
                if self.current_cast.is_some() {
                    return Err(GameError::ChampionBusy);
                }
                if let Some(mut spell) = self.spells.remove(&1) {
                    spell.cast(self, self.stats.attack_damage, self.stats.magic_power, projectile_manager);
                    self.spells.insert(1, spell);
                    return Ok(());
                }
                return Ok(());
            }
            Action::Recall => {
                if self.current_cast.is_some() {
                    return Err(GameError::ChampionBusy);
                }
                self.current_cast = Some(Cast {
                    start_time: Instant::now(),
                    cast_time: Duration::from_secs(6),
                    action: Castable::Ability(Ability::Recall),
                });
                return Ok(());
            }
            Action::AttackMode => {
                self.attack_mode = !self.attack_mode;
                return Ok(());
            }
            Action::InvalidAction => {
                Err(GameError::InvalidInput("InvalidAction found".to_string()))
            }
        };
        res
    }

    fn move_champion(
        &mut self,
        board: &mut Board,
        d_row: isize,
        d_col: isize,
    ) -> Result<(), GameError> {
        // Champion moving cancel any cast
        if self.current_cast.is_some() {
            self.current_cast = None;
        }
        let new_row = if d_row < 0 {
            self.row.saturating_sub(d_row.unsigned_abs() as u16)
        } else {
            self.row.saturating_add(d_row as u16)
        };

        let new_col = if d_col < 0 {
            self.col.saturating_sub(d_col.unsigned_abs() as u16)
        } else {
            self.col.saturating_add(d_col as u16)
        };

        if new_row >= board.rows as u16 || new_col >= board.cols as u16 {
            return Err(GameError::CannotMoveHere(self.player_id));
        }

        if let Some(new_cell) = board.get_cell(new_row as usize, new_col as usize) {
            if new_cell.is_passable() {
                board.move_cell(
                    self.row as usize,
                    self.col as usize,
                    new_row as usize,
                    new_col as usize,
                );
                self.row = new_row;
                self.col = new_col;
                Ok(())
            } else {
                return Err(GameError::NotFoundCell);
            }
        } else {
            return Err(GameError::NotFoundCell);
        }
    }

    pub fn place_at_base(&mut self, board: &mut Board) {
        let old_row = self.row;
        let old_col = self.col;
        match self.team_id {
            Team::Blue => {
                self.row = 149;
                self.col = 0;
            }
            Team::Red => {
                self.row = 0;
                self.col = 149;
            }
        }
        board.move_cell(
            old_row as usize,
            old_col as usize,
            self.row as usize,
            self.col as usize,
        );
    }

    pub fn is_dead(&self) -> bool {
        if Instant::now() > self.death_timer {
            return false;
        } else {
            true
        }
    }

    pub fn get_health(&self) -> (u16, u16) {
        (self.stats.health, self.stats.max_health)
    }

    pub fn get_cast_info(&self) -> (u16, u16) {
        if let Some(cast) = &self.current_cast {
            let elapsed = cast.start_time.elapsed().as_millis() as u16;
            let duration = cast.cast_time.as_millis() as u16;
            return (elapsed, duration);
        } else {
            return (0, 0);
        }
    }
    
    pub fn reset_aa(&mut self) {
        if let Some(instant) = self.last_attacked.checked_sub(self.stats.attack_speed) {
            self.last_attacked = instant
        }
    }

    pub fn restore_max_health_mana(&mut self) {
        self.stats.health = self.stats.max_health;
        self.stats.mana = self.stats.max_mana;
    }

    pub fn regen_health_mana(&mut self) {
        if self.last_regen.elapsed() >= Duration::from_secs(1) {
            self.last_regen = Instant::now();
            // Health regeneration
            self.stats.health_regen_acc += self.stats.hp_per_sec;
            if self.stats.health_regen_acc >= 1.0 {
                let health_to_add = self.stats.health_regen_acc.trunc();
                self.stats.health =
                    (self.stats.health + health_to_add as u16).min(self.stats.max_health);
                self.stats.health_regen_acc -= health_to_add
            }
            // Mana regeneration
            self.stats.mana_regen_acc += self.stats.mp_per_sec;
            if self.stats.mana_regen_acc >= 1.0 {
                let mana_to_add = self.stats.mana_regen_acc.trunc();
                self.stats.mana = (self.stats.mana + mana_to_add as u16).min(self.stats.max_mana);
                self.stats.mana_regen_acc -= mana_to_add
            }
        }
    }
}

impl Fighter for Champion {
    fn take_effect(&mut self, effects: Vec<GameplayEffect>) {
        for effect in effects.into_iter() {
            match effect {
                GameplayEffect::AttackDamage(damage) => {
                    let reduced_damage = reduced_damage(damage, self.stats.armor);
                    self.stats.health = self.stats.health.saturating_sub(reduced_damage as u16);
                    // Check if champion get killed
                    if self.stats.health == 0 {
                        self.death_counter += 1;
                        let timer = ((self.death_counter as f32).sqrt() * 10.) as u64;
                        self.death_timer = Instant::now() + Duration::from_secs(timer);
                    }
                }
                GameplayEffect::MagicDamage(damage) => {
                    let reduced_damage = reduced_damage(damage, self.stats.armor);
                    self.stats.health = self.stats.health.saturating_sub(reduced_damage as u16);
                    // Check if champion get killed
                    if self.stats.health == 0 {
                        self.death_counter += 1;
                        let timer = ((self.death_counter as f32).sqrt() * 10.) as u64;
                        self.death_timer = Instant::now() + Duration::from_secs(timer);
                    }
                }
                GameplayEffect::Heal(heal_amount) => {
                    self.stats.health =
                        (self.stats.health + heal_amount).min(self.stats.max_health);
                }
                GameplayEffect::Buff(mut buff) => {
                    buff.on_apply(self);
                    self.active_buffs.insert(buff.id().to_string(), buff);
                }
            };
        }
    }

    fn can_attack(&mut self) -> Option<AttackAction> {
        // Cannot attack while stun
        if self.is_stunned() {
            return None;
        }
        if self.last_attacked + self.stats.attack_speed < Instant::now() {
            self.last_attacked = Instant::now();
            let animation = MeleeAnimation::new(self.player_id);
            let mut effects: Vec<GameplayEffect> = self.on_hit_effects.drain(..).collect();
            effects.extend(vec![GameplayEffect::AttackDamage(self.stats.attack_damage)]);
            Some(AttackAction::Melee {
                animation: Box::new(animation),
                effects,
            })
        } else {
            None
        }
    }

    fn get_potential_target<'a>(&self, board: &'a Board) -> Option<&'a Cell> {
        let (row_range, col_range) = (
            self.champion_stats.attack_range_row,
            self.champion_stats.attack_range_col,
        );
        let target_area = board.center_view(self.row, self.col, row_range, col_range);
        let center_row = target_area.len() / 2;
        let center_col = target_area[0].len() / 2;

        target_area
            .iter()
            .enumerate()
            .flat_map(|(row_index, row)| {
                row.iter()
                    .enumerate()
                    .map(move |(col_index, cell)| (row_index, col_index, cell))
            })
            .filter_map(|(row, col, cell)| {
                if let Some(content) = &cell.content {
                    let is_enemy = match content {
                        CellContent::Champion(_, team_id)
                        | CellContent::Tower(_, team_id)
                        | CellContent::Minion(_, team_id)
                        | CellContent::Base(team_id) => *team_id != self.team_id,
                        CellContent::Monster(..) => true,
                    };

                    if is_enemy {
                        if self.attack_mode {
                            // If attack_mode is true, only target champions
                            if let CellContent::Champion(_, _) = content {
                                Some((row, col, cell))
                            } else {
                                None
                            }
                        } else {
                            // If attack_mode is false, target any enemy
                            Some((row, col, cell))
                        }
                    } else {
                        None
                    }
                } else {
                    None
                }
            })
            .min_by(|(r1, c1, _), (r2, c2, _)| {
                let dist1 = r1.abs_diff(center_row) + c1.abs_diff(center_col);
                let dist2 = r2.abs_diff(center_row) + c2.abs_diff(center_col);
                dist1.cmp(&dist2)
            })
            .map(|(_, _, &cell)| cell)
    }
}

impl HasBuff for Champion {
    fn get_stats_mut(&mut self) -> &mut Stats {
        return &mut self.stats;
    }

    fn is_stunned(&self) -> bool {
        self.stun_timer
            .map_or(false, |timer_end| Instant::now() < timer_end)
    }

    fn set_stunned(&mut self, stunned: bool, duration: Option<Duration>) {
        if stunned {
            if let Some(dur) = duration {
                self.stun_timer = Some(Instant::now() + dur);
            } else {
                self.stun_timer = Some(Instant::now() + Duration::from_secs(1));
            }
        } else {
            self.stun_timer = None;
        }
    }
}

#[cfg(test)]
mod tests {

    use super::*;
    use crate::config::{ChampionStats, SpellStats};
    use crate::game::BaseTerrain;
    use crate::game::Board;
    use crate::game::buffs::stun_buff::StunBuff;
    use crate::game::entities::item::{Item, ItemStats};
    use crate::game::spell::freeze_wall::FreezeWallSpell;
    use crate::game::spell::pierce::PierceSpell;

    // Helper function to create a dummy board for tests that require one
    fn create_dummy_board(rows: usize, cols: usize) -> Board {
        Board::new(rows, cols)
    }

    fn create_default_champion_stats() -> ChampionStats {
        ChampionStats {
            attack_damage: 20,
            attack_speed_ms: 2500,
            magic_power: 0,
            health: 200,
            mana: 100,
            armor: 5,
            xp_per_level: vec![
                35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 100, 105, 110, 115,
            ],
            level_up_health_increase: 20,
            level_up_attack_damage_increase: 5,
            level_up_armor_increase: 2,
            health_per_sec: 0.005,
            mana_per_sec: 0.005,
            attack_range_row: 3,
            attack_range_col: 3,
        }
    }

    #[test]
    fn test_new_champion() {
        let player_id = 1;
        let team_id = Team::Red;
        let row = 10;
        let col = 20;
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let champion = Champion::new(player_id, team_id, row, col, champion_stats, spell_stats);

        assert_eq!(champion.player_id, player_id);
        assert_eq!(champion.team_id, team_id);
        assert_eq!(champion.row, row);
        assert_eq!(champion.col, col);
        // Check initial stats (assuming default values from new())
        assert_eq!(champion.stats.attack_damage, 20);
        assert_eq!(champion.stats.health, 200);
        assert_eq!(champion.stats.armor, 5);
        assert_eq!(champion.death_counter, 0);
        // death_timer and last_attacked will be Instant::now(), difficult to assert exact value
        assert!(
            champion.is_dead() == false,
            "Newly created champion should not be dead"
        );
    }

    #[test]
    fn test_take_damage() {
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 2, 2, champion_stats, spell_stats);
        let initial_health = champion.stats.health;
        let damage = 30;
        let armor = champion.stats.armor as u16;

        champion.take_effect(vec![GameplayEffect::AttackDamage(damage)]);

        // Calculate expected health after damage reduction by armor
        let reduced_damage = reduced_damage(damage, armor);
        let expected_health = initial_health.saturating_sub(reduced_damage);
        assert_eq!(
            champion.stats.health, expected_health,
            "Health should be reduced after taking damage"
        );
        assert!(
            !champion.is_dead(),
            "Champion should not be dead after taking some damage"
        );

        // Test taking enough damage to be defeated
        let champion_stats_defeat = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion_to_defeat =
            Champion::new(2, Team::Red, 10, 20, champion_stats_defeat, spell_stats);
        let lethal_damage = 250; // Damage exceeding health + armor

        // Use a specific instant for death timer check
        let start_time = Instant::now();
        // We'll need to mock or control time for precise testing of death timer,
        // but for now, we can at least check if it's set to *sometime in the future*
        // and that is_dead returns true immediately after taking lethal damage.

        champion_to_defeat.take_effect(vec![GameplayEffect::AttackDamage(lethal_damage)]);

        assert_eq!(
            champion_to_defeat.stats.health, 0,
            "Health should be 0 after taking lethal damage"
        );
        assert!(
            champion_to_defeat.is_dead(),
            "Champion should be dead after taking lethal damage"
        );
        // Simple check that death timer was set to a future time
        assert!(
            champion_to_defeat.death_timer > start_time,
            "Death timer should be set to a future time"
        );
        assert_eq!(
            champion_to_defeat.death_counter, 1,
            "Death counter should increment after first defeat"
        );

        // Test taking damage when already at 0 health (should not go below 0)
        let champion_stats_already_defeated = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion_already_defeated = Champion::new(
            3,
            Team::Red,
            10,
            20,
            champion_stats_already_defeated,
            spell_stats,
        );
        champion_already_defeated.stats.health = 0;
        let additional_damage = 10;

        champion_already_defeated.take_effect(vec![GameplayEffect::AttackDamage(additional_damage)]);
        assert_eq!(
            champion_already_defeated.stats.health, 0,
            "Health should remain at 0 if already defeated"
        );
    }

    #[test]
    fn test_take_action_move() {
        let mut board = create_dummy_board(5, 5);
        let mut pm = ProjectileManager::new();
        let spell_stats = HashMap::new();
        let initial_row = 2;
        let initial_col = 2;
        let player_id = 1;

        // Place the champion on the board
        let champion_stats = create_default_champion_stats();
        let mut champion = Champion::new(
            player_id,
            Team::Red,
            initial_row,
            initial_col,
            champion_stats.clone(),
            spell_stats,
        );
        board.place_cell(
            CellContent::Champion(player_id, Team::Red),
            initial_row as usize,
            initial_col as usize,
        );

        // Test moving up
        let action_up = Action::MoveUp;
        let result_up = champion.take_action(&action_up, &mut board, &mut pm);
        assert!(result_up.is_ok(), "Moving up should be successful");
        assert_eq!(
            champion.row,
            initial_row - 1,
            "Champion row should decrease after moving up"
        );
        assert_eq!(
            champion.col, initial_col,
            "Champion col should remain the same after moving up"
        );
        // Verify board state: old cell is empty, new cell has champion content
        let old_cell_up = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Old cell should exist");
        assert!(
            old_cell_up.content.is_none(),
            "Old cell should be empty after moving up"
        );
        let new_cell_up = board
            .get_cell((initial_row - 1) as usize, initial_col as usize)
            .expect("New cell should exist");
        assert_eq!(
            new_cell_up.content,
            Some(CellContent::Champion(player_id, Team::Red)),
            "New cell should have champion content after moving up"
        );

        // Reset champion position and board for next move test
        let current_row = champion.row; // Current row is now initial_row - 1
        let current_col = champion.col; // Current col is now initial_col
        board.clear_cell(current_row as usize, current_col as usize);
        champion.row = initial_row;
        champion.col = initial_col;
        board.place_cell(
            CellContent::Champion(player_id, Team::Red),
            initial_row as usize,
            initial_col as usize,
        );

        // Test moving right
        let action_right = Action::MoveRight;
        let result_right = champion.take_action(&action_right, &mut board, &mut pm);
        assert!(result_right.is_ok(), "Moving right should be successful");
        assert_eq!(
            champion.row, initial_row,
            "Champion row should remain the same after moving right"
        );
        assert_eq!(
            champion.col,
            initial_col + 1,
            "Champion col should increase after moving right"
        );
        // Verify board state
        let old_cell_right = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Old cell should exist");
        assert!(
            old_cell_right.content.is_none(),
            "Old cell should be empty after moving right"
        );
        let new_cell_right = board
            .get_cell(initial_row as usize, (initial_col + 1) as usize)
            .expect("New cell should exist");
        assert_eq!(
            new_cell_right.content,
            Some(CellContent::Champion(player_id, Team::Red)),
            "New cell should have champion content after moving right"
        );

        // Add tests for MoveDown and MoveLeft similarly...
        // Reset
        let current_row = champion.row; // Current row is now initial_row
        let current_col = champion.col; // Current col is now initial_col + 1
        board.clear_cell(current_row as usize, current_col as usize);
        champion.row = initial_row;
        champion.col = initial_col;
        board.place_cell(
            CellContent::Champion(player_id, Team::Red),
            initial_row as usize,
            initial_col as usize,
        );

        // Test moving down
        let action_down = Action::MoveDown;
        let result_down = champion.take_action(&action_down, &mut board, &mut pm);
        assert!(result_down.is_ok(), "Moving down should be successful");
        assert_eq!(
            champion.row,
            initial_row + 1,
            "Champion row should increase after moving down"
        );
        assert_eq!(
            champion.col, initial_col,
            "Champion col should remain the same after moving down"
        );
        // Verify board state
        let old_cell_down = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Old cell should exist");
        assert!(
            old_cell_down.content.is_none(),
            "Old cell should be empty after moving down"
        );
        let new_cell_down = board
            .get_cell((initial_row + 1) as usize, initial_col as usize)
            .expect("New cell should exist");
        assert_eq!(
            new_cell_down.content,
            Some(CellContent::Champion(player_id, Team::Red)),
            "New cell should have champion content after moving down"
        );

        // Reset
        let current_row = champion.row; // Current row is now initial_row + 1
        let current_col = champion.col; // Current col is now initial_col
        board.clear_cell(current_row as usize, current_col as usize);
        champion.row = initial_row;
        champion.col = initial_col;
        board.place_cell(
            CellContent::Champion(player_id, Team::Red),
            initial_row as usize,
            initial_col as usize,
        );

        // Test moving left
        let action_left = Action::MoveLeft;
        let result_left = champion.take_action(&action_left, &mut board, &mut pm);
        assert!(result_left.is_ok(), "Moving left should be successful");
        assert_eq!(
            champion.row, initial_row,
            "Champion row should remain the same after moving left"
        );
        assert_eq!(
            champion.col,
            initial_col - 1,
            "Champion col should decrease after moving left"
        );
        // Verify board state
        let old_cell_left = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Old cell should exist");
        assert!(
            old_cell_left.content.is_none(),
            "Old cell should be empty after moving left"
        );
        let new_cell_left = board
            .get_cell(initial_row as usize, (initial_col - 1) as usize)
            .expect("New cell should exist");
        assert_eq!(
            new_cell_left.content,
            Some(CellContent::Champion(player_id, Team::Red)),
            "New cell should have champion content after moving left"
        );
    }

    #[test]
    fn test_take_action_move_into_impassable() {
        let mut board = create_dummy_board(5, 5);
        let mut pm = ProjectileManager::new();
        let spell_stats = HashMap::new();
        let initial_row = 2;
        let initial_col = 2;
        let player_id = 1;

        // Place the champion on the board
        let champion_stats = create_default_champion_stats();
        let mut champion = Champion::new(
            player_id,
            Team::Red,
            initial_row,
            initial_col,
            champion_stats.clone(),
            spell_stats,
        );
        board.place_cell(
            CellContent::Champion(player_id, Team::Red),
            initial_row as usize,
            initial_col as usize,
        );

        // Place a wall next to the champion
        let wall_row = initial_row - 1;
        let wall_col = initial_col;
        board.change_base(BaseTerrain::Wall, wall_row as usize, wall_col as usize);

        // Attempt to move into the wall
        let action_up = Action::MoveUp;
        let result_up = champion.take_action(&action_up, &mut board, &mut pm);

        assert!(
            result_up.is_err(),
            "Moving into a wall should return an error"
        );
        assert_eq!(
            champion.row, initial_row,
            "Champion row should not change after failing to move"
        );
        assert_eq!(
            champion.col, initial_col,
            "Champion col should not change after failing to move"
        );
        // Verify board state: champion should still be in the original cell
        let initial_cell = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Initial cell should exist");
        assert_eq!(
            initial_cell.content,
            Some(CellContent::Champion(player_id, Team::Red)),
            "Champion should remain in the initial cell"
        );

        // Place content in a cell next to the champion
        let content_row = initial_row;
        let content_col = initial_col + 1;
        board.place_cell(
            CellContent::Minion(1, Team::Blue),
            content_row as usize,
            content_col as usize,
        );

        // Attempt to move into the cell with content
        let action_right = Action::MoveRight;
        let result_right = champion.take_action(&action_right, &mut board, &mut pm);

        assert!(
            result_right.is_err(),
            "Moving into a cell with content should return an error"
        );
        assert_eq!(
            champion.row, initial_row,
            "Champion row should not change after failing to move"
        );
        assert_eq!(
            champion.col, initial_col,
            "Champion col should not change after failing to move"
        );
        // Verify board state: champion should still be in the original cell, content still in target cell
        let initial_cell_after_fail = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Initial cell should exist");
        assert_eq!(
            initial_cell_after_fail.content,
            Some(CellContent::Champion(player_id, Team::Red)),
            "Champion should remain in the initial cell"
        );
        let target_cell_after_fail = board
            .get_cell(content_row as usize, content_col as usize)
            .expect("Target cell should exist");
        assert_eq!(
            target_cell_after_fail.content,
            Some(CellContent::Minion(1, Team::Blue)),
            "Content should remain in the target cell"
        );
    }

    #[test]
    fn test_take_action_one() {
        let mut board = create_dummy_board(5, 5);
        let mut pm = ProjectileManager::new();
        let champion_stats = create_default_champion_stats();
        let spell_stat = SpellStats {
            id: 0,
            mana_cost: 10,
            cooldown_secs: 5,
            range: 10,
            width: 5,
            speed: 1,
            base_attack_damage: 20,
            base_magic_damage: 0,
            damage_ratio: 0.8,
            magic_ratio: 0.,
            effect_duration: Some(5),
            is_heal: Some(false),
        };
        let mut spell_stats: HashMap<u8, Box<dyn Spell>> = HashMap::new();
        let spell = Box::new(FreezeWallSpell::new(spell_stat));
        spell_stats.insert(0, spell);

        let mut champion = Champion::new(1, Team::Red, 2, 2, champion_stats, spell_stats);

        // Test Action1 (currently does nothing, should not error)
        let action1 = Action::Action1;
        let result1 = champion.take_action(&action1, &mut board, &mut pm);
        assert!(result1.is_ok(), "Action1 should not return an error");

        // Test Action1 correctly created 5 projectiles
        assert_eq!(pm.projectiles.len(), 5);
    }

    #[test]
    fn test_take_action_other_actions() {
        let mut board = create_dummy_board(5, 5);
        let mut pm = ProjectileManager::new();
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 2, 2, champion_stats, spell_stats);

        // Test Action1 (currently does nothing, should not error)
        let action1 = Action::Action1;
        let result1 = champion.take_action(&action1, &mut board, &mut pm);
        assert!(result1.is_ok(), "Action1 should not return an error");

        // Test Action2 (currently does nothing, should not error)
        let action2 = Action::Action2;
        let result2 = champion.take_action(&action2, &mut board, &mut pm);
        assert!(result2.is_ok(), "Action2 should not return an error");
    }

    #[test]
    fn test_take_action_invalid_action() {
        let mut board = create_dummy_board(5, 5);
        let mut pm = ProjectileManager::new();
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 2, 2, champion_stats.clone(), spell_stats);

        // Test InvalidAction
        let invalid_action = Action::InvalidAction;
        let result = champion.take_action(&invalid_action, &mut board, &mut pm);
        println!("{:?}", result);

        assert!(result.is_err(), "InvalidAction should return an error");
        // Optionally, check the specific error type if needed, but checking for an error is sufficient for now.
    }

    #[test]
    fn test_place_at_base() {
        let mut board = create_dummy_board(200, 200); // Use a board large enough for base position
        let initial_row = 10;
        let initial_col = 10;
        let player_id = 1;
        let base_row = 0;
        let base_col = 149;

        // Place champion at initial position
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(
            player_id,
            Team::Red,
            initial_row,
            initial_col,
            champion_stats.clone(),
            spell_stats,
        );
        board.place_cell(
            CellContent::Champion(player_id, Team::Red),
            initial_row as usize,
            initial_col as usize,
        );

        // Place the champion at base
        champion.place_at_base(&mut board);

        // Check if champion's position updated
        assert_eq!(
            champion.row, base_row,
            "Champion's row should update to base row"
        );
        assert_eq!(
            champion.col, base_col,
            "Champion's col should update to base col"
        );

        // Verify board state: old position is empty, base position has champion content
        let old_cell = board
            .get_cell(initial_row as usize, initial_col as usize)
            .expect("Old cell should exist");
        assert!(
            old_cell.content.is_none(),
            "Old position should be empty after placing at base"
        );
        let base_cell = board
            .get_cell(base_row as usize, base_col as usize)
            .expect("Base cell should exist");
        assert_eq!(
            base_cell.content,
            Some(CellContent::Champion(player_id, Team::Red)),
            "Base position should have champion content"
        );
    }

    #[test]
    fn test_scan_range_no_enemy_in_range() {
        let mut board = create_dummy_board(10, 10);
        let champion_row = 5;
        let champion_col = 5;
        let player_id = 1;
        let champion_team = Team::Red;

        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let champion = Champion::new(
            player_id,
            champion_team,
            champion_row,
            champion_col,
            champion_stats,
            spell_stats,
        );
        board.place_cell(
            CellContent::Champion(player_id, champion_team),
            champion_row as usize,
            champion_col as usize,
        );

        // Case 1: No other entities on the board
        let target_none = champion.get_potential_target(&board);
        assert!(
            target_none.is_none(),
            "scan_range should return None when no other entities are present"
        );

        // Case 2: Ally champion in range
        let ally_id = 2;
        let ally_row = champion_row - 1; // Within 3x3 range
        let ally_col = champion_col;
        board.place_cell(
            CellContent::Champion(ally_id, champion_team),
            ally_row as usize,
            ally_col as usize,
        );
        let target_ally = champion.get_potential_target(&board);
        assert!(
            target_ally.is_none(),
            "scan_range should return None when only allies are in range"
        );
    }

    #[test]
    fn test_scan_range_enemy_in_range() {
        let mut board = create_dummy_board(10, 10);
        let champion_row = 5;
        let champion_col = 5;
        let player_id = 1;
        let champion_team = Team::Red;

        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let champion = Champion::new(
            player_id,
            champion_team,
            champion_row,
            champion_col,
            champion_stats,
            spell_stats,
        );
        board.place_cell(
            CellContent::Champion(player_id, champion_team),
            champion_row as usize,
            champion_col as usize,
        );

        // Place an enemy champion in range
        let enemy_id = 2;
        let enemy_team = Team::Blue; // Different team
        let enemy_row = champion_row + 1; // Within 3x3 range
        let enemy_col = champion_col + 1; // Within 3x3 range
        let enemy_cell_content = CellContent::Champion(enemy_id, enemy_team);
        board.place_cell(
            enemy_cell_content.clone(),
            enemy_row as usize,
            enemy_col as usize,
        );

        let target = champion.get_potential_target(&board);

        assert!(
            target.is_some(),
            "scan_range should return Some when an enemy is in range"
        );
        let target_cell = target.unwrap();
        assert_eq!(
            target_cell.content,
            Some(enemy_cell_content),
            "The returned cell should contain the enemy champion"
        );

        // Check another enemy type (Tower)
        let tower_id = 1;
        let tower_team = Team::Blue;
        let tower_row = champion_row - 1;
        let tower_col = champion_col;
        let tower_cell_content = CellContent::Tower(tower_id, tower_team);
        board.clear_cell(enemy_row as usize, enemy_col as usize); // Remove previous enemy
        board.place_cell(
            tower_cell_content.clone(),
            tower_row as usize,
            tower_col as usize,
        );

        let target_tower = champion.get_potential_target(&board);
        assert!(
            target_tower.is_some(),
            "scan_range should return Some when an enemy tower is in range"
        );
        let target_tower_cell = target_tower.unwrap();
        assert_eq!(
            target_tower_cell.content,
            Some(tower_cell_content),
            "The returned cell should contain the enemy tower"
        );
    }

    #[test]
    fn test_scan_range_multiple_enemies_in_range() {
        let mut board = create_dummy_board(10, 10);
        let champion_row = 5;
        let champion_col = 5;
        let player_id = 1;
        let champion_team = Team::Red;

        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let champion = Champion::new(
            player_id,
            champion_team,
            champion_row,
            champion_col,
            champion_stats,
            spell_stats,
        );
        board.place_cell(
            CellContent::Champion(player_id, champion_team),
            champion_row as usize,
            champion_col as usize,
        );

        // Place multiple enemies at different distances within range
        let enemy_team = Team::Blue;

        // Closest enemy (Manhattan distance 1)
        let closest_enemy_row = champion_row;
        let closest_enemy_col = champion_col + 1;
        let closest_enemy_content = CellContent::Champion(2, enemy_team);
        board.place_cell(
            closest_enemy_content.clone(),
            closest_enemy_row as usize,
            closest_enemy_col as usize,
        );

        // Further enemy (Manhattan distance 2)
        let further_enemy_row = champion_row + 1;
        let further_enemy_col = champion_col + 1;
        let further_enemy_content = CellContent::Minion(1, enemy_team);
        board.place_cell(
            further_enemy_content.clone(),
            further_enemy_row as usize,
            further_enemy_col as usize,
        );

        // Even further enemy (Manhattan distance 2)
        let even_further_enemy_row = champion_row - 1;
        let even_further_enemy_col = champion_col - 1;
        let even_further_enemy_content = CellContent::Tower(1, enemy_team);
        board.place_cell(
            even_further_enemy_content.clone(),
            even_further_enemy_row as usize,
            even_further_enemy_col as usize,
        );

        let target = champion.get_potential_target(&board);

        assert!(
            target.is_some(),
            "scan_range should return Some when multiple enemies are in range"
        );
        let target_cell = target.unwrap();
        // Verify that the returned cell contains the closest enemy
        assert_eq!(
            target_cell.content,
            Some(closest_enemy_content),
            "scan_range should return the closest enemy"
        );
    }

    #[test]
    fn test_scan_range_enemies_outside_range() {
        let mut board = create_dummy_board(10, 10);
        let champion_row = 5;
        let champion_col = 5;
        let player_id = 1;
        let champion_team = Team::Red;

        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let champion = Champion::new(
            player_id,
            champion_team,
            champion_row,
            champion_col,
            champion_stats,
            spell_stats,
        );
        board.place_cell(
            CellContent::Champion(player_id, champion_team),
            champion_row as usize,
            champion_col as usize,
        );

        // Place an enemy champion outside the 3x3 range
        let enemy_id = 2;
        let enemy_team = Team::Blue; // Different team
        let enemy_row_outside = champion_row + 2; // Outside 3x3 range (center is 1 tile away, edge is 1 tile away, 2 is outside)
        let enemy_col_outside = champion_col + 2; // Outside 3x3 range
        board.place_cell(
            CellContent::Champion(enemy_id, enemy_team),
            enemy_row_outside as usize,
            enemy_col_outside as usize,
        );

        let target = champion.get_potential_target(&board);

        assert!(
            target.is_none(),
            "scan_range should return None when enemies are outside the 3x3 range"
        );
    }

    #[test]
    fn test_champion_stun_application() {
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 2, 2, champion_stats, spell_stats);
        let mut board = create_dummy_board(10, 10);
        let mut pm = ProjectileManager::new();
        board.place_cell(CellContent::Champion(1, Team::Red), 2, 2);

        // Apply a stun buff
        let stun_duration_secs = 5;
        let stun_effect = GameplayEffect::Buff(Box::new(StunBuff::new(stun_duration_secs)));
        champion.take_effect(vec![stun_effect]);

        // Assert champion is stunned
        assert!(
            champion.is_stunned(),
            "Champion should be stunned after applying stun buff"
        );

        let initial_row = champion.row;
        let initial_col = champion.col;

        // Assert stunned champion cannot move
        let move_action = Action::MoveUp;
        let move_result = champion.take_action(&move_action, &mut board, &mut pm);
        assert!(
            move_result.is_ok(),
            "take_action for a stunned champion should return Ok"
        );
        assert_eq!(
            champion.row, initial_row,
            "Stunned champion's row should not change"
        );
        assert_eq!(
            champion.col, initial_col,
            "Stunned champion's col should not change"
        );

        // Assert stunned champion cannot attack
        assert!(
            champion.can_attack().is_none(),
            "Stunned champion should not be able to attack"
        );
    }

    #[test]
    fn test_champion_stun_expiration() {
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 0, 0, champion_stats, spell_stats);

        // Apply a very short stun buff
        let stun_effect = GameplayEffect::Buff(Box::new(StunBuff::new(0))); // Duration 0 for immediate expiration
        champion.take_effect(vec![stun_effect]);

        // Manually process buffs to trigger expiration
        let current_buffs = std::mem::take(&mut champion.active_buffs);
        let mut kept_buffs = HashMap::new();
        for (id, mut buff) in current_buffs.into_iter() {
            if buff.on_tick(&mut champion) {
                buff.on_remove(&mut champion);
            } else {
                kept_buffs.insert(id, buff);
            }
        }
        champion.active_buffs = kept_buffs;

        // Assert champion is no longer stunned
        assert!(
            !champion.is_stunned(),
            "Champion should not be stunned after buff expiration"
        );

        // Assert champion can now move (assuming board and pm are set up for a valid move)
        let mut board = create_dummy_board(10, 10);
        let mut pm = ProjectileManager::new();
        // Place champion on board for movement test
        board.place_cell(
            CellContent::Champion(champion.player_id, champion.team_id),
            champion.row as usize,
            champion.col as usize,
        );
        let move_action = Action::MoveDown;
        let move_result = champion.take_action(&move_action, &mut board, &mut pm);
        assert!(
            move_result.is_ok(),
            "Unstunned champion should be able to move"
        );

        // Assert champion can now attack
        // For can_attack to return Some, last_attacked needs to be old enough.
        // In a real test, you might mock Instant::now() or set last_attacked explicitly.
        // For simplicity here, we'll just check if it's not None.
        // Note: This test might be flaky if run too quickly after champion creation due to Instant::now()
        // A more robust test would involve setting champion.last_attacked to a past time.
        champion.last_attacked =
            Instant::now() - champion.stats.attack_speed - Duration::from_secs(1);
        assert!(
            champion.can_attack().is_some(),
            "Unstunned champion should be able to attack"
        );
    }

    #[test]
    fn test_level_up() {
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 0, 0, champion_stats, spell_stats);
        assert_eq!(champion.level, 1);
        assert_eq!(champion.stats.max_health, 200);
        assert_eq!(champion.stats.attack_damage, 20);
        assert_eq!(champion.stats.armor, 5);

        champion.add_xp(35);
        assert_eq!(champion.level, 2);
        assert_eq!(champion.xp, 0);
        assert_eq!(champion.stats.max_health, 220);
        assert_eq!(champion.stats.attack_damage, 25);
        assert_eq!(champion.stats.armor, 7);

        champion.add_xp(40);
        assert_eq!(champion.level, 3);
        assert_eq!(champion.xp, 0);
        assert_eq!(champion.stats.max_health, 240);
        assert_eq!(champion.stats.attack_damage, 30);
        assert_eq!(champion.stats.armor, 9);
    }

    #[test]
    fn test_get_potential_target_attack_mode() {
        let mut board = create_dummy_board(10, 10);
        let champion_row = 5;
        let champion_col = 5;
        let player_id = 1;
        let champion_team = Team::Red;

        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(
            player_id,
            champion_team,
            champion_row,
            champion_col,
            champion_stats,
            spell_stats,
        );
        board.place_cell(
            CellContent::Champion(player_id, champion_team),
            champion_row as usize,
            champion_col as usize,
        );

        let enemy_team = Team::Blue;

        // Place a minion (closest enemy)
        let minion_row = champion_row;
        let minion_col = champion_col + 1;
        let minion_content = CellContent::Minion(1, enemy_team);
        board.place_cell(
            minion_content.clone(),
            minion_row as usize,
            minion_col as usize,
        );

        // Place an enemy champion (further away than minion)
        let enemy_champion_row = champion_row + 1;
        let enemy_champion_col = champion_col + 1;
        let enemy_champion_content = CellContent::Champion(2, enemy_team);
        board.place_cell(
            enemy_champion_content.clone(),
            enemy_champion_row as usize,
            enemy_champion_col as usize,
        );

        // Test with attack_mode = true (should target only champion)
        champion.attack_mode = true;
        let target_with_attack_mode = champion.get_potential_target(&board);
        assert!(
            target_with_attack_mode.is_some(),
            "Should find a target when attack_mode is true"
        );
        assert_eq!(
            target_with_attack_mode.unwrap().content,
            Some(enemy_champion_content.clone()),
            "Should target enemy champion when attack_mode is true"
        );

        // Test with attack_mode = false (should target closest enemy, which is minion)
        champion.attack_mode = false;
        let target_without_attack_mode = champion.get_potential_target(&board);
        assert!(
            target_without_attack_mode.is_some(),
            "Should find a target when attack_mode is false"
        );
        assert_eq!(
            target_without_attack_mode.unwrap().content,
            Some(minion_content.clone()),
            "Should target closest enemy (minion) when attack_mode is false"
        );
    }

    #[test]
    fn test_champion_can_attack_monster() {
        let mut board = create_dummy_board(10, 10);
        let champion_row = 5;
        let champion_col = 5;
        let player_id = 1;
        let champion_team = Team::Red;

        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(
            player_id,
            champion_team,
            champion_row,
            champion_col,
            champion_stats,
            spell_stats,
        );
        board.place_cell(
            CellContent::Champion(player_id, champion_team),
            champion_row as usize,
            champion_col as usize,
        );

        // Place a monster in range
        let monster_id = 1;
        let monster_row = champion_row + 1;
        let monster_col = champion_col;
        board.place_cell(
            CellContent::Monster(monster_id),
            monster_row as usize,
            monster_col as usize,
        );

        // Verify that the monster is a potential target
        let target_cell_option = champion.get_potential_target(&board);
        assert!(
            target_cell_option.is_some(),
            "Champion should be able to target a monster"
        );
        let target_cell = target_cell_option.unwrap();
        assert_eq!(
            target_cell.content,
            Some(CellContent::Monster(monster_id)),
            "Target should be the monster"
        );

        // Verify that the champion can attack
        champion.last_attacked =
            Instant::now() - champion.stats.attack_speed - Duration::from_secs(1); // Ensure cooldown is ready
        let attack_action = champion.can_attack();
        assert!(
            attack_action.is_some(),
            "Champion should be able to attack after targeting a monster"
        );
    }

    #[test]
    fn test_add_item_to_inventory() {
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 0, 0, champion_stats, spell_stats);
        champion.gold = 600;

        let item1 = Item {
            id: 1,
            name: "Sword".to_string(),
            cost: 300,
            crafting_cost: None,
            required: None,
            stats: ItemStats {
                attack_damage: Some(10),
                attack_speed: None,
                magic_power: None,
                health: None,
                armor: None,
                mana: None,
                health_regen: None,
            },
        };

        let item2 = Item {
            id: 2,
            name: "Shield".to_string(),
            cost: 200,
            crafting_cost: None,
            required: None,
            stats: ItemStats {
                attack_damage: None,
                attack_speed: None,
                magic_power: None,
                health: Some(50),
                armor: Some(5),
                mana: None,
                health_regen: None,
            },
        };

        // Add first item
        let result = champion.add_item(item1.clone());
        assert!(result.is_ok());
        assert_eq!(champion.inventory[0], Some(item1.clone()));
        assert_eq!(champion.inventory[1], None);

        // Add second item
        let result = champion.add_item(item2.clone());
        assert!(result.is_ok());
        assert_eq!(champion.inventory[0], Some(item1.clone()));
        assert_eq!(champion.inventory[1], Some(item2.clone()));

        // Fill the inventory
        for i in 2..6 {
            let item = Item {
                id: i as u32,
                name: format!("Item {}", i),
                cost: 10,
                crafting_cost: None,
                required: None,
                stats: ItemStats {
                    attack_damage: None,
                    attack_speed: None,
                    magic_power: None,
                    health: None,
                    armor: None,
                    mana: None,
                    health_regen: None,
                },
            };
            champion.add_item(item.clone()).unwrap();
        }

        // Try to add another item to a full inventory
        let extra_item = Item {
            id: 7,
            name: "Extra Item".to_string(),
            cost: 10,
            crafting_cost: None,
            required: None,
            stats: ItemStats {
                attack_damage: None,
                attack_speed: None,
                magic_power: None,
                health: None,
                armor: None,
                mana: None,
                health_regen: None,
            },
        };
        let result = champion.add_item(extra_item);
        assert!(result.is_err());
        assert_eq!(result.unwrap_err(), GameError::InventoryFull);
    }

    #[test]
    fn test_add_item_not_enough_gold() {
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 0, 0, champion_stats, spell_stats);
        champion.gold = 200;

        let item = Item {
            id: 1,
            name: "Expensive Sword".to_string(),
            cost: 300,
            crafting_cost: None,
            required: None,
            stats: ItemStats {
                attack_damage: Some(10),
                attack_speed: None,
                magic_power: None,
                health: None,
                armor: None,
                mana: None,
                health_regen: None,
            },
        };

        let result = champion.add_item(item);
        assert!(result.is_err());
        assert_eq!(result.unwrap_err(), GameError::NotEnoughGold);
    }

    #[test]
    fn test_recalculate_stats_with_items_and_level_up() {
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 0, 0, champion_stats, spell_stats);
        champion.gold = 1000;

        let item1 = Item {
            id: 1,
            name: "Sword".to_string(),
            cost: 300,
            crafting_cost: None,
            required: None,
            stats: ItemStats {
                attack_damage: Some(10),
                attack_speed: None,
                magic_power: None,
                health: None,
                armor: None,
                mana: None,
                health_regen: None,
            },
        };

        let item2 = Item {
            id: 2,
            name: "Shield".to_string(),
            cost: 200,
            crafting_cost: None,
            required: None,
            stats: ItemStats {
                attack_damage: None,
                attack_speed: None,
                magic_power: None,
                health: Some(50),
                armor: Some(5),
                mana: None,
                health_regen: None,
            },
        };

        // Add items and check stats
        champion.add_item(item1).unwrap();
        champion.add_item(item2).unwrap();

        assert_eq!(champion.stats.attack_damage, 20 + 10);
        assert_eq!(champion.stats.max_health, 200 + 50);
        assert_eq!(champion.stats.armor, 5 + 5);

        // Level up and check stats
        champion.add_xp(35); // Level up to 2
        assert_eq!(champion.level, 2);

        let expected_health = 200 + 20 + 50; // Base + Level up + Item
        let expected_attack_damage = 20 + 5 + 10; // Base + Level up + Item
        let expected_armor = 5 + 2 + 5; // Base + Level up + Item

        assert_eq!(champion.stats.max_health, expected_health);
        assert_eq!(champion.stats.attack_damage, expected_attack_damage);
        assert_eq!(champion.stats.armor, expected_armor);
    }

    #[test]
    fn test_recalculate_stats_with_health_regen_item() {
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 0, 0, champion_stats, spell_stats);
        champion.gold = 1000;

        let item = Item {
            id: 6,
            name: "Vial of Renewal".to_string(),
            cost: 150,
            crafting_cost: None,
            required: None,
            stats: ItemStats {
                attack_damage: None,
                attack_speed: None,
                magic_power: None,
                health: None,
                armor: None,
                mana: None,
                health_regen: Some(1.0),
            },
        };

        champion.add_item(item).unwrap();

        let base_hp_per_sec =
            champion.stats.max_health as f32 * champion.champion_stats.health_per_sec;
        let expected_hp_per_sec = base_hp_per_sec * 2.0; // 1.0 bonus = 100% increase

        assert!((champion.stats.hp_per_sec - expected_hp_per_sec).abs() < f32::EPSILON);
    }

    #[test]
    fn test_recalculate_stats_with_multiple_health_regen_items() {
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 0, 0, champion_stats, spell_stats);
        champion.gold = 1000;

        let item1 = Item {
            id: 6,
            name: "Vial of Renewal".to_string(),
            cost: 150,
            crafting_cost: None,
            required: None,
            stats: ItemStats {
                attack_damage: None,
                attack_speed: None,
                magic_power: None,
                health: None,
                armor: None,
                mana: None,
                health_regen: Some(1.0),
            },
        };
        let item2 = Item {
            id: 8,
            name: "Another Vial".to_string(),
            cost: 150,
            crafting_cost: None,
            required: None,
            stats: ItemStats {
                attack_damage: None,
                attack_speed: None,
                magic_power: None,
                health: None,
                armor: None,
                mana: None,
                health_regen: Some(0.5),
            },
        };

        champion.add_item(item1).unwrap();
        champion.add_item(item2).unwrap();

        let base_hp_per_sec =
            champion.stats.max_health as f32 * champion.champion_stats.health_per_sec;
        let expected_hp_per_sec = base_hp_per_sec * (1.0 + 1.0 + 0.5); // 1.0 base + 1.0 from item1 + 0.5 from item2

        assert!((champion.stats.hp_per_sec - expected_hp_per_sec).abs() < f32::EPSILON);
    }

    #[test]
    fn test_recalculate_stats_with_health_and_regen_item() {
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 0, 0, champion_stats, spell_stats);
        champion.gold = 1000;

        let item = Item {
            id: 9,
            name: "Magic Shield".to_string(),
            cost: 350,
            crafting_cost: None,
            required: None,
            stats: ItemStats {
                attack_damage: None,
                attack_speed: None,
                magic_power: None,
                health: Some(50),
                armor: Some(5),
                mana: None,
                health_regen: Some(1.0),
            },
        };

        champion.add_item(item).unwrap();

        let expected_max_health = 200 + 50;
        assert_eq!(champion.stats.max_health, expected_max_health);

        let base_hp_per_sec = expected_max_health as f32 * champion.champion_stats.health_per_sec;
        let expected_hp_per_sec = base_hp_per_sec * (1.0 + 1.0);

        assert!((champion.stats.hp_per_sec - expected_hp_per_sec).abs() < f32::EPSILON);
    }

    #[test]
    fn test_regen_health() {
        use std::thread;
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 0, 0, champion_stats, spell_stats);

        // Base regen is 200 * 0.005 = 1 hp/sec
        assert_eq!(champion.stats.hp_per_sec, 1.0);

        champion.stats.health = 100;

        thread::sleep(Duration::from_secs(1));
        champion.regen_health_mana();
        assert_eq!(champion.stats.health, 101);

        // Test not exceeding max health
        champion.stats.health = 199;
        thread::sleep(Duration::from_secs(1));
        champion.regen_health_mana();
        assert_eq!(champion.stats.health, 200);

        champion.regen_health_mana(); // Should do nothing as not enough time has passed
        assert_eq!(champion.stats.health, 200);

        thread::sleep(Duration::from_secs(1));
        champion.regen_health_mana();
        assert_eq!(champion.stats.health, 200); // Still at max
    }

    #[test]
    fn test_fractional_health_regen() {
        use std::thread;
        let mut champion_stats = create_default_champion_stats();
        champion_stats.health_per_sec = 0.0025; // 200 * 0.0025 = 0.5 hp/sec
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 0, 0, champion_stats, spell_stats);

        assert!((champion.stats.hp_per_sec - 0.5).abs() < f32::EPSILON);
        champion.stats.health = 100;

        thread::sleep(Duration::from_secs(1));
        champion.regen_health_mana();
        assert_eq!(champion.stats.health, 100); // 0.5 added to accumulator, not enough to add 1 health
        assert!((champion.stats.health_regen_acc - 0.5).abs() < f32::EPSILON);

        thread::sleep(Duration::from_secs(1));
        champion.regen_health_mana();
        assert_eq!(champion.stats.health, 101); // acc becomes 1.0, adds 1 health, acc becomes 0
        assert!(champion.stats.health_regen_acc.abs() < f32::EPSILON);

        // Test with 1.5 hp/sec
        champion.stats.hp_per_sec = 1.5;
        champion.stats.health = 100;
        champion.stats.health_regen_acc = 0.0;

        thread::sleep(Duration::from_secs(1));
        champion.regen_health_mana();
        assert_eq!(champion.stats.health, 101); // acc becomes 1.5, adds 1, acc becomes 0.5
        assert!((champion.stats.health_regen_acc - 0.5).abs() < f32::EPSILON);

        thread::sleep(Duration::from_secs(1));
        champion.regen_health_mana();
        assert_eq!(champion.stats.health, 103); // acc becomes 0.5 + 1.5 = 2.0, adds 2, acc becomes 0.0
        assert!(champion.stats.health_regen_acc.abs() < f32::EPSILON);
    }

    // Helper function to create a basic Item for testing
    fn create_test_item(
        id: u32,
        name: &str,
        cost: u32,
        crafting_cost: Option<u32>,
        required: Option<Vec<u32>>,
    ) -> Item {
        Item {
            id,
            name: name.to_string(),
            cost,
            crafting_cost,
            required,
            stats: ItemStats {
                attack_damage: None,
                attack_speed: None,
                magic_power: None,
                health: None,
                armor: None,
                mana: None,
                health_regen: None,
            },
        }
    }

    #[test]
    fn test_add_item_crafting_not_enough_gold_and_missing_items() {
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 0, 0, champion_stats, spell_stats);
        champion.gold = 50; // Not enough for crafting (100) or buyout (700)

        let sword_of_power = create_test_item(1, "Sword of Power", 300, None, None);
        // Add one sword, but two are required
        champion.inventory[0] = Some(sword_of_power.clone());

        let double_edged_sword =
            create_test_item(8, "Double-edged Sword", 700, Some(100), Some(vec![1, 1]));

        let result = champion.add_item(double_edged_sword);
        assert!(result.is_err());
        // Expecting NotEnoughGold because gold check happens before missing items check for buyout
        assert_eq!(result.unwrap_err(), GameError::NotEnoughGold);
        // Inventory should remain unchanged
        assert_eq!(champion.inventory[0], Some(sword_of_power));
        assert_eq!(champion.gold, 50);
    }

    #[test]
    fn test_add_item_crafting_success() {
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 0, 0, champion_stats, spell_stats);
        champion.gold = 200; // Enough for crafting cost (100)

        let sword_of_power_1 = create_test_item(1, "Sword of Power", 300, None, None);
        let sword_of_power_2 = create_test_item(1, "Sword of Power", 300, None, None);
        // Add two swords to inventory
        champion.inventory[0] = Some(sword_of_power_1);
        champion.inventory[1] = Some(sword_of_power_2);

        let double_edged_sword =
            create_test_item(8, "Double-edged Sword", 700, Some(100), Some(vec![1, 1]));
        let expected_gold_after_craft =
            champion.gold - double_edged_sword.crafting_cost.unwrap() as u16;

        let result = champion.add_item(double_edged_sword.clone());
        assert!(result.is_ok());
        // Inventory should contain the crafted item, and required items should be removed
        assert_eq!(champion.inventory[0], Some(double_edged_sword));
        assert_eq!(champion.inventory[1], None);
        assert_eq!(champion.gold, expected_gold_after_craft);
    }

    #[test]
    fn test_add_item_buyout_success_without_required_items() {
        let champion_stats = create_default_champion_stats();
        let spell_stats = HashMap::new();
        let mut champion = Champion::new(1, Team::Red, 0, 0, champion_stats, spell_stats);
        champion.gold = 800; // Enough for buyout cost (700)

        // No required items in inventory

        let double_edged_sword =
            create_test_item(8, "Double-edged Sword", 700, Some(100), Some(vec![1, 1]));
        let expected_gold_after_buyout = champion.gold - double_edged_sword.cost as u16;

        let result = champion.add_item(double_edged_sword.clone());
        assert!(result.is_ok());
        // Inventory should contain the item, and gold should be reduced by buyout cost
        assert_eq!(champion.inventory[0], Some(double_edged_sword));
        assert_eq!(champion.gold, expected_gold_after_buyout);
    }

    #[test]
    fn test_pierce_spell_and_on_hit_effect() {
        let mut board = create_dummy_board(10, 10);
        let mut pm = ProjectileManager::new();
        let champion_stats = create_default_champion_stats();
        let mut spell_stats_map: HashMap<u8, Box<dyn Spell>> = HashMap::new();

        let pierce_spell_stats = SpellStats {
            id: 2, // some id
            mana_cost: 10,
            cooldown_secs: 5,
            range: 0,
            width: 0,
            speed: 0,
            base_attack_damage: 10,
            base_magic_damage: 0,
            damage_ratio: 0.5,
            magic_ratio: 0.,
            effect_duration: Some(5),
            is_heal: Some(false),
        };
        let pierce_spell = Box::new(PierceSpell::new(pierce_spell_stats));
        spell_stats_map.insert(1, pierce_spell); // Use action 2 (id 1)

        let mut champion = Champion::new(1, Team::Red, 2, 2, champion_stats, spell_stats_map);
        champion.stats.mana = 100; // Ensure enough mana

        // The champion has no on-hit effects initially.
        assert!(champion.on_hit_effects.is_empty());

        // Cast the pierce spell (Action2)
        let action = Action::Action2;
        let result = champion.take_action(&action, &mut board, &mut pm);
        assert!(result.is_ok());

        // The champion should now have one on-hit effect.
        assert_eq!(champion.on_hit_effects.len(), 1);
        match &champion.on_hit_effects[0] {
            GameplayEffect::Buff(_) => (),
            _ => panic!("Effect should be a buff"),
        }

        // Now, let's make the champion attack.
        // For that, there must be a target.
        let enemy_id = 2;
        let enemy_team = Team::Blue;
        let enemy_row = 2;
        let enemy_col = 3;
        board.place_cell(
            CellContent::Champion(enemy_id, enemy_team),
            enemy_row as usize,
            enemy_col as usize,
        );

        // Ensure champion can attack (cooldown ready)
        champion.last_attacked =
            Instant::now() - champion.stats.attack_speed - Duration::from_secs(1);

        // The champion should have a target.
        assert!(champion.get_potential_target(&board).is_some());

        // Perform the attack
        let attack_action_opt = champion.can_attack();
        assert!(attack_action_opt.is_some());

        if let Some(AttackAction::Melee {
            animation: _,
            effects,
        }) = attack_action_opt
        {
            // The attack action should contain the on-hit effect.
            // We should have the first effect as the on-hit and the second one as the aa
            assert_eq!(effects.len(), 2);
            match &effects[0] {
                GameplayEffect::Buff(_) => (),
                _ => panic!("Effect in attack action should be a buff"),
            }
        } else {
            panic!("Expected a Melee attack action");
        }

        // After the attack, the champion's on_hit_effects should be empty again.
        assert!(champion.on_hit_effects.is_empty());
    }
}
