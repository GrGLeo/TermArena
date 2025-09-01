fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_build::configure()
        .build_server(true)
        .compile(&["../../pkg/proto/auth/auth.proto"], &["../../pkg/proto"])?;
    Ok(())
}
