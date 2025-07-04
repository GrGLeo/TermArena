use crate::game::cell::Team;

#[derive(Debug)]
pub struct EndGamePacket {
    pub winner: Team,
}

impl EndGamePacket {
    pub fn new(winner: Team) -> Self {
        EndGamePacket { winner }
    }

    pub fn serialize(&self) -> Vec<u8> {
        let mut bytes = Vec::new();
        bytes.push(1);
        bytes.push(12);
        bytes.push(match self.winner {
            Team::Red => 0,
            Team::Blue => 1,
        });
        bytes
    }
}
