use std::io::{self, ErrorKind};

pub struct SpellSelectionPacket {
    pub version: u8,
    pub code: u8,
    pub spell1: u8,
    pub spell2: u8,
}

impl SpellSelectionPacket {
    pub fn deserialize(buffer: &[u8]) -> io::Result<Self> {
        if buffer.len() < 4 {
            return Err(io::Error::new(
                ErrorKind::InvalidData,
                "SpellSelectionPacket buffer too short",
            ));
        }
        Ok(SpellSelectionPacket {
            version: buffer[0],
            code: buffer[1],
            spell1: buffer[2],
            spell2: buffer[3],
        })
    }
}
