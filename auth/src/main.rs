use prost::bytes::Buf;
use tonic::{transport::Server, Request, Response, Status};
use serde::{Serialize, Deserialize};
use tokio::fs::{File, OpenOptions};
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader, BufWriter};
use std::io::BufWriter;
use std::path::Path;
use std::sync::Arc;
use tokio::sync::Mutex;
use auth::login_service_server::{LoginService, LoginServiceServer};
use auth::create_service_server::{CreateService, CreateServiceServer};
use auth::{AuthentificationRequest, AuthentificationResponse, SigninRequest, SigninResponse};

pub mod auth {
    tonic::include_proto!("auth");
}

#[derive(Debug, Deserialize, Serialize, Clone)]
struct UserRecord {
    username: String,
    password_hash: String,
}

const USER_DATA_FILE: &str = "users.jsonl";

async fn load_users(path: &Path) -> Result<Vec<UserRecord>, std::io::Error> {
    if !path.exists() {
        return Ok(Vec::new());
    }
    let file = File::open(path).await?;
    let reader = BufReader::new(file);
    let mut lines = reader.lines();
    let mut users = Vec::new();

    while let Some(line) = lines.next_line().await? {
        if line.trim().is_empty() {
            continue;
        }
        match serde_json::from_str::<UserRecord>(&line) {
            Ok(user) => users.push(user),
            Err(e) => {
                eprintln!("Warning: Failed to parse line: {}. Error: {}", line, e);
            }
        }
    }
    Ok(users)
}

async fn append_user(path: &Path, user: &UserRecord) -> Result<(), std::io::Error> {
    let file = OpenOptions::new()
        .create(true)
        .append(true)
        .open(path)
        .await?;

    let mut writer = BufWriter::new(file);
    let json_line = serde_json::to_string(user)? + "\n";
    writer.write_all(json_line.as_bytes()).await?;
    writer.flush().await?;
    Ok(())
}


#[derive(Debug, Default)]
pub struct MyLoginService {}


#[tonic::async_trait]
impl LoginService for MyLoginService {
    async fn authentificate(
        &self, 
        request: Request<AuthentificationRequest>,
    ) -> Result<Response<AuthentificationResponse>, Status> {
        println!("Receive login request");
        let reply = AuthentificationResponse{
            success: true,
            user_id: "user123".to_string(),
            message: "Authentification successfull".to_string(),
        };
        println!("Sending login response");
        Ok(Response::new(reply))
    }
}

#[derive(Debug, Default)]
pub struct MyCreateService {}

#[tonic::async_trait]
impl CreateService for MyCreateService {
    async fn signin(
        &self,
        request: Request<SigninRequest>,
        ) -> Result<Response<SigninResponse>, Status> {
        let reply = SigninResponse {
            success: true,
            message: "Sign-in successfull".to_string(),
        };
        Ok(Response::new(reply))
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let addr = "0.0.0.0:50051".parse()?;
    let login_service = MyLoginService::default();
    let create_service = MyCreateService::default();

    println!("Server listening on {}", addr);

    Server::builder()
        .add_service(LoginServiceServer::new(login_service))
        .add_service(CreateServiceServer::new(create_service))
        .serve(addr)
        .await?;

    Ok(())
}
