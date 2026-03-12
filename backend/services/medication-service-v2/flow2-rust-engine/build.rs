//! Build script for Flow2 Rust Engine
//! Compiles Protocol Buffers into Rust code

fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Compile Flow2 protobuf definition
    tonic_build::configure()
        .build_server(true)
        .build_client(true)
        .out_dir("src/grpc")
        .compile(
            &["proto/flow2.proto"],
            &["proto"],
        )?;

    // Tell cargo to recompile if protobuf files change
    println!("cargo:rerun-if-changed=proto/flow2.proto");
    println!("cargo:rerun-if-changed=build.rs");

    Ok(())
}