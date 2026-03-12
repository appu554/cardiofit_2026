use criterion::{black_box, criterion_group, criterion_main, Criterion};
use flow2_rust_engine::*;

fn benchmark_rule_evaluation(c: &mut Criterion) {
    c.bench_function("rule_evaluation", |b| {
        b.iter(|| {
            // Placeholder benchmark - will be implemented later
            black_box(1 + 1)
        })
    });
}

criterion_group!(benches, benchmark_rule_evaluation);
criterion_main!(benches);
