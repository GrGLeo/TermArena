mod cell;
mod entities;
mod board;
mod config;
use clap::Parser;


#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct CliArgs {
    #[arg(long = "port", value_name = "PORT", value_parser = clap::value_parser!(u16))]
    port: u16,

    #[arg(long = "map", value_name = "MAP_ID", value_parser = clap::value_parser!(u8))]
    map_id: u8,
}
 
fn main() {
    let args = CliArgs::parse();

    println!("Starting server...");
    println!("Port: {}", args.port);
    println!("Map_id: {}", args.map_id);
}
