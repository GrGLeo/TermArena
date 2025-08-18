use std::sync::Arc;
use tokio::sync::Mutex;
use tonic::{Request, Response, Status, transport::Server};

pub mod auth {
    tonic::include_proto!("auth");
}
use auth::auth_service_server::{AuthService, AuthServiceServer};
use auth::{
    AuthentificateRequest, AuthentificateResponse, GetLoginChallengeRequest,
    GetLoginChallengeResponse, RegisterRequest, RegisterResponse,
};

use tokio_rusqlite::Connection;

use rand::RngCore;
use rsa::{Pkcs1v15Sign, RsaPublicKey, pkcs8::DecodePublicKey};
use sha2::{Digest, Sha256};

#[derive(Debug)]
pub struct TermArenaAuthService {
    db: Arc<Mutex<Connection>>,
}

#[tonic::async_trait]
impl AuthService for TermArenaAuthService {
    async fn register(
        &self,
        request: Request<RegisterRequest>,
    ) -> Result<Response<RegisterResponse>, Status> {
        let req = request.into_inner();
        let username_for_log = req.username.clone(); // Clone for logging
        println!("Registering user: {}", username_for_log);

        let conn = self.db.lock().await;
        let username = req.username.clone();
        let res = conn
            .call(move |conn| {
                conn.execute(
                    "INSERT INTO users (username, public_key) VALUES (?1, ?2)",
                    &[
                        &username as &dyn rusqlite::ToSql,
                        &req.public_key as &dyn rusqlite::ToSql,
                    ],
                )
                .map_err(|e| e.into())
            })
            .await;

        match res {
            Ok(_) => {
                // Username registered, now we need to generate a challenge
                println!("Starting challenge creation for user: {}", username_for_log);
                let mut challenge = [0u8; 32];
                rand::thread_rng().fill_bytes(&mut challenge);

                let res = conn
                    .call({
                        let username = req.username.clone();
                        let challenge_vec = challenge.to_vec();
                        move |conn| {
                            conn.execute(
                                "INSERT INTO challenges (username, challenge, expires_at) VALUES (?1, ?2, strftime('%s','now') + 300)",
                                &[&username as &dyn rusqlite::ToSql, &challenge_vec as &dyn rusqlite::ToSql],
                            ).map_err(|e| e.into())
                        }
                    })
                    .await;
                println!("Challenge stored for user: {}", username_for_log);

                if let Err(e) = res {
                    eprintln!("Failed to store challenge for {}: {}", req.username, e);
                    return Err(Status::internal("Failed to prepare login challenge"));
                }

                println!("Challenge created for user: {}", username_for_log);

                Ok(Response::new(RegisterResponse {
                    success: true,
                    message: "User registered".into(),
                    challenge: challenge.to_vec(),
                }))
            }
            Err(e) => {
                println!("Failed to register user {}: {}", username_for_log, e);
                Ok(Response::new(RegisterResponse {
                    success: false,
                    message: "Username may already be taken.".into(),
                    challenge: vec![],
                }))
            }
        }
    }

    async fn get_login_challenge(
        &self,
        request: Request<GetLoginChallengeRequest>,
    ) -> Result<Response<GetLoginChallengeResponse>, Status> {
        let req = request.into_inner();
        println!("Generating challenge for user: {}", req.username);

        let mut challenge = [0u8; 32];
        rand::thread_rng().fill_bytes(&mut challenge);

        let conn = self.db.lock().await;
        let res = conn
            .call({
                let username = req.username.clone();
                let challenge_vec = challenge.to_vec();
                move |conn| {
                    conn.execute(
                        "INSERT INTO challenges (username, challenge, expires_at) VALUES (?1, ?2, strftime('%s','now') + 300)",
                        &[&username as &dyn rusqlite::ToSql, &challenge_vec as &dyn rusqlite::ToSql],
                    ).map_err(|e| e.into())
                }
            })
            .await;

        if let Err(e) = res {
            eprintln!("Failed to store challenge for {}: {}", req.username, e);
            return Err(Status::internal("Failed to prepare login challenge"));
        }

        Ok(Response::new(GetLoginChallengeResponse {
            challenge: challenge.to_vec(),
        }))
    }

    async fn authentificate(
        &self,
        request: Request<AuthentificateRequest>,
    ) -> Result<Response<AuthentificateResponse>, Status> {
        let req = request.into_inner();
        let username_for_log = req.username.clone(); // Clone for logging
        println!("Authenticating user: {}", username_for_log);

        let db = self.db.lock().await;

        let maybe_data: Result<(Vec<u8>, Vec<u8>), _> = db.call({
            let username = req.username.clone();
            move |conn| {
                conn.query_row(
                    "SELECT u.public_key, c.challenge FROM users u JOIN challenges c ON u.username = c.username
                     WHERE u.username = ?1 AND c.expires_at > strftime('%s','now')
                     ORDER BY c.id DESC LIMIT 1",
                    [&username],
                    |row| Ok((row.get(0)?, row.get(1)?)),
                ).map_err(|e| e.into())
            }
        }).await;

        let _ = db
            .call({
                let username = req.username.clone();
                move |conn| {
                    conn.execute("DELETE FROM challenges WHERE username = ?1", [&username])
                        .map_err(|e| e.into())
                }
            })
            .await;

        let (public_key_der, original_challenge) = match maybe_data {
            Ok(data) => data,
            Err(_) => {
                return Ok(Response::new(AuthentificateResponse {
                    success: false,
                    message: "Invalid username or challenge expired.".into(),
                }));
            }
        };

        let public_key = match RsaPublicKey::from_public_key_der(&public_key_der) {
            Ok(key) => key,
            Err(_) => return Err(Status::internal("Failed to parse public key")),
        };

        let mut hasher = Sha256::new();
        hasher.update(&original_challenge);
        let hashed_challenge = hasher.finalize();

        if public_key
            .verify(
                Pkcs1v15Sign::new::<Sha256>(),
                &hashed_challenge,
                &req.signed_challenge,
            )
            .is_ok()
        {
            println!("Successfully authenticated user: {}", username_for_log);
            Ok(Response::new(AuthentificateResponse {
                success: true,
                message: "Authentication successful".into(),
            }))
        } else {
            println!(
                "Authentication failed (invalid signature) for user: {}",
                username_for_log
            );
            Ok(Response::new(AuthentificateResponse {
                success: false,
                message: "Invalid signature.".into(),
            }))
        }
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let addr = "0.0.0.0:50051".parse()?;

    let conn = Connection::open("auth.db").await?;
    let db = Arc::new(Mutex::new(conn));

    db.lock()
        .await
        .call(|conn| {
            conn.execute_batch(
                "CREATE TABLE IF NOT EXISTS users (
                id INTEGER PRIMARY KEY,
                username TEXT NOT NULL UNIQUE,
                public_key BLOB NOT NULL
            );
            CREATE TABLE IF NOT EXISTS challenges (
                id INTEGER PRIMARY KEY,
                username TEXT NOT NULL,
                challenge BLOB NOT NULL,
                expires_at INTEGER NOT NULL
            );",
            )
            .map_err(|e| e.into())
        })
        .await?;

    println!("Database is ready.");

    let auth_service = TermArenaAuthService { db };

    println!("AuthService listening on {}", addr);

    Server::builder()
        .add_service(AuthServiceServer::new(auth_service))
        .serve(addr)
        .await?;

    Ok(())
}
