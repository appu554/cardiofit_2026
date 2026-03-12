use criterion::{black_box, criterion_group, criterion_main, Criterion, BenchmarkId};
use flow2_rust_engine::{
    unified_clinical_engine::{UnifiedClinicalEngine, knowledge_base::KnowledgeBase, ClinicalRequest, PatientContext},
    models::{BiologicalSex, PregnancyStatus, RenalFunction, HepaticFunction},
};
use std::{sync::Arc, collections::HashMap};
use chrono::Utc;
use tokio::runtime::Runtime;

fn create_test_engine() -> Arc<UnifiedClinicalEngine> {
    let rt = Runtime::new().unwrap();
    rt.block_on(async {
        let knowledge_base = Arc::new(KnowledgeBase::new("../knowledge".to_string()).unwrap());
        Arc::new(UnifiedClinicalEngine::new(knowledge_base).unwrap())
    })
}

fn create_test_request(drug_id: &str) -> ClinicalRequest {
    ClinicalRequest {
        request_id: format!("bench-{}", uuid::Uuid::new_v4()),
        drug_id: drug_id.to_string(),
        indication: "hypertension".to_string(),
        patient_context: PatientContext {
            age_years: 45,
            weight_kg: 70.0,
            height_cm: 170.0,
            sex: BiologicalSex::Male,
            pregnancy_status: PregnancyStatus::NotApplicable,
            renal_function: RenalFunction::default(),
            hepatic_function: HepaticFunction::default(),
            active_medications: vec![],
            allergies: vec![],
            conditions: vec![],
            lab_values: HashMap::new(),
        },
        timestamp: Utc::now(),
    }
}

fn bench_dose_calculation(c: &mut Criterion) {
    let engine = create_test_engine();
    let rt = Runtime::new().unwrap();

    let mut group = c.benchmark_group("dose_calculation");
    
    // Test different drug types
    let drugs = vec!["metformin", "lisinopril", "atorvastatin"];
    
    for drug in drugs {
        group.bench_with_input(
            BenchmarkId::new("unified_engine", drug),
            &drug,
            |b, &drug| {
                b.to_async(&rt).iter(|| async {
                    let request = create_test_request(drug);
                    black_box(engine.process_clinical_request(request).await)
                });
            },
        );
    }
    
    group.finish();
}

fn bench_concurrent_requests(c: &mut Criterion) {
    let engine = create_test_engine();
    let rt = Runtime::new().unwrap();

    let mut group = c.benchmark_group("concurrent_processing");
    
    for concurrency in [1, 5, 10, 20].iter() {
        group.bench_with_input(
            BenchmarkId::new("concurrent_requests", concurrency),
            concurrency,
            |b, &concurrency| {
                b.to_async(&rt).iter(|| async {
                    let mut handles = Vec::new();
                    
                    for i in 0..concurrency {
                        let engine = engine.clone();
                        let handle = tokio::spawn(async move {
                            let request = create_test_request("metformin");
                            engine.process_clinical_request(request).await
                        });
                        handles.push(handle);
                    }
                    
                    for handle in handles {
                        black_box(handle.await.unwrap());
                    }
                });
            },
        );
    }
    
    group.finish();
}

fn bench_knowledge_base_access(c: &mut Criterion) {
    let engine = create_test_engine();
    
    c.bench_function("knowledge_base_lookup", |b| {
        b.iter(|| {
            // Simulate knowledge base access patterns
            black_box(engine.get_drug_rules("metformin"));
        });
    });
}

criterion_group!(
    benches,
    bench_dose_calculation,
    bench_concurrent_requests,
    bench_knowledge_base_access
);
criterion_main!(benches);
