// Packet code constants for the game service
// These must match the constants in the Go shared package

pub const CODE_GAME_START: u8 = 100;
pub const CODE_ACTION: u8 = 101;
pub const CODE_BOARD: u8 = 102;
pub const CODE_DELTA: u8 = 103;
pub const CODE_GAME_CLOSE: u8 = 104;
pub const CODE_END_GAME: u8 = 105;
pub const CODE_SPELL_SELECTION: u8 = 106;
pub const CODE_SHOP_REQUEST: u8 = 107;
pub const CODE_SHOP_RESPONSE: u8 = 108;
pub const CODE_PURCHASE_ITEM: u8 = 109;

// For backwards compatibility with existing code
pub const CODE_USERNAME: u8 = 106; // Alias for spell selection

pub mod action_packet;
pub mod board_packet;
pub mod end_game_packet;
pub mod shop_packet;
pub mod username_packet;
pub mod start_packet;
