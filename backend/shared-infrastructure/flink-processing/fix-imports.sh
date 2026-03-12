#!/bin/bash

# Fix CanonicalEvent imports in all necessary files
files_needing_import=(
  "src/main/java/com/cardiofit/flink/sinks/RedisCacheSink.java"
  "src/main/java/com/cardiofit/flink/operators/Module3_SemanticMesh.java"
  "src/main/java/com/cardiofit/flink/patterns/ClinicalPatterns.java"
  "src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java"
  "src/main/java/com/cardiofit/flink/operators/Module2_ContextAssembly.java"
  "src/main/java/com/cardiofit/flink/operators/Module4_PatternDetection.java"
  "src/main/java/com/cardiofit/flink/operators/Module5_MLInference.java"
  "src/main/java/com/cardiofit/flink/operators/Module6_EgressRouting.java"
  "src/main/java/com/cardiofit/flink/sinks/ClickHouseSink.java"
  "src/main/java/com/cardiofit/flink/sinks/ElasticsearchSink.java"
  "src/main/java/com/cardiofit/flink/sinks/GoogleFHIRStoreSink.java"
  "src/main/java/com/cardiofit/flink/sinks/Neo4jGraphSink.java"
)

for file in "${files_needing_import[@]}"; do
  if [ -f "$file" ]; then
    # Check if the correct import already exists
    if ! grep -q "import com.cardiofit.stream.models.CanonicalEvent;" "$file"; then
      # Find the package line and add import after other imports
      sed -i.bak '/^package /,/^import /{
        /^import / {
          # At the last import line, add our import
          /^import.*$/ {
            # Check if this is the last import
            N
            if [[ $'\n'* != *"import"* ]]; then
              # This is the last import, add our import after it
              s/\(^import.*\)\(\n\)/\1\2import com.cardiofit.stream.models.CanonicalEvent;\2/
            else
              # Not the last import, put the line back
              s/\(.*\)\n\(.*\)/\1\n\2/
            fi
          }
        }
      }' "$file"
      echo "Added CanonicalEvent import to $file"
    else
      echo "CanonicalEvent import already exists in $file"
    fi
  else
    echo "File not found: $file"
  fi
done