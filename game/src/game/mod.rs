pub mod board;
pub mod cell;
pub mod entities;
pub mod animation;

use crate::packet::board_packet::BoardPacket;
use animation::{AnimationCommand, AnimationTrait};
pub use board::Board;
use bytes::BytesMut;
pub use cell::{BaseTerrain, Cell, CellContent, PlayerId, TowerId, MinionId};
pub use entities::champion::Champion;
use entities::{tower::Tower, Fighter, Target};
use tokio::sync::mpsc;

use std::{collections::HashMap, usize};

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

pub struct GameManager {
    players_count: usize,
    max_players: usize,
    pub game_started: bool,
    player_action: HashMap<PlayerId, Action>,
    champions: HashMap<PlayerId, Champion>,
    towers: HashMap<TowerId, Tower>,
    animations: Vec<Box<dyn AnimationTrait>>,
    pub client_channel: HashMap<PlayerId, mpsc::Sender<ClientMessage>>,
    board: Board,
    pub tick: u64,
}

impl GameManager {
    pub fn new() -> Self {
        println!("Initializing GameManager...");
        let file_path = "game/assets/map.json";
        let mut board = match Board::from_json(file_path) {
            Ok(board) => board,
            Err(e) => {
                eprintln!("Failed to initialize the board from {}: {}", file_path, e);
                std::process::exit(1);
            }
        };
        let mut towers: HashMap<TowerId, Tower> = HashMap::new();
        // Tower placement
        {
            let tower_1 = Tower::new(1, 1, 196, 150);
            tower_1.place_tower(&mut board);
            towers.insert(tower_1.tower_id, tower_1);
            let tower_2 = Tower::new(2, 2, 150, 196);
            tower_2.place_tower(&mut board);
            towers.insert(tower_2.tower_id, tower_2);
        }

        GameManager {
            players_count: 0,
            max_players: 1,
            game_started: false,
            player_action: HashMap::new(),
            champions: HashMap::new(),
            towers,
            animations: Vec::new(),
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
                let champion = Champion::new(player_id, 1, row, col);
                self.champions.insert(player_id, champion);
                self.board.place_cell(
                    cell::CellContent::Champion(player_id, 1),
                    row as usize,
                    col as usize,
                );
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
        let mut new_animations: Vec<Box<dyn AnimationTrait>> = Vec::new();
        let mut pending_damages: Vec<(Target, u8)> = Vec::new();

        // --- Game Logic ---
        // Player turn
        for (player_id, champ) in &mut self.champions {
            // 0. Check death and replace
            // BUG: Champ dead can still move but is replace each tick
            if champ.is_dead() {
                champ.place_at_base(&mut self.board)
            }
            // 1. Iterate through player action
            if let Some(action) = self.player_action.get(&player_id) {
                if let Err(e) = champ.take_action(action, &mut self.board) {
                    println!("Error on player action: {}", e);
                }
            }

            // 2. auto_attack
            if let Some(enemy) = champ.scan_range(&self.board) {
                match &enemy.content {
                    Some(content) => {
                        println!("Got content: {:?}", content);
                        match content {
                            CellContent::Tower(id, _) => {
                                if let Some((damage, animation)) = champ.can_attack() {
                                    new_animations.push(animation);
                                    pending_damages.push((Target::Tower(*id), damage))
                                }
                            }
                            CellContent::Minion(_, _) => {
                                todo!()
                            }
                            CellContent::Champion(id, _) => {
                                if let Some((damage, animation)) = champ.can_attack() {
                                    self.animations.push(animation);
                                    pending_damages.push((Target::Champion(*id), damage))
                                }
                            }
                            _ => break,
                        }
                    }
                    None => break,
                }
            }
        }
        
        // 3. Apply dealt damages
        pending_damages.into_iter().for_each(|(target, damage)| {
            match target {
                Target::Tower(id) => {
                    if let Some(tower) = self.towers.get_mut(&id) {
                        tower.take_damage(damage);
                        if tower.is_destroyed() {
                            tower.destroy_tower(&mut self.board);
                            self.towers.remove(&id);
                        }
                    }
                }
                Target::Minion(_) => {
                    todo!()
                }
                Target::Champion(id) => {
                    if let Some(champ) = self.champions.get_mut(&id) {
                        champ.take_damage(damage);
                    }
                }
            }
        });

        // Tower turn
        // 1. Scan range
        // 2. attack closest enemy
        self.tower_turn();

        // Render animation
        let mut kept_animations: Vec<Box<dyn AnimationTrait>> = Vec::new();
        let mut animation_commands_executable: Vec<AnimationCommand> = Vec::new();

        // 1. clear past frame animation
        for anim in &self.animations {
            if let Some((row, col)) = anim.get_last_drawn_pos() {
                animation_commands_executable.push(AnimationCommand::Clear { row, col })
            }
        }
        // 2. Process next frame animations
        for mut anim in self.animations.drain(..) {
            let owner_pos = if let Some(champ) = self.champions.get(&anim.get_owner_id()) {
                Some((champ.row, champ.col))
            } else if let Some(tower) = self.towers.get(&anim.get_owner_id()) {
                Some((tower.row, tower.col))
            } else {
                None // Owner might have been removed
            };

            if let Some((owner_row, owner_col)) = owner_pos {
                let command = anim.next_frame(owner_row, owner_col);
                match command {
                    AnimationCommand::Done => {
                    }
                    AnimationCommand::Draw { .. } => {
                        animation_commands_executable.push(command);
                        kept_animations.push(anim);
                    }
                    AnimationCommand::Clear { .. } => {
                        // This command should be handle before
                    }
                } 
            } else {
                // Owner is gone, animation should finish and clear in its last frame
            }
        }
        kept_animations.extend(new_animations);
        self.animations = kept_animations;

        // 3. Execute animation command
        for command in animation_commands_executable {
            match command {
                AnimationCommand::Draw { row, col, animation_type } => {
                    // Add bounds check
                    if row < self.board.rows as u16 && col < self.board.cols as u16 {
                        self.board.place_animation(animation_type, row as usize, col as usize);
                    } else {
                         eprintln!("Animation draw position ({}, {}) out of bounds!", row, col);
                    }
                }
                AnimationCommand::Clear { row, col } => {
                    if row < self.board.rows as u16 && col < self.board.cols as u16 {
                        self.board.clean_animation(row as usize, col as usize);
                    } else {
                        eprintln!("Animation clear position ({}, {}) out of bounds!", row, col);
                    }
                }
                AnimationCommand::Done => {
                    // This command should be handled in the loop above, not executed on the board
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

    fn tower_turn(&mut self) {
        let pending_damages = self.towers
            .iter_mut()
            .map(|(_, tower)| {
                if let Some(enemy) = tower.scan_range(&self.board) {
                    match &enemy.content {
                        Some(content) => {
                            match content {
                                CellContent::Minion(_, _) => {
                                    todo!()
                                }
                                CellContent::Champion(id, _) => {
                                    if let Some((damage, animation)) = tower.can_attack() {
                                        Some((Target::Champion(*id), damage, animation))
                                    } else {
                                        None
                                    }
                                }
                                _ => None,
                            }
                        }
                        None => None,
                    }
                } else {
                    None
                }
            })
            .filter_map(|option| option)
            .collect::<Vec<(Target, u8)>>();

        pending_damages.into_iter().for_each(|(target, damage)| {
            match target {
                Target::Tower(id) => {
                    if let Some(tower) = self.towers.get_mut(&id) {
                        tower.take_damage(damage);
                    }
                }
                Target::Minion(_) => {
                    todo!()
                }
                Target::Champion(id) => {
                    if let Some(champ) = self.champions.get_mut(&id) {
                        champ.take_damage(damage);
                    }
                }
            }
        });
    }
}
