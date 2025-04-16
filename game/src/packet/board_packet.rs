use byteorder::{BigEndian, ByteOrder};
use bytes::BufMut;
use bytes::BytesMut;

#[derive(Debug)]
pub struct BoardPacket {
    pub version: u8,
    pub code: u8,
    pub points: u16,
    pub length: u16,
    pub encoded_board: Vec<u8>,
}

impl BoardPacket {
    pub fn new(encoded_board: Vec<u8>) -> Self {
        println!("{:?}", encoded_board);
        let length = encoded_board.len().try_into().unwrap();
        BoardPacket {
            version: 1,
            code: 9,
            points:0,
            length,
            encoded_board,
        }
    }

    pub fn serialize(&self) -> BytesMut {
        let mut buffer = BytesMut::new();
        buffer.put_u8(self.version);
        buffer.put_u8(self.code);
        buffer.put_u16(self.points);
        buffer.put_u16(self.length);
        buffer.extend_from_slice(&self.encoded_board);
        buffer
    }
}
