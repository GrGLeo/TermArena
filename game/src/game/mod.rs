pub mod board;
pub mod cell;
pub mod entities;

use crate::packet::{board_packet::BoardPacket, start_packet};
pub use board::Board;
use bytes::BytesMut;
pub use cell::{BaseTerrain, Cell, PlayerId};
pub use entities::champion::Champion;
use tokio::sync::mpsc;

use std::collections::HashMap;

#[derive(Debug, Clone, Copy)]
pub enum Action {
    MoveUp,
    MoveDown,
    MoveLeft,
    MoveRight,
    Action1,
    Action2,
    InvalidAction,
}

pub type ClientMessage = BytesMut;

#[derive(Debug)]
pub struct GameManager {
    players_count: usize,
    max_players: usize,
    pub game_started: bool,
    player_action: HashMap<PlayerId, Action>,
    champions: HashMap<PlayerId, Champion>,
    pub client_channel: HashMap<PlayerId, mpsc::Sender<ClientMessage>>,
    board: Board,
    pub tick: u64
}

impl GameManager {
    pub fn new() -> Self {
        println!("Initializing GameManager...");
        /*
        let rows = 200;
        let cols = 500;
        let board = Board::new(rows, cols);
        */
        let file_path = "game/assets/map.json";
        let board = match Board::from_json(file_path) {
            Ok(board) => board,
            Err(e) => {
                eprintln!("Failed to initialize the board from {}: {}", file_path, e);
                std::process::exit(1);
            }
        };

        GameManager {
            players_count: 0,
            max_players: 1,
            game_started: false,
            player_action: HashMap::new(),
            champions: HashMap::new(),
            client_channel: HashMap::new(),
            board,
            tick: 20,
        }
    }

    pub fn print_game_state(&self) {
        println!(
            "Player connected: {}/{}",
            self.players_count, self.max_players
        );
        if self.player_action.is_empty() {
            println!("No action received");
        } else {
            for (player_id, action) in &self.player_action {
                println!("Player: {} / Action: {:?}", player_id, action);
            }
        }
        println!("Board size: {}.{}", self.board.rows, self.board.cols);
    }

    pub fn clear_action(&mut self) {
        self.player_action.clear();
    }

    pub fn add_player(&mut self) -> Option<PlayerId> {
        if self.players_count < self.max_players {
            self.players_count += 1;
            let player_id = self.players_count;
            // Assign Champion to player, and place it on the board
            {
                let row = 199;
                let col = 0;
                let champion = Champion::new(player_id, row, col);
                self.champions.insert(player_id, champion);
                self.board.place_cell(cell::CellContent::Champion(player_id), row as usize, col as usize);
            }

            // We check if we can start the game and send a Start to each player
            if self.players_count == self.max_players {
                self.game_started = true;
            }
            Some(player_id)
        } else {
            None
        }
    }

    pub fn remove_player(&mut self, player_id: &PlayerId) {
        if self.players_count > 0 {
            self.players_count -= 1;
            self.player_action.remove(&player_id);
            self.client_channel.remove(&player_id);
            println!(
                "Player {} disconnected. Total player now: {}/{}",
                player_id, self.players_count, self.max_players
            );
            if self.game_started && self.players_count < self.max_players {
                self.game_started = false;
            }
        } else {
            println!("Warning: Tried to remove player, but player count already at 0.");
        }
    }

    pub fn store_player_action(&mut self, player_id: PlayerId, action_value: u8) {
        let action = match action_value {
            1 => Action::MoveUp,
            2 => Action::MoveDown,
            3 => Action::MoveLeft,
            4 => Action::MoveRight,
            5 => Action::Action1,
            6 => Action::Action2,
            _other => Action::InvalidAction,
        };
        self.player_action.insert(player_id, action);
    }

    pub async fn send_to_player(&self, player_id: PlayerId, message: ClientMessage) {
        if let Some(sender) = self.client_channel.get(&player_id) {
            let sender_clone = sender.clone();
            // We use spawn to send without blocking the game manager lock
            tokio::spawn(async move {
                if let Err(e) = sender_clone.send(message).await {
                    eprintln!("Error sending message to player {}: {}", player_id, e);
                }
            });
        } else {
            eprintln!(
                "Attempted to send message to disconnected or non-existent player {}",
                player_id
            );
        }
    }

    pub fn game_tick(&mut self) -> HashMap<PlayerId, ClientMessage> {
        println!("---- Game Tick -----");
        self.print_game_state();

        let mut updates = HashMap::new();

        // --- Game Logic ---
        // Iterate through player action
        for (player_id, action) in &self.player_action {
            if let Some(player_champ) = self.champions.get_mut(&player_id) {
                if let Err(e) = player_champ.take_action(action, &mut self.board) {
                    println!("Error on player action: {}", e);
                }
            }
        }

        // --- Send per player there board view ---
        for (player_id, champion) in &self.champions {
            // 1. Get player-specific board view
            let board_rle_vec = self.board.run_length_encode(champion.row, champion.col);
            // 2. Create the board packet
            let board_packet = BoardPacket::new(board_rle_vec);
            let serialized_packet = board_packet.serialize();

            // 3. Store the serialized packet to be sent later
            updates.insert(*player_id, serialized_packet);
        }
        println!("--------------------");
        updates
    }
}
