use tonic::{transport::Server, Request, Response, Status};
use auth::login_service_server::{LoginService, LoginServiceServer};
use auth::create_service_server::{CreateService, CreateServiceServer};
use auth::{AuthentificationRequest, AuthentificationResponse, SigninRequest, SigninResponse};

pub mod auth {
    tonic::include_proto!("auth");
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
