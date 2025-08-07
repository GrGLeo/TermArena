use crate::game::{ClientMessage, GameManager, PlayerId};
use clap::Parser;
use packet::shop_packet::{PurchaseItemPacket, ShopResponsePacket};
use packet::start_packet::StartPacket;
use std::collections::HashMap;
use std::net::SocketAddr;
use std::sync::Arc;
use tokio::io::{AsyncReadExt, AsyncWriteExt, BufReader, split};
use tokio::net::{TcpListener, TcpStream};
use tokio::spawn;
use tokio::sync::Mutex;
use tokio::sync::mpsc;
use tokio::time::{Duration, sleep};

mod config;
mod errors;
mod game;
mod packet;

// Cli Parser
#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct CliArgs {
    #[arg(long = "port", value_name = "PORT", value_parser = clap::value_parser!(u16))]
    port: u16,

    #[arg(long = "map", value_name = "MAP_ID", value_parser = clap::value_parser!(u8))]
    map_id: Option<u8>,

    #[arg(long = "max-players", value_name = "MAX_PLAYERS", value_parser = clap::value_parser!(u8), default_value_t = 1)]
    max_players: u8,
}

async fn handle_client(stream: TcpStream, addr: SocketAddr, game_manager: Arc<Mutex<GameManager>>) {
    println!("Handler task started for connection from: {:?}", addr);

    let (reader, mut writer) = split(stream);
    let mut buf_reader = BufReader::new(reader);

    // --- Initial Packet: Spell Selection ---
    let mut initial_packet_header = [0; 2]; // Read version and code
    if buf_reader
        .read_exact(&mut initial_packet_header)
        .await
        .is_err()
    {
        eprintln!("Error reading initial packet header from {:?}", addr);
        if let Err(e) = writer.shutdown().await {
            eprintln!("Error shutting down stream for {:?}: {}", addr, e);
        }
        return;
    }

    let version = initial_packet_header[0];
    let code = initial_packet_header[1];

    let (spell1, spell2) = if version == 1 && code == 13 {
        // Code for SpellSelectionPacket
        let mut spell_payload = [0; 2]; // Read spell1 and spell2
        if buf_reader.read_exact(&mut spell_payload).await.is_err() {
            eprintln!("Error reading spell payload from {:?}", addr);
            if let Err(e) = writer.shutdown().await {
                eprintln!("Error shutting down stream for {:?}: {}", addr, e);
            }
            return;
        }
        (spell_payload[0], spell_payload[1])
    } else {
        eprintln!(
            "Invalid initial packet from {:?}: Version={}, Code={}",
            addr, version, code
        );
        if let Err(e) = writer.shutdown().await {
            eprintln!("Error shutting down stream for {:?}: {}", addr, e);
        }
        return;
    };

    let player_id: PlayerId;
    let (tx, mut rx) = mpsc::channel::<ClientMessage>(32);

    {
        let mut manager = game_manager.lock().await;
        if let Some(id) = manager.add_player(spell1, spell2) {
            player_id = id;
            manager.client_channel.insert(id, tx);
            println!(
                "Player {} ({:?}) joined with spells {} and {}",
                id, addr, spell1, spell2
            );
        } else {
            println!("Rejecting connection from {:?}: Server is full.", addr);
            let rejection_msg = "Server is full. Try again later.\n";
            if let Err(e) = writer.write_all(rejection_msg.as_bytes()).await {
                eprintln!("Error sending rejection message to {:?}: {}", addr, e);
            }
            if let Err(e) = writer.shutdown().await {
                eprintln!("Error shutting down rejected stream for {:?}: {}", addr, e);
            }
            return;
        }
    }

    // -- Split Stream and Spawn Writer Task --
    // The reader and writer are already split from the initial read
    // Spawn a separate task that owns the 'writer' and listens on 'rx'
    let _ = spawn(async move {
        while let Some(message) = rx.recv().await {
            if writer.write_all(&message).await.is_err() {
                eprintln!(
                    "Error writting message to client {}, connection likely closed.",
                    player_id
                );
                rx.close();
                break;
            }
        }
        println!("Writer task for player {} ending.", player_id);
        if let Err(e) = writer.shutdown().await {
            eprintln!("Error shutting down writer for player {}: {}", player_id, e);
        }
    });

    // -- Verify if we can start game --
    // Scope to release the lock
    {
        let manager = game_manager.lock().await;
        if manager.game_started {
            println!("Sending StartPacket to all client");
            for player_id in manager.client_channel.keys() {
                let message = StartPacket::new(0).serialize();
                manager.send_to_player(*player_id, message).await;
            }
        }
    }

    // -- Read Client Action loop --
    println!("Listening for Player {} ({:?}) actions...", player_id, addr);
    loop {
        let mut packet_header = [0; 2]; // Read version and code
        if buf_reader.read_exact(&mut packet_header).await.is_err() {
            eprintln!("Error reading packet header from {:?}", addr);
            break;
        }
        println!("Packet header: {:?}", packet_header);

        let version = packet_header[0];
        let code = packet_header[1];

        if version != 1 {
            eprintln!("Invalid packet version from {:?}: {}", addr, version);
            break;
        }

        match code {
            8 => {
                // Action Packet
                let mut action_payload = [0; 1];
                if buf_reader.read_exact(&mut action_payload).await.is_err() {
                    eprintln!("Error reading action payload from {:?}", addr);
                    break;
                }
                let mut manager = game_manager.lock().await;
                manager.store_player_action(player_id, action_payload[0]);
            }
            14 => {
                // Shop Request Packet
                println!("Got a requests shop packet");
                let manager = game_manager.lock().await;
                if let Some(champion) = manager.get_champion(&player_id) {
                    let message =
                        ShopResponsePacket::new(champion.stats(), champion.get_inventory())
                            .serialize();
                    manager.send_to_player(player_id, message).await;
                } else {
                    println!("Player: {} champion not found", player_id);
                }
            }
            16 => {
                // Purchase Item Packet
                let mut purchase_payload = [0; 2];
                if buf_reader.read_exact(&mut purchase_payload).await.is_err() {
                    eprintln!("Error reading purchase payload from {:?}", addr);
                    break;
                }
                if let Ok(packet) = PurchaseItemPacket::deserialize(&purchase_payload) {
                    let mut manager = game_manager.lock().await;
                    if let Some(item) = manager
                        .get_config()
                        .items
                        .get(&packet.item_id.into())
                        .cloned()
                    {
                        if let Some(champion) = manager.get_mut_champion(&player_id) {
                            if let Err(e) = champion.add_item(item) {
                                eprintln!("Player {} failed to buy item: {}", player_id, e);
                            } else {
                                // Send back the updated champion stats
                                let message = ShopResponsePacket::new(
                                    champion.stats(),
                                    champion.get_inventory(),
                                )
                                .serialize();
                                manager.send_to_player(player_id, message).await;
                            }
                        }
                    }
                }
            }
            _ => {
                eprintln!("Invalid packet code from {:?}: {}", addr, code);
                break;
            }
        }
    }
    println!("Reader loop for player {} ({:?}) ended.", player_id, addr);

    // -- CLeanup --
    {
        let mut manager = game_manager.lock().await;
        manager.remove_player(&player_id);
    }
    println!(
        "Handler task for player {} ({:?}) finished cleanup.",
        player_id, addr
    );
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let args = CliArgs::parse();
    let address = format!("0.0.0.0:{}", args.port);
    let listener = TcpListener::bind(&address).await?;
    println!("Server listening  on {}", address);

    let config = config::GameConfig::load("game/stats.toml", "game/spells.toml", "game/items.toml")
        .expect("Failed to load game configuration");
    let game_manager = GameManager::new(config, args.max_players);
    let arc_gm = Arc::new(Mutex::new(game_manager));
    println!("GameManager created and wrapped.");

    // -- Game Tick Task --
    let tick_manager = Arc::clone(&arc_gm);
    spawn(async move {
        loop {
            let game_started: bool;
            {
                let manager = tick_manager.lock().await;
                game_started = manager.game_started;
            }
            if game_started {
                sleep(Duration::from_millis(40)).await;

                let updates: HashMap<PlayerId, ClientMessage>;
                {
                    let mut manager = tick_manager.lock().await;
                    updates = manager.game_tick();
                    manager.clear_action();
                }
                let manager = tick_manager.lock().await;
                for (player_id, message) in updates {
                    println!("Message length to be sent: {:?}", message.len());
                    manager.send_to_player(player_id, message).await;
                }
                drop(manager);
            } else {
                sleep(Duration::from_secs(5)).await;
                println!("Waiting for all players to connect...");
            }
        }
    });

    // -- Accept Connections Loop --
    loop {
        match listener.accept().await {
            Ok((stream, addr)) => {
                println!("Accepted connection form {:?}", addr);
                let game_manager_for_task = Arc::clone(&arc_gm);
                spawn(async move {
                    handle_client(stream, addr, game_manager_for_task).await;
                });
            }
            Err(e) => {
                eprintln!("Error accepting connection: {}", e);
                let _ = sleep(Duration::from_secs(1));
            }
        }
    }
}
