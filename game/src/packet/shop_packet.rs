use bytes::BufMut;
use bytes::BytesMut;

pub struct ShopRequestPacket {
    pub version: u8,
    pub code: u8,
}

impl ShopRequestPacket {
    pub fn deserialize(bytes: &[u8]) -> Result<Self, &'static str> {
        if bytes.len() != 2 {
            return Err("Action packet must be  3 bytes long");
        }
        let version = bytes[0];
        let code = bytes[1];

        Ok(ShopRequestPacket{
            version,
            code,
        })
    }
}

pub struct ShopResponsePacket {
    pub version: u8,
    pub code: u8,
}

impl ShopResponsePacket {
    pub fn new() -> Self {
        ShopResponsePacket {
            version: 1,
            code: 15,
        }
    }

    pub fn serialize(&self) -> BytesMut {
        let mut buffer = BytesMut::new();
        buffer.put_u8(self.version);
        buffer.put_u8(self.code);
        buffer
    }
}
