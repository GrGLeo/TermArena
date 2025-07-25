use bytes::BufMut;
use bytes::BytesMut;

#[derive(Debug)]
pub struct BoardPacket {
    pub version: u8,
    pub code: u8,
    pub points: u16,
    pub health: u16,
    pub max_health: u16,
    pub mana: u16,
    pub max_mana: u16,
    pub level: u8,
    pub xp: u32,
    pub xp_needed: u32,
    pub length: u16,
    pub encoded_board: Vec<u8>,
}

impl BoardPacket {
    pub fn new(
        health: u16,
        max_health: u16,
        mana: u16,
        max_mana: u16,
        level: u8,
        xp: u32,
        xp_needed: u32,
        encoded_board: Vec<u8>,
    ) -> Self {
        let length = encoded_board.len().try_into().unwrap();
        BoardPacket {
            version: 1,
            code: 9,
            points: 0,
            health,
            max_health,
            mana,
            max_mana,
            level,
            xp,
            xp_needed,
            length,
            encoded_board,
        }
    }

    pub fn serialize(&self) -> BytesMut {
        let mut buffer = BytesMut::new();
        buffer.put_u8(self.version);
        buffer.put_u8(self.code);
        buffer.put_u16(self.points);
        buffer.put_u16(self.health);
        buffer.put_u16(self.max_health);
        buffer.put_u16(self.mana);
        buffer.put_u16(self.max_mana);
        buffer.put_u8(self.level);
        buffer.put_u32(self.xp);
        buffer.put_u32(self.xp_needed);
        buffer.put_u16(self.length);
        buffer.extend_from_slice(&self.encoded_board);
        buffer
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use bytes::BufMut; // Import BufMut for creating expected BytesMut

    #[test]
    fn test_board_packet_new() {
        let encoded_board_data = vec![0, 1, 1, 2, 3, 1, 1]; // Sample encoded board data
        let health = 400;
        let max_health = 400;
        let mana = 100;
        let max_mana = 100;
        let level = 1;
        let xp = 0;
        let xp_needed = 35;
        let expected_length = encoded_board_data.len() as u16;

        let packet = BoardPacket::new(
            health,
            max_health,
            mana,
            max_mana,
            level,
            xp,
            xp_needed,
            encoded_board_data.clone(),
        );

        assert_eq!(packet.version, 1);
        assert_eq!(packet.code, 9);
        assert_eq!(packet.points, 0); // Points should be 0 as per implementation
        assert_eq!(packet.health, 400);
        assert_eq!(packet.max_health, 400);
        assert_eq!(packet.mana, 100);
        assert_eq!(packet.max_mana, 100);
        assert_eq!(packet.level, 1);
        assert_eq!(packet.xp, 0);
        assert_eq!(packet.xp_needed, 35);
        assert_eq!(packet.length, expected_length);
        assert_eq!(packet.encoded_board, encoded_board_data);
    }

    #[test]
    fn test_board_packet_serialize() {
        let encoded_board_data = vec![0, 1, 1, 2, 3, 1, 1]; // Sample encoded board data
        let health = 300;
        let max_health = 400;
        let mana = 100;
        let max_mana = 100;
        let level = 1;
        let xp = 0;
        let xp_needed = 35;
        let packet = BoardPacket::new(
            health,
            max_health,
            mana,
            max_mana,
            level,
            xp,
            xp_needed,
            encoded_board_data.clone(),
        );

        let serialized_buffer = packet.serialize();

        // Manually construct the expected byte buffer
        let mut expected_buffer = BytesMut::new();
        expected_buffer.put_u8(packet.version); // 1
        expected_buffer.put_u8(packet.code); // 9
        expected_buffer.put_u16(packet.points); // 0 (as BigEndian)
        expected_buffer.put_u16(packet.health); // 400 (as BigEndian)
        expected_buffer.put_u16(packet.max_health); // 400 (as BigEndian)
        expected_buffer.put_u16(packet.mana); // 400 (as BigEndian)
        expected_buffer.put_u16(packet.max_mana); // 400 (as BigEndian)
        expected_buffer.put_u8(packet.level);
        expected_buffer.put_u32(packet.xp);
        expected_buffer.put_u32(packet.xp_needed);
        expected_buffer.put_u16(packet.length); // encoded_board_data.len() as u16 (as BigEndian)
        expected_buffer.extend_from_slice(&packet.encoded_board); // [0, 1, 1, 2, 3, 1, 1]

        assert_eq!(
            serialized_buffer, expected_buffer,
            "Serialized buffer should match expected format"
        );
    }
}
