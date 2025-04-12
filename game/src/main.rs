mod cell;
mod entities;
mod board;
mod manager;
mod config;
use clap::Parser;
use manager::game::GameManager;
use std::io::{self, Read, Write};
use std::fs::File;
use std::error::Error;
use tokio::net::TcpListener;


fn log_to_file(message: &str) -> Result<(), Box<dyn Error>> {
    let mut file = File::options()
        .create(true)
        .append(true)
        .open("game.log")?;
    writeln!(file, "{}", message)?;
    Ok(())
}

#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct CliArgs {
    #[arg(long = "port", value_name = "PORT", value_parser = clap::value_parser!(u16))]
    port: u16,

    #[arg(long = "map", value_name = "MAP_ID", value_parser = clap::value_parser!(u8))]
    map_id: u8,
}

 
#[tokio::main]
async fn main() -> Result<(), std::io::Error> {
    let args = CliArgs::parse();
    println!("Starting server...");
    let game = GameManager::new();
    let listener = TcpListener::bind(format!("127.1.0.0:{}", args.port)).await.unwrap();
    let mut incoming = listener.incoming();
    while let Some(stream) = incoming.next() {
        match stream {
            Ok(stream) => {
            todo!()
            }
            Err(e) => {
                log_to_file("Failed to read stream");
            }
        }
    }
    Ok(())

}
