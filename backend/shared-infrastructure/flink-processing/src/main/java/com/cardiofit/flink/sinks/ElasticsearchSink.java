
package com.cardiofit.flink.sinks;

import com.cardiofit.flink.models.RoutedEvent;
import com.cardiofit.flink.models.CanonicalEvent;
import org.apache.flink.api.connector.sink2.Sink;
import org.apache.flink.api.connector.sink2.WriterInitContext;
import org.apache.flink.api.connector.sink2.SinkWriter;
import org.apache.http.HttpHost;
import org.apache.http.auth.AuthScope;
import org.apache.http.auth.UsernamePasswordCredentials;
import org.apache.http.client.CredentialsProvider;
import org.apache.http.impl.client.BasicCredentialsProvider;
import org.elasticsearch.action.bulk.BulkProcessor;
import org.elasticsearch.action.bulk.BulkRequest;
import org.elasticsearch.action.bulk.BulkResponse;
import org.elasticsearch.action.index.IndexRequest;
import org.elasticsearch.client.RequestOptions;
import org.elasticsearch.client.RestClient;
import org.elasticsearch.client.RestClientBuilder;
import org.elasticsearch.client.RestHighLevelClient;
import org.elasticsearch.common.unit.ByteSizeUnit;
import org.elasticsearch.common.unit.ByteSizeValue;
import org.elasticsearch.xcontent.XContentType;
import org.elasticsearch.core.TimeValue;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.time.LocalDate;
import java.time.format.DateTimeFormatter;
import java.util.Map;
import java.util.HashMap;
import java.util.concurrent.TimeUnit;

/**
 * Elasticsearch sink for analytics and search capabilities
 * Stores enriched clinical events with full-text search support
 * Supports time-based indexing for efficient data management
 *
 * Migrated to Flink 2.x Sink API (replaces RichSinkFunction)
 */
public class ElasticsearchSink implements Sink<RoutedEvent> {

    private static final Logger LOG = LoggerFactory.getLogger(ElasticsearchSink.class);

    // Configuration
    private final String elasticsearchHost;
    private final int elasticsearchPort;
    private final String username;
    private final String password;
    private final String indexPrefix;
    private final boolean useTimeBasedIndex;

    public ElasticsearchSink(String host, int port, String username, String password, String indexPrefix) {
        this.elasticsearchHost = host;
        this.elasticsearchPort = port;
        this.username = username;
        this.password = password;
        this.indexPrefix = indexPrefix;
        this.useTimeBasedIndex = true;
    }

    public ElasticsearchSink() {
        // Default configuration matching shared infrastructure setup
        this.elasticsearchHost = System.getenv().getOrDefault("ELASTICSEARCH_HOST", "localhost");
        this.elasticsearchPort = Integer.parseInt(System.getenv().getOrDefault("ELASTICSEARCH_PORT", "9200"));
        this.username = System.getenv().getOrDefault("ELASTICSEARCH_USERNAME", "elastic");
        this.password = System.getenv().getOrDefault("ELASTICSEARCH_PASSWORD", "ElasticCardioFit2024!");
        this.indexPrefix = System.getenv().getOrDefault("ELASTICSEARCH_INDEX_PREFIX", "cardiofit-clinical");
        this.useTimeBasedIndex = Boolean.parseBoolean(System.getenv().getOrDefault("ELASTICSEARCH_USE_TIME_INDEX", "true"));
    }

    @Override
    public SinkWriter<RoutedEvent> createWriter(WriterInitContext context) throws IOException {
        return new ElasticsearchSinkWriter(elasticsearchHost, elasticsearchPort, username, password, indexPrefix, useTimeBasedIndex);
    }

    /**
     * SinkWriter implementation for Elasticsearch bulk operations
     */
    private static class ElasticsearchSinkWriter implements SinkWriter<RoutedEvent> {

        private static final Logger LOG = LoggerFactory.getLogger(ElasticsearchSinkWriter.class);

        private final String elasticsearchHost;
        private final int elasticsearchPort;
        private final String username;
        private final String password;
        private final String indexPrefix;
        private final boolean useTimeBasedIndex;

        private transient RestHighLevelClient client;
        private transient BulkProcessor bulkProcessor;

        public ElasticsearchSinkWriter(String host, int port, String username, String password,
                                      String indexPrefix, boolean useTimeBasedIndex) {
            this.elasticsearchHost = host;
            this.elasticsearchPort = port;
            this.username = username;
            this.password = password;
            this.indexPrefix = indexPrefix;
            this.useTimeBasedIndex = useTimeBasedIndex;

            try {
                initializeElasticsearch();
            } catch (Exception e) {
                LOG.error("Failed to initialize Elasticsearch sink", e);
                throw new RuntimeException("Elasticsearch initialization failed", e);
            }
        }

        private void initializeElasticsearch() {
            // Create Elasticsearch client
            RestClientBuilder builder = RestClient.builder(new HttpHost(elasticsearchHost, elasticsearchPort, "http"));

            // Add authentication if credentials are provided
            if (username != null && !username.isEmpty() && password != null && !password.isEmpty()) {
                CredentialsProvider credentialsProvider = new BasicCredentialsProvider();
                credentialsProvider.setCredentials(AuthScope.ANY,
                    new UsernamePasswordCredentials(username, password));

                builder.setHttpClientConfigCallback(httpClientBuilder ->
                    httpClientBuilder.setDefaultCredentialsProvider(credentialsProvider));
            }

            client = new RestHighLevelClient(builder);

            // Create bulk processor for batch indexing
            BulkProcessor.Listener listener = new BulkProcessor.Listener() {
                @Override
                public void beforeBulk(long executionId, BulkRequest request) {
                    LOG.debug("Executing bulk request with {} actions", request.numberOfActions());
                }

                @Override
                public void afterBulk(long executionId, BulkRequest request, BulkResponse response) {
                    if (response.hasFailures()) {
                        LOG.error("Bulk request failed: {}", response.buildFailureMessage());
                    } else {
                        LOG.debug("Bulk request completed successfully with {} actions", request.numberOfActions());
                    }
                }

                @Override
                public void afterBulk(long executionId, BulkRequest request, Throwable failure) {
                    LOG.error("Bulk request failed with exception", failure);
                }
            };

            bulkProcessor = BulkProcessor.builder(
                    (request, bulkListener) -> {
                        try {
                            BulkResponse response = client.bulk(request, RequestOptions.DEFAULT);
                            bulkListener.onResponse(response);
                        } catch (IOException e) {
                            bulkListener.onFailure(e);
                        }
                    },
                    listener)
                .setBulkActions(100)  // Flush after 100 actions
                .setBulkSize(new ByteSizeValue(5, ByteSizeUnit.MB))  // Flush after 5MB
                .setFlushInterval(TimeValue.timeValueSeconds(5))  // Flush every 5 seconds
                .setConcurrentRequests(2)  // Use 2 concurrent bulk requests
                .build();

            LOG.info("Initialized Elasticsearch sink connecting to {}:{}", elasticsearchHost, elasticsearchPort);
        }

        @Override
        public void write(RoutedEvent event, Context context) throws IOException, InterruptedException {
            // Skip if not routed to analytics
            if (!event.hasDestination("analytics") && !event.hasDestination("elasticsearch")) {
                return;
            }

            String indexName = getIndexName(event);
            Map<String, Object> document = createDocument(event);

            IndexRequest indexRequest = new IndexRequest(indexName)
                .id(event.getId())
                .source(document, XContentType.JSON);

            // Add to bulk processor
            bulkProcessor.add(indexRequest);

            LOG.debug("Indexed event {} to Elasticsearch index {}", event.getId(), indexName);
        }

        private String getIndexName(RoutedEvent event) {
            if (useTimeBasedIndex) {
                // Create daily indices for efficient data management
                String dateSuffix = LocalDate.now().format(DateTimeFormatter.ofPattern("yyyy.MM.dd"));
                return String.format("%s-%s", indexPrefix, dateSuffix);
            }
            return indexPrefix;
        }

        private Map<String, Object> createDocument(RoutedEvent event) {
            Map<String, Object> document = new HashMap<>();

            // Core event fields
            document.put("event_id", event.getId());
            document.put("patient_id", event.getPatientId());
            document.put("event_type", event.getSourceEventType());
            document.put("timestamp", event.getRoutingTime());
            document.put("priority", event.getPriority().name());

            // Add clinical context if available
            if (event.getOriginalPayload() != null) {
                Map<String, Object> payload = new HashMap<>();
                // Flatten the original payload for better searchability
                flattenPayload(event.getOriginalPayload(), payload, "");
                document.put("clinical_data", payload);
            }

            // Add transformed data if available
            if (event.getTransformedPayloads() != null && event.getTransformedPayloads().containsKey("ANALYTICS_FLATTEN")) {
                document.put("analytics_data", event.getTransformedPayloads().get("ANALYTICS_FLATTEN"));
            }

            // Add metadata
            if (event.getTransformationMetadata() != null) {
                document.put("metadata", event.getTransformationMetadata());
            }

            // Add searchable text fields for full-text search
            document.put("search_text", buildSearchableText(event));

            return document;
        }

        private void flattenPayload(Object payload, Map<String, Object> result, String prefix) {
            if (payload instanceof Map) {
                @SuppressWarnings("unchecked")
                Map<String, Object> map = (Map<String, Object>) payload;
                for (Map.Entry<String, Object> entry : map.entrySet()) {
                    String key = prefix.isEmpty() ? entry.getKey() : prefix + "." + entry.getKey();
                    if (entry.getValue() instanceof Map) {
                        flattenPayload(entry.getValue(), result, key);
                    } else {
                        result.put(key, entry.getValue());
                    }
                }
            } else {
                result.put(prefix, payload);
            }
        }

        private String buildSearchableText(RoutedEvent event) {
            StringBuilder searchText = new StringBuilder();

            searchText.append(event.getPatientId()).append(" ");
            searchText.append(event.getSourceEventType()).append(" ");

            if (event.getOriginalPayload() != null) {
                searchText.append(extractTextFromPayload(event.getOriginalPayload()));
            }

            return searchText.toString();
        }

        private String extractTextFromPayload(Object payload) {
            // Extract meaningful text from the payload for full-text search
            // This is simplified - in production, you'd have more sophisticated extraction
            if (payload != null) {
                return payload.toString().replaceAll("[^a-zA-Z0-9\\s]", " ");
            }
            return "";
        }

        @Override
        public void flush(boolean endOfInput) throws IOException, InterruptedException {
            if (bulkProcessor != null) {
                // BulkProcessor auto-flushes based on configuration
                LOG.debug("Flush called (endOfInput: {})", endOfInput);
            }
        }

        @Override
        public void close() throws Exception {
            if (bulkProcessor != null) {
                bulkProcessor.awaitClose(10, TimeUnit.SECONDS);
            }
            if (client != null) {
                client.close();
            }
            LOG.info("Elasticsearch sink closed");
        }
    }
}
