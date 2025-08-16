pub mod stun_buff;
pub mod health_buff;
use std::{fmt::Debug, time::Duration};

use health_buff::RedBuff;

use super::entities::Stats;

pub trait HasBuff {
    fn get_stats_mut(&mut self) -> &mut Stats;

    // StunBuff
    fn is_stunned(&self) -> bool;
    fn set_stunned(&mut self, stunned: bool, duration: Option<Duration>);

}

pub trait Buff: Send + Sync + Debug {
    fn clone_box(&self) -> Box<dyn Buff>;
    fn id(&self) -> &str;
    fn on_apply(&mut self, target: &mut dyn HasBuff);
    fn on_tick(&mut self, target: &mut dyn HasBuff) -> bool;
    fn on_remove(&mut self, target: &mut dyn HasBuff);
}

pub fn create_buff(buff_name: &str) -> Option<Box<dyn Buff>> {
    match buff_name {
        "red_buff" => Some(Box::new(RedBuff::new(120))),
        _ => {
            None
        }
    }
}
