use crate::game::cell::Team;

const PACKET_VERSION: u8 = 1;
const PACKET_CODE: u8 = 12;


#[derive(Debug)]
pub struct EndGamePacket {
    pub winner: Team,
}

impl EndGamePacket {
    pub fn new(winner: Team) -> Self {
        Self { winner }
    }

    pub fn serialize(&self) -> Vec<u8> {
        let mut bytes = Vec::new();
        bytes.push(PACKET_VERSION);
        bytes.push(PACKET_CODE);
        bytes.push(match self.winner {
            Team::Red => 0,
            Team::Blue => 1,
        });
        bytes
    }
}
