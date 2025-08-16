use std::time::{Duration, Instant};

use super::Buff;

#[derive(Debug, Clone)]
pub struct RedBuff {
    pub duration_remaining: Duration,
    pub applied_at: Instant,
}

impl RedBuff {
    pub fn new(duration: u64) -> RedBuff {
        RedBuff {
            duration_remaining: Duration::from_secs(duration),
            applied_at: Instant::now(),
        }
    }
}

impl Buff for RedBuff {
    fn id(&self) -> &str {
        "RedBuff"
    }

    fn on_apply(&mut self, target: &mut dyn super::HasBuff) {
        target.update_health_regen(true, 2);
    }

    fn on_tick(&mut self, _target: &mut dyn super::HasBuff) -> bool {
        self.applied_at.elapsed() > self.duration_remaining
    }

    fn on_remove(&mut self, target: &mut dyn super::HasBuff) {
        target.update_health_regen(false, 2);
    }

    fn clone_box(&self) -> Box<dyn Buff> {
        Box::new(self.clone())
    }
}
