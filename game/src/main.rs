mod cell;
mod entities;
mod board;
mod config;
use clap::Parser;
use std::io::{self, Read, Write};
use std::net::{TcpListener, TcpStream};
use std::thread;
use std::fs::File;
use std::error::Error;


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

fn handle_client(mut stream: TcpStream) -> io::Result<()> {
    if let Err(e) = log_to_file("client connected") {
        println!("Failed to write log: {}", e);
    }
    let mut buffer = [0; 1024];

    loop {
        let bytes_read = stream.read(&mut buffer)?;
        if bytes_read == 0 {
            println!("Client disconnected");
            break;
        }
        let data = &buffer[0..bytes_read];
        println!("Data received {:?}", data);
        stream.flush()?;
    }
    Ok(())
}
 
fn main() -> io::Result<()> {

    let args = CliArgs::parse();
    println!("Starting server...");
    let listener = TcpListener::bind(format!("127.0.0.1:{}", args.port))?;
    println!("Port: {}", args.port);
    for stream in listener.incoming() {
        match stream {
            Ok(stream) => {
                thread::spawn(move || {
                    if let Err(e) = handle_client(stream) {
                        eprintln!("Error accepting new client {}", e);
                    }
                });
            }
            Err(e) => {
                eprintln!("Error accepting new connection {}", e);
            }
        }
    }
    Ok(())
}
