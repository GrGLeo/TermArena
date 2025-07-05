use bytes::BufMut;
use bytes::BytesMut;

#[derive(Debug)]
pub struct StartPacket {
    pub version: u8,
    pub code: u8,
    pub success: u8,
}

impl StartPacket {
    pub fn new(success: u8) -> Self {
        StartPacket {
            version: 1,
            code: 7,
            success,
        }
    }

    pub fn serialize(&self) -> BytesMut {
        let mut buffer = BytesMut::new();
        buffer.put_u8(self.version);
        buffer.put_u8(self.code);
        buffer.put_u8(self.success);
        buffer
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use bytes::BufMut; // Import BufMut for creating expected BytesMut

    #[test]
    fn test_start_packet_new() {
        // Test case with success = 0
        let success_fail = 0;
        let packet_fail = StartPacket::new(success_fail);
        assert_eq!(packet_fail.version, 1);
        assert_eq!(packet_fail.code, 7);
        assert_eq!(packet_fail.success, success_fail);

        // Test case with success = 1
        let success_ok = 1;
        let packet_ok = StartPacket::new(success_ok);
        assert_eq!(packet_ok.version, 1);
        assert_eq!(packet_ok.code, 7);
        assert_eq!(packet_ok.success, success_ok);
    }

    #[test]
    fn test_start_packet_serialize() {
        let success_value = 1;
        let packet = StartPacket::new(success_value);

        let serialized_buffer = packet.serialize();

        // Manually construct the expected byte buffer
        let mut expected_buffer = BytesMut::new();
        expected_buffer.put_u8(packet.version); // 1
        expected_buffer.put_u8(packet.code); // 7
        expected_buffer.put_u8(packet.success); // success_value (e.g., 1)

        assert_eq!(
            serialized_buffer, expected_buffer,
            "Serialized buffer should match expected format"
        );
    }
}
