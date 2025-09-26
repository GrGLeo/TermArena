use super::CODE_END_GAME;

#[derive(Debug)]
pub struct EndGamePacket {
    pub version: u8,
    pub code: u8,
    pub winner: bool,
}

impl EndGamePacket {
    pub fn new(winner: bool) -> Self {
        EndGamePacket {
            version: 1,
            code: CODE_END_GAME,
            winner,
        }
    }

    pub fn serialize(&self) -> Vec<u8> {
        let mut bytes = Vec::new();
        bytes.push(self.version);
        bytes.push(self.code);
        bytes.push(self.winner as u8);
        bytes
    }
}
