use std::time::{Duration, Instant};

use super::Buff;

#[derive(Debug, Clone)]
pub struct BlueBuff {
    pub duration_remaining: Duration,
    pub applied_at: Instant,
    mana_bonus: f32,
    last_tick: Instant,
}

impl BlueBuff {
    pub fn new(duration: u64) -> BlueBuff {
        BlueBuff {
            duration_remaining: Duration::from_secs(duration),
            applied_at: Instant::now(),
            mana_bonus: 0.0,
            last_tick: Instant::now(),
        }
    }
}

impl Buff for BlueBuff {
    fn id(&self) -> &str {
        "BlueBuff"
    }

    fn on_apply(&mut self, target: &mut dyn super::HasBuff) {
        self.mana_bonus = target.get_stats_mut().mp_per_sec
    }

    fn on_tick(&mut self, target: &mut dyn super::HasBuff) -> bool {
        if self.last_tick.elapsed() >= Duration::from_secs(1) {
            target.get_stats_mut().mana_regen_acc += self.mana_bonus;
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
