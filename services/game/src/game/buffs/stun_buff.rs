use std::time::{Duration, Instant};

use super::Buff;

#[derive(Debug, Clone)]
pub struct StunBuff {
    pub duration_remaining: Duration,
    pub applied_at: Instant,
}

impl StunBuff {
    pub fn new(duration: u64) -> StunBuff {
        StunBuff {
            duration_remaining: Duration::from_secs(duration),
            applied_at: Instant::now(),
        }
    }
}

impl Buff for StunBuff {
    fn id(&self) -> &str {
        "Stun"
    }

    fn on_apply(&mut self, target: &mut dyn super::HasBuff) {
        target.set_stunned(true, Some(self.duration_remaining));
    }

    fn on_tick(&mut self, _target: &mut dyn super::HasBuff) -> bool {
        self.applied_at.elapsed() > self.duration_remaining
    }

    fn on_remove(&mut self, target: &mut dyn super::HasBuff) {
        target.set_stunned(false, None);
    }

    fn clone_box(&self) -> Box<dyn Buff> {
        Box::new(self.clone())
    }
}
