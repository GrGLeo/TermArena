use crate::game::{ClientMessage, GameManager, PlayerId};
use clap::Parser;
use packet::action_packet::ActionPacket;
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
}

async fn handle_client(stream: TcpStream, addr: SocketAddr, game_manager: Arc<Mutex<GameManager>>) {
    println!("Handler task started for connection from: {:?}", addr);
    let player_id_option: Option<PlayerId>;

    // Create the player channel
    let (tx, mut rx) = mpsc::channel::<ClientMessage>(32);
    {
        let mut manager = game_manager.lock().await;
        if let Some(id) = manager.add_player() {
            player_id_option = Some(id);
            manager.client_channel.insert(id, tx);
            println!("Player {} ({:?}) joined", id, addr);
        } else {
            println!("Rejecting connection form {:?}: Server is full.", addr);
            let mut stream = stream;
            let rejection_msg = "Server is full. Try again later.\n";
            if let Err(e) = stream.write_all(rejection_msg.as_bytes()).await {
                eprintln!("Error sending rejection message to {:?}: {}", addr, e);
            }
            if let Err(e) = stream.shutdown().await {
                eprintln!("Error shutting down rejected stream for {:?}: {}", addr, e);
            }
            return;
        }
    }

    let player_id = match player_id_option {
        Some(id) => id,
        None => {
            eprintln!(
                "Error failed to get player ID for {:?} after lock release.",
                addr
            );
            let mut stream = stream;
            if let Err(e) = stream.shutdown().await {
                eprintln!(
                    "Error shutting down  stream for {:?} after ID error: {}",
                    addr, e
                );
            }
            return;
        }
    };

    // -- Split Stream and Spawn Writer Task --
    let (reader, mut writer) = split(stream);
    let mut buf_reader = BufReader::new(reader);
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
        let mut packet_buffer = [0; 3];
        match buf_reader.read_exact(&mut packet_buffer).await {
            Ok(3) => match ActionPacket::deserialize(&packet_buffer) {
                Ok(packet) => {
                    if packet.version == 1 && packet.code == 8 {
                        let mut manager = game_manager.lock().await;
                        manager.store_player_action(player_id, packet.action);
                        println!("Received action from: {} ({:?})", player_id, addr);
                        drop(manager);
                    } else {
                        eprintln!(
                            "Player {} send packet with invalid version/code: V={}, C={}",
                            player_id, packet.version, packet.code
                        );
                    }
                }
                Err(e) => {
                    eprintln!("Player {} sent invalid packet format: {}", player_id, e);
                }
            },
            Ok(0) => {
                // Connection closed by client
                println!(
                    "Player {} ({:?}) disconnected (read 0 bytes)",
                    player_id, addr
                );
                break;
            }
            Ok(_n) => {
                println!(
                    "Incomplete read for player {} ({:?}), likely disconnected",
                    player_id, addr
                );
                break;
            }
            Err(e) => {
                println!(
                    "Error reading action for player {} ({:?}): {}. Disconnecting.",
                    player_id, addr, e
                );
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

    let config =
        config::GameConfig::load("game/stats.toml").expect("Failed to load game configuration");
    let game_manager = GameManager::new(config);
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
