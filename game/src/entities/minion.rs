use crate::entities::common::Stats;
pub type MinionId = String;

struct Minion {
    minion_id: MinionId,
    position: super::common::Position,
    stats: Stats,
}
