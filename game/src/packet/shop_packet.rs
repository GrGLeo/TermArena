use bytes::BufMut;
use bytes::BytesMut;
use byteorder::{BigEndian, ReadBytesExt};
use std::io::Cursor;

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

        Ok(ShopRequestPacket { version, code })
    }
}

pub struct ShopResponsePacket {
    pub version: u8,
    pub code: u8,
    health: u16,
    mana: u16,
    damage: u16,
    armor: u16,
    gold: u16,
    inventory: Vec<u16>,
}

impl ShopResponsePacket {
    pub fn new(stats: (u16, u16, u16, u16, u16), inventory: Vec<u16>) -> Self {
        ShopResponsePacket {
            version: 1,
            code: 15,
            health: stats.0,
            mana: stats.1,
            damage: stats.2,
            armor: stats.3,
            gold: stats.4,
            inventory,
        }
    }

    pub fn serialize(&self) -> BytesMut {
        let mut buffer = BytesMut::new();
        buffer.put_u8(self.version);
        buffer.put_u8(self.code);
        buffer.put_u16(self.health);
        buffer.put_u16(self.mana);
        buffer.put_u16(self.damage);
        buffer.put_u16(self.armor);
        buffer.put_u16(self.gold);
        // Always write 6 inventory slots
        for i in 0..6 {
            if i < self.inventory.len() {
                buffer.put_u16(self.inventory[i]);
            } else {
                buffer.put_u16(0); // Empty slot
            }
        }
        buffer
    }
}

#[derive(Debug, Clone, Copy)]
pub struct PurchaseItemPacket {
    pub version: u8,
    pub code: u8,
    pub item_id: u16,
}

impl PurchaseItemPacket {
    pub fn deserialize(bytes: &[u8]) -> Result<Self, &'static str> {
        if bytes.len() != 2 {
            return Err("PurchaseItemPacket payload must be 2 bytes long");
        }
        let mut cursor = Cursor::new(bytes);
        let item_id = cursor.read_u16::<BigEndian>().unwrap();

        Ok(PurchaseItemPacket {
            version: 1, // Version and code are handled in the main loop
            code: 16,
            item_id,
        })
    }
}
