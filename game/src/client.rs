// client.rs
use rand::Rng;
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
use tokio::net::TcpStream;
use tokio::spawn;
use tokio::time::{Duration, sleep};

#[derive(Debug)]
struct ActionPacket {
    version: u8,
    code: u8,
    action: u8,
}

impl ActionPacket {
    pub fn serialize(&self) -> [u8; 3] {
        [self.version, self.code, self.action]
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let server_address = "127.0.0.1:8080";
    let stream = TcpStream::connect(server_address).await?;
    println!("Connected to server: {}", server_address);

    let (reader, mut writer) = tokio::io::split(stream);
    let mut buf_reader = BufReader::new(reader);

    spawn(async move {
        let mut response_buffer = String::new();
        loop {
            response_buffer.clear();
            match buf_reader.read_line(&mut response_buffer).await {
                Ok(0) => {
                    println!("Server disconnected.");
                    break;
                }
                Ok(_) => {
                    println!("Received from server: {}", response_buffer.trim());
                }
                Err(e) => {
                    eprintln!("Error reading from server: {}", e);
                    break;
                }
            }
        }
    });

    let mut rng = rand::thread_rng();
    let mut message_counter = 1;
    loop {
        // Generate a random action value between 1 and 5
        let action_value: u8 = rng.gen_range(1..=5);

        // Create a new ActionPacket
        let packet = ActionPacket {
            version: 1,
            code: 8,
            action: action_value,
        };

        // Serialize the packet into a byte array
        let serialized_packet = packet.serialize();

        // Send the serialized packet to the server
        if let Err(e) = writer.write_all(&serialized_packet).await {
            eprintln!("Error sending action packet: {}", e);
            break;
        }
        println!("Client sent action packet: {:?}", packet);

        message_counter += 1;
        sleep(Duration::from_secs(1)).await;
    }

    println!("Client finished sending actions.");
    Ok(())
}
