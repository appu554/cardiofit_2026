#!/bin/bash

echo "Fixing all compilation errors comprehensively..."

# Fix missing methods in SemanticEvent
cat >> src/main/java/com/cardiofit/flink/models/SemanticEvent.java << 'EOF'

    public boolean hasPatientRelationships() {
        return patientRelationships != null && !patientRelationships.isEmpty();
    }

    public boolean hasClinicalConceptRelationships() {
        return clinicalConceptRelationships != null && !clinicalConceptRelationships.isEmpty();
    }
EOF

# Fix missing methods in PatternEvent
cat >> src/main/java/com/cardiofit/flink/models/PatternEvent.java << 'EOF'

    public String getUrgency() {
        if (severity == Severity.CRITICAL) return "IMMEDIATE";
        if (severity == Severity.HIGH) return "URGENT";
        if (severity == Severity.MEDIUM) return "MODERATE";
        return "LOW";
    }
EOF

# Fix missing methods in MLPrediction
cat >> src/main/java/com/cardiofit/flink/models/MLPrediction.java << 'EOF'

    public double getConfidence() {
        return confidence != null ? confidence : 0.0;
    }
EOF

# Create DetectedPattern inner class in Module6_EgressRouting
sed -i.bak '/class RoutedEventToEnrichedEventMapper/a\
    public static class DetectedPattern {\
        private String patternType;\
        private double confidence;\
        public DetectedPattern(String type, double conf) { this.patternType = type; this.confidence = conf; }\
        public String getPatternType() { return patternType; }\
        public double getConfidence() { return confidence; }\
    }\
' src/main/java/com/cardiofit/flink/operators/Module6_EgressRouting.java

# Fix DataStream.name() to uid()
find src -name "*.java" -exec sed -i.bak 's/\.name("/\.uid("/g' {} \;

# Fix getSideOutput - need to cast to SingleOutputStreamOperator
find src -name "Module6_EgressRouting.java" -exec sed -i.bak 's/DataStream<RoutedEvent> transformedEvents/SingleOutputStreamOperator<RoutedEvent> transformedEvents/g' {} \;

# Add SingleOutputStreamOperator import
for file in src/main/java/com/cardiofit/flink/operators/*.java; do
  if ! grep -q "import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;" "$file"; then
    sed -i.bak '/import org.apache.flink.streaming.api.datastream.DataStream;/a\
import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;' "$file"
  fi
done

# Fix window operations - need keyBy first
sed -i.bak 's/\.window(SlidingEventTimeWindows/\.keyBy(event -> event.getPatientId()).window(SlidingEventTimeWindows/g' src/main/java/com/cardiofit/flink/operators/Module4_PatternDetection.java
sed -i.bak 's/\.window(TumblingEventTimeWindows/\.keyBy(event -> event.getPatientId()).window(TumblingEventTimeWindows/g' src/main/java/com/cardiofit/flink/operators/Module4_PatternDetection.java

# Fix getPatientId() on Object - need to cast
find src -name "*.java" -exec sed -i.bak 's/event\.getPatientId()/((CanonicalEvent)event).getPatientId()/g' {} \;

# Fix Google FHIR HttpBody issue
sed -i.bak 's/import com.google.api.services.healthcare.v1.model.HttpBody;/&\nimport com.google.api.services.healthcare.v1.model.HttpBody;/' src/main/java/com/cardiofit/flink/sinks/GoogleFHIRStoreSink.java
sed -i.bak 's/\.executePatient(fhirStore, resource)/\.executePatient(fhirStore, new HttpBody().setData(resource.getBytes()))/g' src/main/java/com/cardiofit/flink/sinks/GoogleFHIRStoreSink.java

# Fix getTaskInfo() - doesn't exist, use getRuntimeContext
sed -i.bak 's/ctx\.getTaskInfo()/getRuntimeContext().getTaskName()/g' src/main/java/com/cardiofit/flink/operators/Module1_Ingestion.java

# Add missing constants in ClinicalPathwayAdherenceFunction
cat >> src/main/java/com/cardiofit/stream/patterns/ClinicalPathwayAdherenceFunction.java << 'EOF'

    private static final String MEDICATION_PRESCRIBED = "MEDICATION_PRESCRIBED";
    private static final String MEDICATION_DISCONTINUED = "MEDICATION_DISCONTINUED";
EOF

# Try to compile
echo "Attempting compilation..."
mvn compile -DskipTests 2>&1 | tee compile.log

echo "Check compile.log for any remaining errors."