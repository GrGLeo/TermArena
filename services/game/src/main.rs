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
use tokio::time::{Duration, Instant, sleep};
use std::env;
use tracing::{info, warn, error, debug};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

mod config;
mod errors;
mod game;
mod packet;

const TICK_RATE: Duration = Duration::from_millis(40);

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

    #[arg(long = "usernames", value_name = "USERNAMES", value_delimiter = ',')]
    usernames: Vec<String>,

    #[arg(long = "teams", value_name = "TEAMS", value_delimiter = ',', value_parser = clap::value_parser!(u8))]
    teams: Vec<u8>,

    #[arg(long = "spell1s", value_name = "SPELL1S", value_delimiter = ',', value_parser = clap::value_parser!(u8))]
    spell1s: Vec<u8>,

    #[arg(long = "spell2s", value_name = "SPELL2S", value_delimiter = ',', value_parser = clap::value_parser!(u8))]
    spell2s: Vec<u8>,
}

async fn handle_client(stream: TcpStream, addr: SocketAddr, game_manager: Arc<Mutex<GameManager>>) {
    debug!(component = "game", address = %addr, "client handler started");

    let (reader, mut writer) = split(stream);
    let mut buf_reader = BufReader::new(reader);

    // --- Initial Packet: Spell Selection ---
    let mut initial_packet_header = [0; 2]; // Read version and code
    if buf_reader
        .read_exact(&mut initial_packet_header)
        .await
        .is_err()
    {
        error!(component = "game", address = %addr, "failed to read initial packet header");
        if let Err(e) = writer.shutdown().await {
            error!(component = "game", address = %addr, error = %e, "failed to shutdown stream");
        }
        return;
    }

    let version = initial_packet_header[0];
    let code = initial_packet_header[1];

    let (spell1, spell2) = if version == 1 && code == 16 {
        // Code for SpellSelectionPacket
        let mut spell_payload = [0; 2]; // Read spell1 and spell2
        if buf_reader.read_exact(&mut spell_payload).await.is_err() {
            error!(component = "game", address = %addr, "failed to read spell payload");
            if let Err(e) = writer.shutdown().await {
                error!(component = "game", address = %addr, error = %e, "failed to shutdown stream");
            }
            return;
        }
        (spell_payload[0], spell_payload[1])
    } else {
        warn!(component = "game", address = %addr, version = version, code = code, "invalid initial packet");
        if let Err(e) = writer.shutdown().await {
            error!(component = "game", address = %addr, error = %e, "failed to shutdown stream");
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
            info!(component = "game", player_id = id, address = %addr, spell1 = spell1, spell2 = spell2, "player joined");
        } else {
            warn!(component = "game", address = %addr, "connection rejected - server full");
            let rejection_msg = "Server is full. Try again later.\n";
            if let Err(e) = writer.write_all(rejection_msg.as_bytes()).await {
                error!(component = "game", address = %addr, error = %e, "failed to send rejection message");
            }
            if let Err(e) = writer.shutdown().await {
                error!(component = "game", address = %addr, error = %e, "failed to shutdown rejected stream");
            }
            return;
        }
    }

    // -- Split Stream and Spawn Writer Task --
    // Spawn a separate task that owns the 'writer' and listens on 'rx'
    let _ = spawn(async move {
        while let Some(message) = rx.recv().await {
            if writer.write_all(&message).await.is_err() {
                error!(component = "game", player_id = player_id, "failed to write message to client");
                rx.close();
                break;
            }
        }
        debug!(component = "game", player_id = player_id, "writer task ending");
        if let Err(e) = writer.shutdown().await {
            error!(component = "game", player_id = player_id, error = %e, "failed to shutdown writer");
        }
    });

    // -- Verify if we can start game --
    // Scope to release the lock
    {
        let manager = game_manager.lock().await;
        if manager.game_started {
            info!(component = "game", "sending start packet to all clients");
            for player_id in manager.client_channel.keys() {
                let message = StartPacket::new(0).serialize();
                manager.send_to_player(*player_id, message).await;
            }
        }
    }

    // -- Read Client Action loop --
    debug!(component = "game", player_id = player_id, address = %addr, "listening for player actions");
    loop {
        let mut packet_header = [0; 2]; // Read version and code
        if buf_reader.read_exact(&mut packet_header).await.is_err() {
            error!(component = "game", address = %addr, "failed to read packet header");
            break;
        }
        debug!(component = "game", address = %addr, ?packet_header, "packet header received");

        let version = packet_header[0];
        let code = packet_header[1];

        if version != 1 {
            warn!(component = "game", address = %addr, version = version, "invalid packet version");
            break;
        }

        match code {
            11 => {
                // Action Packet
                let mut action_payload = [0; 1];
                if buf_reader.read_exact(&mut action_payload).await.is_err() {
                    error!(component = "game", address = %addr, "failed to read action payload");
                    break;
                }
                let mut manager = game_manager.lock().await;
                manager.store_player_action(player_id, action_payload[0]);
            }
            17 => {
                // Shop Request Packet
                debug!(component = "game", player_id = player_id, "shop request received");
                let manager = game_manager.lock().await;
                if let Some(champion) = manager.get_champion(&player_id) {
                    let message =
                        ShopResponsePacket::new(champion.stats(), champion.get_inventory())
                            .serialize();
                    manager.send_to_player(player_id, message).await;
                } else {
                    warn!(component = "game", player_id = player_id, "champion not found for shop request");
                }
            }
            19 => {
                // Purchase Item Packet
                let mut purchase_payload = [0; 2];
                if buf_reader.read_exact(&mut purchase_payload).await.is_err() {
                    error!(component = "game", address = %addr, "failed to read purchase payload");
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
                                error!(component = "game", player_id = player_id, error = %e, "failed to buy item");
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
                warn!(component = "game", address = %addr, code = code, "invalid packet code");
                break;
            }
        }
    }
    debug!(component = "game", player_id = player_id, address = %addr, "reader loop ended");

    // -- CLeanup --
    {
        let mut manager = game_manager.lock().await;
        manager.remove_player(&player_id);
    }
    debug!(component = "game", player_id = player_id, address = %addr, "handler task cleanup finished");
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize structured logging
    tracing_subscriber::registry()
        .with(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "game=info".into()),
        )
        .with(tracing_subscriber::fmt::layer().json())
        .init();

    // Change to the parent directory (root) to ensure correct relative paths
    let mut current_dir = env::current_dir()?;
    if current_dir.ends_with("bin") {
        current_dir.pop();
        env::set_current_dir(&current_dir)?;
        debug!(component = "game", directory = %current_dir.display(), "working directory changed");
    }

    let args = CliArgs::parse();
    let address = format!("0.0.0:{}", args.port);
    let listener = TcpListener::bind(&address).await?;
    info!(component = "game", address = %address, "game server listening");

    let config = config::GameConfig::load("services/game/stats.toml", "services/game/spells.toml", "services/game/items.toml")
        .expect("Failed to load game configuration");
    let game_manager = GameManager::new(config, args.max_players);
    let arc_gm = Arc::new(Mutex::new(game_manager));
    info!(component = "game", "game manager initialized");

    // -- Game Tick Task --
    let tick_manager = Arc::clone(&arc_gm);
    spawn(async move {
        loop {
            let starting_time = Instant::now();
            let game_started: bool;
            {
                let manager = tick_manager.lock().await;
                game_started = manager.game_started;
            }
            if game_started {
                let start_time = Instant::now();
                // sleep(Duration::from_millis(40)).await;

                let updates: HashMap<PlayerId, ClientMessage>;
                {
                    let mut manager = tick_manager.lock().await;
                    updates = manager.game_tick();
                    manager.clear_action();
                }
                let manager = tick_manager.lock().await;
                for (player_id, message) in updates {
                    debug!(component = "game", player_id = player_id, message_len = message.len(), "sending game update");
                    manager.send_to_player(player_id, message).await;
                }
                drop(manager);
                if let Some(duration) = TICK_RATE.checked_sub(start_time.elapsed()) {
                    sleep(duration).await
                }
                debug!(component = "game", tick_time = ?starting_time.elapsed(), "game tick completed");
            } else {
                sleep(Duration::from_secs(5)).await;
                debug!(component = "game", "waiting for players to connect");
            }
        }
    });

    // -- Accept Connections Loop --
    loop {
        match listener.accept().await {
            Ok((stream, addr)) => {
                info!(component = "game", address = %addr, "connection accepted");
                let game_manager_for_task = Arc::clone(&arc_gm);
                spawn(async move {
                    handle_client(stream, addr, game_manager_for_task).await;
                });
            }
            Err(e) => {
                error!(component = "game", error = %e, "failed to accept connection");
                let _ = sleep(Duration::from_secs(1));
            }
        }
    }
}
