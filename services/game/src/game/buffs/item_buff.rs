use std::time::{Duration, Instant};

use super::Buff;

#[derive(Debug, Clone)]
pub struct HealthRegenItem {
    hp_bonus: f32,
    last_tick: Instant,
}

impl HealthRegenItem {
    pub fn new(hp_bonus: f32) -> HealthRegenItem {
        HealthRegenItem {
            hp_bonus,
            last_tick: Instant::now(),
        }
    }
}

impl Buff for HealthRegenItem {
    fn id(&self) -> &str {
        "health_regen_item"
    }

    fn on_apply(&mut self, _target: &mut dyn super::HasBuff) {
    }

    fn on_tick(&mut self, target: &mut dyn super::HasBuff) -> bool {
        if self.last_tick.elapsed() >= Duration::from_secs(1) {
            target.get_stats_mut().health_regen_acc += self.hp_bonus;
            self.last_tick = Instant::now();
        }
        // Always return false, as item buff are not truly removed
        return false
    }

    fn on_remove(&mut self, _target: &mut dyn super::HasBuff) {
    }

    fn clone_box(&self) -> Box<dyn Buff> {
        Box::new(self.clone())
    }
}

#[derive(Debug, Clone)]
pub struct ThornDamageItem {
    damage: f32
}

impl ThornDamageItem {
    pub fn new(damage: f32) -> ThornDamageItem {
        ThornDamageItem {
            damage
        }
    }
}

impl Buff for ThornDamageItem {
    fn id(&self) -> &str {
        "thorn_item"
    }

    fn on_apply(&mut self, _target: &mut dyn super::HasBuff) {
    }

    fn on_tick(&mut self, _target: &mut dyn super::HasBuff) -> bool {
        return false
    }

    fn on_remove(&mut self, _target: &mut dyn super::HasBuff) {
    }

    fn clone_box(&self) -> Box<dyn Buff> {
        Box::new(self.clone())
    }
}

