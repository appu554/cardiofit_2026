#!/bin/bash

echo "Fixing remaining compilation errors..."

# Fix Elasticsearch TimeValue issue
sed -i.bak 's/BulkProcessor.builder((req, listener) -> bulkAsync(req, RequestOptions.DEFAULT, listener), listener)/BulkProcessor.builder((req, listener) -> client.bulkAsync(req, RequestOptions.DEFAULT, listener), listener)/g' src/main/java/com/cardiofit/flink/sinks/ElasticsearchSink.java
sed -i.bak 's/.setBulkActions(1000)/&.setBulkSize(new ByteSizeValue(5, ByteSizeUnit.MB))/g' src/main/java/com/cardiofit/flink/sinks/ElasticsearchSink.java
sed -i.bak 's/\.setFlushInterval(5000L)/\.setFlushInterval(TimeValue.timeValueSeconds(5))/g' src/main/java/com/cardiofit/flink/sinks/ElasticsearchSink.java

# Fix Google FHIR HttpBody
sed -i.bak 's/\.executePatient(fhirStore, resource)/\.executePatient(fhirStore, new HttpBody().setData(resource.getBytes()))/g' src/main/java/com/cardiofit/flink/sinks/GoogleFHIRStoreSink.java

# Add missing imports for TimeValue
sed -i.bak '1 s/^/import org.elasticsearch.core.TimeValue;\n/' src/main/java/com/cardiofit/flink/sinks/ElasticsearchSink.java

# Fix DataStream.name() calls (not available in all Flink versions)
find . -name "*.java" -exec sed -i.bak 's/\.name(".*")/\.uid("\1")/g' {} \;

# Fix getSideOutput calls (need SingleOutputStreamOperator)
find . -name "*.java" -exec sed -i.bak 's/DataStream<\([^>]*\)> \([^ ]*\) = \([^;]*\);/SingleOutputStreamOperator<\1> \2 = \3;/g' {} \;

# Fix window operations
sed -i.bak 's/\.window(SlidingEventTimeWindows/\.windowAll(SlidingEventTimeWindows/g' src/main/java/com/cardiofit/flink/operators/Module4_PatternDetection.java
sed -i.bak 's/\.window(TumblingEventTimeWindows/\.windowAll(TumblingEventTimeWindows/g' src/main/java/com/cardiofit/flink/operators/Module4_PatternDetection.java

# Add missing imports
echo "Adding missing imports..."
for file in src/main/java/com/cardiofit/flink/operators/*.java; do
  if ! grep -q "import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;" "$file"; then
    sed -i.bak '1s/^/import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;\n/' "$file"
  fi
done

echo "Running compilation..."
mvn compile -DskipTests

echo "If compilation still fails, check the errors and we can fix them specifically."