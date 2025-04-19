use crate::game::{cell::CellAnimation, TowerId};


pub struct TowerAnimation {
    pub tower_id: TowerId,
    cycle: u8,
    counter: u8,
    row: Option<u16>,
    col: Option<u16>,
    animation: CellAnimation,
}

impl TowerAnimation {
    pub fn new(tower_id: TowerId) -> Self {
        TowerAnimation {
            tower_id,
            cycle: 8,
            counter: 0,
            row: None,
            col: None,
            animation: CellAnimation::TowerHit,
        }
    }
}
