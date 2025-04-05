use std::error::Error;
use tonic::{transport::Server, Request, Response, Status};
use serde::{Serialize, Deserialize};
use tokio::fs::{File, OpenOptions};
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader, BufWriter};
use std::path::Path;
use std::sync::Arc;
use tokio::sync::Mutex;
use auth::login_service_server::{LoginService, LoginServiceServer};
use auth::create_service_server::{CreateService, CreateServiceServer};
use auth::{AuthentificationRequest, AuthentificationResponse, SigninRequest, SigninResponse};
use argon2::{
    password_hash::{
        rand_core::OsRng, PasswordHash, PasswordHasher, PasswordVerifier, SaltString
    },
    Argon2
};

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
        File::create(path).await?;
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
    println!("Loaded {} users from {}", users.len(), path.display());
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

fn password_hash(password: &str) -> Result<String, argon2::password_hash::Error> {
    let salt = SaltString::generate(&mut OsRng);
    let argon2 = Argon2::default();
    let password_hash = argon2.hash_password(password.as_bytes(), &salt)?.to_string();
    Ok(password_hash)
}

fn verify_password(hashed_password: &str, password: &str) -> Result<bool, argon2::password_hash::Error> {
    let parsed_hash = PasswordHash::new(hashed_password)?;
    Ok(Argon2::default().verify_password(password.as_bytes(), &parsed_hash).is_ok())
}

type SharedUsers = Arc<Mutex<Vec<UserRecord>>>;

#[derive(Debug)]
pub struct MyLoginService {
    users: SharedUsers
}

#[tonic::async_trait]
impl LoginService for MyLoginService {
    async fn authentificate(
        &self, 
        request: Request<AuthentificationRequest>,
    ) -> Result<Response<AuthentificationResponse>, Status> {
        let req_data = request.into_inner();
        println!("Received login request from {}", req_data.username);
        let user_guard = self.users.lock().await;
        let user_record = user_guard.iter().find(|u| u.username == req_data.username);
        match user_record {
            Some(user) => {
                match verify_password(&user.password_hash, &req_data.password) {
                    Ok(true) => {
                        println!("Authentification successfull for user: {}", user.username);
                        let reply = AuthentificationResponse{
                            success: true,
                            user_id: user.username.clone(),
                            message: "Authentification successfull".to_string(),
                        };
                        Ok(Response::new(reply))
                    }
                    Ok(false) => {
                        println!("Authentification failed, wrong password for user: {}", user.username);
                        let reply = AuthentificationResponse{
                            success: false,
                            user_id: user.username.clone(),
                            message: "Authentification failed".to_string(),
                        };
                        Ok(Response::new(reply))
                    }
                    Err(e) => {
                        eprintln!("Password verification failed for user {}: {}", user.username, e);
                        Err(Status::internal("Password verification failed"))
                    }
                }
            }
            None => {
                println!("Authentification failed (user not found): {}", req_data.username);
                let reply = AuthentificationResponse{
                    success: false,
                    user_id: req_data.username.clone(),
                    message: "Authentification failed: user not found".to_string(),
                };
                Ok(Response::new(reply))
            }
        }
    }
}

#[derive(Debug)]
pub struct MyCreateService {
    users: SharedUsers,
    user_file_path: Arc<Path>,
}

#[tonic::async_trait]
impl CreateService for MyCreateService {
    async fn signin(
        &self,
        request: Request<SigninRequest>,
        ) -> Result<Response<SigninResponse>, Status> {
        let req_data = request.into_inner();
        println!("Received signin request for user: {}", req_data.username);

        if req_data.username.trim().is_empty() || req_data.password.trim().is_empty() {
            return Err(Status::invalid_argument("Username and password cannot be empty"))
        }
        let mut users_guard = self.users.lock().await;

        if users_guard.iter().any(|u| u.username == req_data.username) {
            println!("Signin failed: Username '{}' already exist", req_data.username);
            return Err(Status::already_exists(format!("Username already exist {}", req_data.username)));
        }

        let password_hash = match password_hash(&req_data.password) {
            Ok(hash) => hash,
            Err(e) => {
                eprintln!("Failed to hash password for {}: {}", req_data.username, e);
                return Err(Status::internal("Failed to process password."))
            }
        };
        let new_user = UserRecord{
            username: req_data.username.clone(),
            password_hash,
        };
        if let Err(e) = append_user(&self.user_file_path, &new_user).await {
            eprintln!("Failed to append user {} to file: {}", req_data.username, e);
            return Err(Status::internal("Failed to saved user data"))
        }
        users_guard.push(new_user.clone());

        println!("Successfully created user: {}", req_data.username);
        let reply = SigninResponse{
            success: true,
            message: format!("Username '{}' created", req_data.username),
        };
        Ok(Response::new(reply))
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let addr = match "0.0.0.0:50051".parse() {
        Ok(v) => v,
        Err(e) => return Err(Box::new(e) as Box<dyn Error>),
    };
    let user_file = Path::new(USER_DATA_FILE).to_path_buf();

    let initial_users = match load_users(&user_file).await {
        Ok(users) => users,
        Err(e) => {
            eprintln!("FATAL: could not load or create file {}: {}", user_file.display(), e);
            return Err(Box::new(e))
        }
    };

    let shared_users = Arc::new(Mutex::new(initial_users));
    let user_file_path_arc = Arc::from(user_file);

    let login_service = MyLoginService{
        users: Arc::clone(&shared_users),
    };
    let create_service = MyCreateService{
        users: Arc::clone(&shared_users),
        user_file_path: Arc::clone(&user_file_path_arc),
    };
    println!("Server listening on {}", addr);


    Server::builder()
        .add_service(LoginServiceServer::new(login_service))
        .add_service(CreateServiceServer::new(create_service))
        .serve(addr)
        .await?;

    Ok(())
}
