package com.cardiofit.flink.test;

import org.apache.kafka.clients.admin.AdminClient;
import org.apache.kafka.clients.admin.AdminClientConfig;
import org.apache.kafka.clients.admin.ListTopicsResult;
import java.util.Properties;
import java.util.Set;

public class SimpleKafkaTest {
    public static void main(String[] args) {
        System.out.println("Starting Simple Kafka Connection Test");

        Properties props = new Properties();
        props.put(AdminClientConfig.BOOTSTRAP_SERVERS_CONFIG, "kafka:9092");
        props.put(AdminClientConfig.REQUEST_TIMEOUT_MS_CONFIG, "5000");
        props.put(AdminClientConfig.DEFAULT_API_TIMEOUT_MS_CONFIG, "5000");

        System.out.println("Connecting to Kafka at: kafka:9092");

        try (AdminClient adminClient = AdminClient.create(props)) {
            System.out.println("Admin client created successfully");

            ListTopicsResult topics = adminClient.listTopics();
            Set<String> topicNames = topics.names().get();

            System.out.println("Successfully connected to Kafka!");
            System.out.println("Found " + topicNames.size() + " topics:");
            for (String topic : topicNames) {
                System.out.println("  - " + topic);
            }
        } catch (Exception e) {
            System.err.println("Failed to connect to Kafka:");
            e.printStackTrace();
        }
    }
}