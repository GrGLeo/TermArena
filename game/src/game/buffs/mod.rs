pub mod stun_buff;
use std::{fmt::Debug, time::Duration};

pub trait HasBuff {
    fn is_stunned(&self) -> bool;
    fn set_stunned(&mut self, stunned: bool, duration: Option<Duration>);
}

pub trait Buff: Send + Sync + Debug {
    fn id(&self) -> &str;
    fn on_apply(&mut self, target: &mut dyn HasBuff);
    fn on_tick(&mut self, target: &mut dyn HasBuff) -> bool;
    fn on_remove(&mut self, target: &mut dyn HasBuff);
}
