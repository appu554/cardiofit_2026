use std::fs;
use flow2_rust_engine::unified_clinical_engine::knowledge_base::KnowledgeBase;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("Testing TOML file parsing...");
    
    // Path to the knowledge base directory
    let kb_path = "knowledge/kb_drug_rules";
    
    // Initialize the knowledge base
    println!("\n🔍 Loading knowledge base from: {}", kb_path);
    let kb = KnowledgeBase::new(kb_path).await?;
    
    // Get stats about the loaded knowledge base
    let stats = kb.get_stats();
    println!("\n📊 Knowledge Base Statistics:");
    println!("  - Total drug rules loaded: {}", stats.drug_rules_count);
    println!("  - Total DDI rules loaded: {}", stats.ddi_rules_count);
    
    // List all loaded drug IDs
    let drug_ids = kb.get_all_drug_ids();
    println!("\n💊 Loaded Drug Rules:");
    for drug_id in drug_ids {
        if let Some(rules) = kb.get_drug_rules(&drug_id) {
            println!("  - {} (v{})", rules.meta.generic_name, rules.meta.version);
            println!("    Evidence: {:?}", rules.meta.evidence_sources);
        }
    }
    
    // Check for warfarin specifically
    if let Some(warfarin_rules) = kb.get_drug_rules("warfarin") {
        println!("\n✅ Warfarin rules loaded successfully!");
        println!("  - Generic name: {}", warfarin_rules.meta.generic_name);
        println!("  - Version: {}", warfarin_rules.meta.version);
        println!("  - Evidence: {:?}", warfarin_rules.meta.evidence_sources);
    } else {
        println!("\n❌ Warfarin rules not found!");
    }
    
    println!("\n✅ All TOML files parsed successfully!");
    Ok(())
}
