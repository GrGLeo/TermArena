
#[derive(Debug, Clone, Copy)]
pub struct ActionPacket {
    pub version: u8,
    pub code: u8,
    pub action: u8,
}

impl ActionPacket {
    pub fn deserialize(bytes: &[u8]) -> Result<Self, &'static str> {
        if bytes.len() != 3 {
            return Err("Action packet must be  3 bytes long");
        }
        let version = bytes[0];
        let code = bytes[1];
        let action = bytes[2];

        Ok(ActionPacket { version, code, action })
    }
}
