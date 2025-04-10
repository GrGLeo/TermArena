
pub type PlayerId = String;

struct Player {
    player_id: PlayerId,
    position: super::common::Position,
    stats: Stats,
}

