use std::env;

fn main() {
    let crate_dir = env::var("CARGO_MANIFEST_DIR").unwrap();

    cbindgen::Builder::new()
        .with_crate(crate_dir)
        .with_language(cbindgen::Language::C)
        .with_pragma_once(true)
        .with_include_guard("SAFETY_ENGINES_H")
        .with_no_includes()
        .with_cpp_compat(true)
        .with_documentation(true)
        .generate()
        .expect("Unable to generate C bindings")
        .write_to_file("target/cae_engine.h");

    println!("cargo:rerun-if-changed=src/");
    println!("cargo:rerun-if-changed=build.rs");
}