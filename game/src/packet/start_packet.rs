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
