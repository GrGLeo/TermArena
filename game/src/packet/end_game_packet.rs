use crate::game::cell::Team;

pub struct EndGamePacket {
    pub winner: Team,
}

impl EndGamePacket {
    pub fn new(winner: Team) -> Self {
        Self { winner }
    }

    pub fn serialize(&self) -> Vec<u8> {
        let mut bytes = Vec::new();
        bytes.push(match self.winner {
            Team::Red => 0,
            Team::Blue => 1,
        });
        bytes
    }
}
