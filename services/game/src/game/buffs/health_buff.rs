use std::time::{Duration, Instant};

use super::Buff;

#[derive(Debug, Clone)]
pub struct RedBuff {
    pub duration_remaining: Duration,
    pub applied_at: Instant,
    hp_bonus: f32,
    last_tick: Instant,
}

impl RedBuff {
    pub fn new(duration: u64) -> RedBuff {
        RedBuff {
            duration_remaining: Duration::from_secs(duration),
            applied_at: Instant::now(),
            hp_bonus: 0.0,
            last_tick: Instant::now(),
        }
    }
}

impl Buff for RedBuff {
    fn id(&self) -> &str {
        "RedBuff"
    }

    fn on_apply(&mut self, target: &mut dyn super::HasBuff) {
        self.hp_bonus = target.get_stats_mut().hp_per_sec
    }

    fn on_tick(&mut self, target: &mut dyn super::HasBuff) -> bool {
        if self.last_tick.elapsed() >= Duration::from_secs(1) {
            target.get_stats_mut().health_regen_acc += self.hp_bonus;
            self.last_tick = Instant::now();
        }
        self.applied_at.elapsed() > self.duration_remaining
    }

    fn on_remove(&mut self, _target: &mut dyn super::HasBuff) {
    }

    fn clone_box(&self) -> Box<dyn Buff> {
        Box::new(self.clone())
    }
}

#[derive(Debug, Clone)]
pub struct DoTBuff {
    pub duration_remaining: Duration,
    pub applied_at: Instant,
    damage_per_sec: u8,
    last_tick: Instant,
}

impl DoTBuff {
    pub fn new(duration: u64, damage_per_sec: u8) -> DoTBuff {
        DoTBuff {
            duration_remaining: Duration::from_secs(duration),
            applied_at: Instant::now(),
            damage_per_sec,

            last_tick: Instant::now(),
        }
    }
}

impl Buff for DoTBuff {
    fn id(&self) -> &str {
        "DoTBuff"
    }

    fn on_apply(&mut self, _target: &mut dyn super::HasBuff) {
    }

    fn on_tick(&mut self, target: &mut dyn super::HasBuff) -> bool {
        if self.last_tick.elapsed() >= Duration::from_secs(1) {
            let health = &mut target.get_stats_mut().health;
            *health = health.saturating_sub(self.damage_per_sec as u16);
            self.last_tick = Instant::now();
        }
        self.applied_at.elapsed() > self.duration_remaining
    }

    fn on_remove(&mut self, _target: &mut dyn super::HasBuff) {
    }

    fn clone_box(&self) -> Box<dyn Buff> {
        Box::new(self.clone())
    }
}

