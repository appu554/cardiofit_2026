use std::fs;

fn main() {
    let content = fs::read_to_string("knowledge/kb_drug_rules/warfarin.toml").unwrap();
    
    // Check for any # characters
    for (line_num, line) in content.lines().enumerate() {
        if line.contains('#') {
            println!("Found # character at line {}: {}", line_num + 1, line);
        }
    }
    
    // Try to parse as TOML
    match toml::from_str::<toml::Value>(&content) {
        Ok(_) => println!("TOML parsing successful!"),
        Err(e) => println!("TOML parsing error: {}", e),
    }
}
