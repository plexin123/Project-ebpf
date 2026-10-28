fn main() -> Result<(), Box<dyn std::error::Error>> {
    // solo el cliente: el servidor gRPC vive en el backend de Go
    tonic_prost_build::configure()
        .build_server(false)
        .compile_protos(&["../proto/profiler.proto"], &["../proto"])?;
    Ok(())
}
