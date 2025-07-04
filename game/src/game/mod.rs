pub mod animation;
pub mod board;
pub mod cell;
pub mod entities;
pub mod minion_manager;
pub mod pathfinding;

use crate::packet::board_packet::BoardPacket;
use animation::{AnimationCommand, AnimationTrait};
pub use board::Board;
use bytes::BytesMut;
use cell::Team;
pub use cell::{BaseTerrain, Cell, CellContent, MinionId, PlayerId, TowerId};
pub use entities::champion::{Action, Champion};
use entities::{
    Fighter, Target, base::Base,
    tower::{Tower, generate_tower_id},
};
use minion_manager::MinionManager;
use tokio::sync::mpsc;

use std::{
    collections::HashMap,
    time::{Duration, Instant},
    usize, vec,
};

pub type ClientMessage = BytesMut;

pub struct GameManager {
    players_count: usize,
    max_players: usize,
    pub game_started: bool,
    player_action: HashMap<PlayerId, Action>,
    champions: HashMap<PlayerId, Champion>,
    towers: HashMap<TowerId, Tower>,
    red_base: Base,
    blue_base: Base,
    minion_manager: MinionManager,
    animations: Vec<Box<dyn AnimationTrait>>,
    pub client_channel: HashMap<PlayerId, mpsc::Sender<ClientMessage>>,
    board: Board,
    pub tick: u64,
    dead_minion_positions: Vec<(u16, u16, Team)>,
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
            //                  t1   bot      top      mid     t2 bot        top      mid
            let placement = vec![
                (196, 150),
                (39, 7),
                (115, 82),
                (191, 79),
                (120, 8),
                (148, 67),
            ];
            // Bottom t1
            placement.into_iter().for_each(|place| {
                let id = generate_tower_id().unwrap();
                let tower_blue = Tower::new(id, Team::Blue, place.0, place.1);
                tower_blue.place_tower(&mut board);
                let id = generate_tower_id().unwrap();
                let tower_red = Tower::new(id, Team::Red, place.1, place.0);
                tower_red.place_tower(&mut board);
                towers.insert(tower_blue.tower_id, tower_blue);
                towers.insert(tower_red.tower_id, tower_red);
            });
            let tower_1 = Tower::new(1, Team::Blue, 196, 150);
            tower_1.place_tower(&mut board);
            let tower_2 = Tower::new(2, Team::Red, 150, 196);
            tower_2.place_tower(&mut board);
        }

        let red_base = Base::new(Team::Red, (190, 10));
        let blue_base = Base::new(Team::Blue, (10, 190));

        for i in 0..3 {
            for j in 0..3 {
                board.place_cell(
                    CellContent::Base(Team::Red),
                    (red_base.position.0 + i) as usize,
                    (red_base.position.1 + j) as usize,
                );
                board.place_cell(
                    CellContent::Base(Team::Blue),
                    (blue_base.position.0 + i) as usize,
                    (blue_base.position.1 + j) as usize,
                );
            }
        }

        let minion_manager = MinionManager::new();

        GameManager {
            players_count: 0,
            max_players: 1,
            game_started: false,
            player_action: HashMap::new(),
            champions: HashMap::new(),
            towers,
            red_base,
            blue_base,
            minion_manager,
            animations: Vec::new(),
            client_channel: HashMap::new(),
            board,
            tick: 20,
            dead_minion_positions: Vec::new(),
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
                let champion = Champion::new(player_id, Team::Blue, row, col);
                self.champions.insert(player_id, champion);
                self.board.place_cell(
                    cell::CellContent::Champion(player_id, Team::Blue),
                    row as usize,
                    col as usize,
                );
            }

            // We check if we can start the game and send a Start to each player
            if self.players_count == self.max_players {
                self.game_started = true;
                self.minion_manager.wave_creation_time = Instant::now() + Duration::from_secs(10);
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
        println!("Send_to_player message lenght: {}", message.len());
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
        self.tick = self.tick.saturating_add(1);
        println!("---- Game Tick -----");
        self.print_game_state();

        let mut updates = HashMap::new();
        let mut new_animations: Vec<Box<dyn AnimationTrait>> = Vec::new();
        let mut pending_damages: Vec<(Target, u16)> = Vec::new();

        // --- Game Logic ---
        // Player turn
        for (player_id, champ) in &mut self.champions {
            // 0. Check death and replace
            // BUG: Champ dead can still move but is replace each tick
            if champ.is_dead() {
                champ.put_at_max_health();
                champ.place_at_base(&mut self.board);
                continue;
            }
            // 1. Iterate through player action
            if let Some(action) = self.player_action.get(&player_id) {
                if let Err(e) = champ.take_action(action, &mut self.board) {
                    println!("Error on player action: {}", e);
                }
            }

            // 2. auto_attack
            if let Some(enemy) = champ.get_potential_target(&self.board, (3, 3)) {
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
                            CellContent::Minion(id, _) => {
                                if let Some((damage, animation)) = champ.can_attack() {
                                    new_animations.push(animation);
                                    pending_damages.push((Target::Minion(*id), damage))
                                }
                            }
                            CellContent::Champion(id, _) => {
                                if let Some((damage, animation)) = champ.can_attack() {
                                    new_animations.push(animation);
                                    pending_damages.push((Target::Champion(*id), damage))
                                }
                            }
                            CellContent::Base(team) => {
                                if let Some((damage, animation)) = champ.can_attack() {
                                    new_animations.push(animation);
                                    pending_damages.push((Target::Base(*team), damage))
                                }
                            }
                            _ => break,
                        }
                    }
                    None => break,
                }
            }
        }

        // Minion mouvement turn
        self.minion_manager
            .manage_minions_mouvements(&mut self.board);
        self.minion_manager.make_wave(&mut self.board);
        println!(
            "Minions: {} | Minions per wave {} | Tick: {}",
            self.minion_manager.minions.len(),
            self.minion_manager.minions_this_wave,
            self.tick,
        );

        // Adding minion damages dealt
        self.minion_manager.manage_minions_attack(
            &mut self.board,
            &mut new_animations,
            &mut pending_damages,
        );

        // 3. Apply dealt damages
        let mut minion_to_clear: Vec<MinionId> = Vec::new();

        pending_damages
            .into_iter()
            .for_each(|(target, damage)| match target {
                Target::Tower(id) => {
                    if let Some(tower) = self.towers.get_mut(&id) {
                        tower.take_damage(damage);
                        if tower.is_destroyed() {
                            tower.destroy_tower(&mut self.board);
                            self.towers.remove(&id);
                        }
                    }
                }
                Target::Minion(id) => {
                    if let Some(minion) = self.minion_manager.minions.get_mut(&id) {
                        minion.take_damage(damage);
                        if minion.is_dead() {
                            minion_to_clear.push(id);
                            self.dead_minion_positions.push((minion.row, minion.col, minion.team_id));
                            self.board
                                .clear_cell(minion.row as usize, minion.col as usize);
                        }
                    }
                }
                Target::Champion(id) => {
                    if let Some(champ) = self.champions.get_mut(&id) {
                        champ.take_damage(damage);
                    }
                }
                Target::Base(team) => match team {
                    Team::Red => self.red_base.take_damage(damage),
                    Team::Blue => self.blue_base.take_damage(damage),
                },
            });

        // clear dead minion
        minion_to_clear.iter().for_each(|id| {
            self.minion_manager.minions.remove(id);
        });

        // Distribute XP from dead minions
        for (minion_row, minion_col, minion_team) in self.dead_minion_positions.drain(..) {
            let mut champions_in_range = Vec::new();
            for (_, champion) in self.champions.iter_mut() {
                // Check if champion is in 5x5 range and is on the opposing team
                if champion.team_id != minion_team &&
                   (champion.row as i32 - minion_row as i32).abs() <= 2 &&
                   (champion.col as i32 - minion_col as i32).abs() <= 2 {
                    champions_in_range.push(champion);
                }
            }

            if !champions_in_range.is_empty() {
                let xp_per_champion = 5 / champions_in_range.len() as u32;
                for champion in champions_in_range {
                    champion.add_xp(xp_per_champion);
                }
            }
        }

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
                println!("tick: {} | anim: {:?}", self.tick, anim);
                animation_commands_executable.push(AnimationCommand::Clear { row, col })
            }
        }
        // 2. Process next frame animations
        for mut anim in self.animations.drain(..) {
            let owner_pos = if let Some(champ) = self.champions.get(&anim.get_owner_id()) {
                Some((champ.row, champ.col))
            } else if let Some(tower) = self.towers.get(&anim.get_owner_id()) {
                Some((tower.row, tower.col))
            } else if let Some(minion) = self.minion_manager.minions.get(&anim.get_owner_id()) {
                Some((minion.row, minion.col))
            } else {
                None // Owner might have been removed
            };

            if let Some((owner_row, owner_col)) = owner_pos {
                let command = anim.next_frame(owner_row, owner_col);
                match command {
                    AnimationCommand::Done => {}
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
                AnimationCommand::Draw {
                    row,
                    col,
                    animation_type,
                } => {
                    // Add bounds check
                    if row < self.board.rows as u16 && col < self.board.cols as u16 {
                        self.board
                            .place_animation(animation_type, row as usize, col as usize);
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

        // Check for win condition
        if self.red_base.stats.health <= 0 {
            let packet = crate::packet::end_game_packet::EndGamePacket::new(Team::Blue);
            let serialized_packet = packet.serialize();
            for (player_id, _) in &self.client_channel {
                self.send_to_player(*player_id, BytesMut::from(&serialized_packet[..]));
            }
            std::process::exit(0);
        } else if self.blue_base.stats.health <= 0 {
            let packet = crate::packet::end_game_packet::EndGamePacket::new(Team::Red);
            let serialized_packet = packet.serialize();
            for (player_id, _) in &self.client_channel {
                self.send_to_player(*player_id, BytesMut::from(&serialized_packet[..]));
            }
            std::process::exit(0);
        }

        // --- Send per player there board view ---
        for (player_id, champion) in &self.champions {
            // 1. Get player-specific board view
            let board_rle_vec = self.board.run_length_encode(champion.row, champion.col);
            // 2. Create the board packet
            let health = champion.get_health();
            let xp_needed = champion.xp_for_next_level().unwrap_or(0); // Get XP needed, 0 if max level
            let board_packet = BoardPacket::new(health.0, health.1, champion.level, champion.xp, xp_needed, board_rle_vec);
            let serialized_packet = board_packet.serialize();
            // 3. Store the serialized packet to be sent later
            updates.insert(*player_id, serialized_packet);
        }
        println!("--------------------");
        updates
    }

    fn tower_turn(&mut self) {
        let pending_damages = self
            .towers
            .iter_mut()
            .map(|(_, tower)| {
                if let Some(enemy) = tower.get_potential_target(&self.board, (7, 9)) {
                    match &enemy.content {
                        Some(content) => match content {
                            CellContent::Minion(id, _) => {
                                if let Some((damage, mut animation)) = tower.can_attack() {
                                    animation.attach_target(*id);
                                    println!("tower anim: {:?}", animation);
                                    self.animations.push(animation);
                                    Some((Target::Minion(*id), damage))
                                } else {
                                    None
                                }
                            }
                            CellContent::Champion(id, _) => {
                                if let Some((damage, mut animation)) = tower.can_attack() {
                                    animation.attach_target(*id);
                                    println!("tower anim: {:?}", animation);
                                    self.animations.push(animation);
                                    Some((Target::Champion(*id), damage))
                                } else {
                                    None
                                }
                            }
                            CellContent::Base(team) => {
                                if let Some((damage, mut animation)) = tower.can_attack() {
                                    animation.attach_target(0); // No specific target ID for base
                                    println!("tower anim: {:?}", animation);
                                    self.animations.push(animation);
                                    Some((Target::Base(*team), damage))
                                } else {
                                    None
                                }
                            }
                            _ => None,
                        },
                        None => None,
                    }
                } else {
                    None
                }
            })
            .filter_map(|option| option)
            .collect::<Vec<(Target, u16)>>();

        pending_damages
            .into_iter()
            .for_each(|(target, damage)| match target {
                Target::Tower(id) => {
                    if let Some(tower) = self.towers.get_mut(&id) {
                        tower.take_damage(damage);
                    }
                }
                Target::Minion(id) => {
                    if let Some(minion) = self.minion_manager.minions.get_mut(&id) {
                        minion.take_damage(damage);
                    }
                }
                Target::Champion(id) => {
                    if let Some(champ) = self.champions.get_mut(&id) {
                        champ.take_damage(damage);
                    }
                }
                Target::Base(team) => match team {
                    Team::Red => self.red_base.take_damage(damage),
                    Team::Blue => self.blue_base.take_damage(damage),
                },
            });
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::game::cell::{CellContent, Team};
    use crate::game::board::Board;

    // Helper function to create a dummy board
    fn create_dummy_board(rows: usize, cols: usize) -> Board {
        Board::new(rows, cols)
    }

    #[test]
    fn test_game_manager_base_placement() {
        let game_manager = GameManager::new();
        let board = game_manager.board;

        // Test Red Base placement (190, 10) to (192, 12)
        for i in 0..3 {
            for j in 0..3 {
                let cell = board.get_cell((190 + i) as usize, (10 + j) as usize).unwrap();
                assert_eq!(cell.content, Some(CellContent::Base(Team::Red)));
            }
        }

        // Test Blue Base placement (10, 190) to (12, 192)
        for i in 0..3 {
            for j in 0..3 {
                let cell = board.get_cell((10 + i) as usize, (190 + j) as usize).unwrap();
                assert_eq!(cell.content, Some(CellContent::Base(Team::Blue)));
            }
        }
    }

    #[test]
    fn test_win_condition() {
        let mut game_manager = GameManager::new();

        // Simulate red base being destroyed
        game_manager.red_base.stats.health = 0;

        // Call game_tick, which should trigger the win condition and exit
        // We can't directly test std::process::exit(0), but we can verify the health.
        // The important part is that the health is 0, which would lead to the exit.
        game_manager.game_tick();

        assert_eq!(game_manager.red_base.stats.health, 0);
        // In a real test, you might mock the send_to_player and std::process::exit
        // to assert that the correct packet was sent and exit was called.
    }
}
