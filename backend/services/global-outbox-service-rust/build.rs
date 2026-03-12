fn main() -> Result<(), Box<dyn std::error::Error>> {
    tonic_build::configure()
        .build_server(true)
        .build_client(false)
        .out_dir("src/proto")
        .compile(&["proto/outbox.proto"], &["proto"])?;
    
    println!("cargo:rerun-if-changed=proto/outbox.proto");
    Ok(())
}