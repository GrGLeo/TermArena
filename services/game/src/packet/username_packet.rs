use std::io::{self, ErrorKind};

pub struct UsernamePacket {
    pub version: u8,
    pub code: u8,
    pub username: String,
}

impl UsernamePacket {
    pub fn deserialize(buffer: &[u8]) -> io::Result<Self> {
        if buffer.len() < 3 {
            return Err(io::Error::new(
                ErrorKind::InvalidData,
                "SpellSelectionPacket buffer too short",
            ));
        }
        let username_len = buffer[2] as usize;
        if buffer.len() < 3 + username_len {
            return Err(io::Error::new(
                ErrorKind::InvalidData,
                "SpellSelectionPacket buffer incomplete",
            ));
        }
        let username = String::from_utf8(buffer[3..3 + username_len].to_vec()).map_err(|_| {
            io::Error::new(ErrorKind::InvalidData, "Invalid UTF-8 in username")
        })?;
        Ok(UsernamePacket {
            version: buffer[0],
            code: buffer[1],
            username,
        })
    }
}
