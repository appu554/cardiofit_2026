#!/bin/bash

# Add CanonicalEvent import to all Java files that reference it but don't have the import
files=(
  "src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java"
  "src/main/java/com/cardiofit/flink/operators/Module2_ContextAssembly.java"
  "src/main/java/com/cardiofit/flink/operators/Module4_PatternDetection.java"
  "src/main/java/com/cardiofit/flink/operators/Module5_MLInference.java"
  "src/main/java/com/cardiofit/flink/operators/Module6_EgressRouting.java"
  "src/main/java/com/cardiofit/flink/operators/TransactionalMultiSinkRouter.java"
  "src/main/java/com/cardiofit/flink/sinks/ClickHouseSink.java"
  "src/main/java/com/cardiofit/flink/sinks/ElasticsearchSink.java"
  "src/main/java/com/cardiofit/flink/sinks/GoogleFHIRStoreSink.java"
  "src/main/java/com/cardiofit/flink/sinks/Neo4jGraphSink.java"
  "src/main/java/com/cardiofit/flink/state/HealthcareStateDescriptors.java"
  "src/main/java/com/cardiofit/flink/monitoring/HealthcareMetrics.java"
  "src/main/java/com/cardiofit/flink/error/HealthcareErrorHandling.java"
  "src/main/java/com/cardiofit/stream/sinks/FHIRStoreSink.java"
  "src/main/java/com/cardiofit/stream/functions/EventEnrichmentFunction.java"
  "src/main/java/com/cardiofit/stream/jobs/PatientEventEnrichmentJob.java"
)

for file in "${files[@]}"; do
  if [ -f "$file" ]; then
    # Check if file already has the import
    if ! grep -q "import com.cardiofit.stream.models.CanonicalEvent;" "$file"; then
      # Find the last import line and add our import after it
      awk '
        /^import/ { imports[NR] = $0 }
        /^$/ && last_import_line == 0 && NR > 1 {
          # Found first blank line after imports
          for (i in imports) print imports[i]
          print "import com.cardiofit.stream.models.CanonicalEvent;"
          print ""
          last_import_line = NR
          next
        }
        { if (!/^import/ && last_import_line == 0 && NF > 0) {
            # Non-import, non-blank line found, insert imports here
            for (i in imports) print imports[i]
            print "import com.cardiofit.stream.models.CanonicalEvent;"
            print ""
            last_import_line = NR
          }
          if (last_import_line > 0) print $0
        }
      ' "$file" > "$file.tmp" && mv "$file.tmp" "$file"
      echo "Added import to $file"
    else
      echo "Import already exists in $file"
    fi
  else
    echo "File not found: $file"
  fi
done