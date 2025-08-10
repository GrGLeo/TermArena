#[derive(Debug)]
pub struct EndGamePacket {
    pub winner: bool,
}

impl EndGamePacket {
    pub fn new(winner: bool) -> Self {
        EndGamePacket { winner }
    }

    pub fn serialize(&self) -> Vec<u8> {
        let mut bytes = Vec::new();
        bytes.push(1);
        bytes.push(12);
        bytes.push(self.winner as u8);
        bytes
    }
}
