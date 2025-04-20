
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_action_packet_deserialize() {
        // Test case with a valid byte slice
        let valid_bytes: [u8; 3] = [1, 8, 3]; // version=1, code=8, action=3
        let packet_result = ActionPacket::deserialize(&valid_bytes);

        assert!(packet_result.is_ok(), "Deserializing a valid byte slice should succeed");

        let packet = packet_result.unwrap();
        assert_eq!(packet.version, 1, "Deserialized version should match byte slice");
        assert_eq!(packet.code, 8, "Deserialized code should match byte slice");
        assert_eq!(packet.action, 3, "Deserialized action should match byte slice");

        // Test case with an invalid byte slice length (too short)
        let invalid_bytes_short: [u8; 2] = [1, 8];
        let packet_result_short = ActionPacket::deserialize(&invalid_bytes_short);

        assert!(packet_result_short.is_err(), "Deserializing a short byte slice should return an error");
        assert_eq!(packet_result_short.unwrap_err(), "Action packet must be  3 bytes long", "Error message for short slice should be correct");


        // Test case with an invalid byte slice length (too long)
        let invalid_bytes_long: [u8; 4] = [1, 8, 3, 99];
        let packet_result_long = ActionPacket::deserialize(&invalid_bytes_long);

        assert!(packet_result_long.is_err(), "Deserializing a long byte slice should return an error");
         assert_eq!(packet_result_long.unwrap_err(), "Action packet must be  3 bytes long", "Error message for long slice should be correct");
    }
}
