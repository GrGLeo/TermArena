use crate::game::cell::PlayerId;

#[derive(Debug)]
pub struct Champion {
    pub player_id: PlayerId,
    pub row: u16,
    pub col: u16,
}

impl Champion {
    pub fn new(player_id: PlayerId, row: u16, col: u16) -> Self {
        Champion { player_id, row, col }
    }
}

