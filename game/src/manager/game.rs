use crate::entities::player::Player;
use crate::{board::board::Board, entities::player::PlayerId};
use std::sync::Arc;
use tokio::net::TcpStream;
use tokio::sync::Mutex;
use std::collections::HashMap;

#[derive(Debug)]
pub struct GameManager {
    board: Board,
    connection: Arc<Mutex<HashMap<PlayerId, TcpStream>>>,
    character: Arc<Mutex<HashMap<PlayerId, Player>>>,
    started: bool,
}

impl GameManager {
    pub fn new() -> Self {
        GameManager {
            board: Board::new(),
            connection: Arc::new(Mutex::new(HashMap::new())),
            character: Arc::new(Mutex::new(HashMap::new())),
            started: false,
        }
    }

    pub async fn handle_connection(&mut self, player_id: PlayerId, stream: TcpStream) {
        let mut connection = self.connection.lock().await;
        connection.insert(player_id, stream);
    }
}
