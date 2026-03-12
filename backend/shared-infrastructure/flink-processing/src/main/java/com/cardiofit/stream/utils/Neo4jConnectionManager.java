package com.cardiofit.stream.utils;

import org.neo4j.driver.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;

/**
 * Neo4jConnectionManager manages Neo4j database connections
 */
public class Neo4jConnectionManager implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(Neo4jConnectionManager.class);

    private final String uri;
    private final String username;
    private final String password;
    private final String database;

    private transient Driver driver;

    public Neo4jConnectionManager(String uri, String username, String password, String database) {
        this.uri = uri;
        this.username = username;
        this.password = password;
        this.database = database;
    }

    public Neo4jConnectionManager() {
        this.uri = System.getenv().getOrDefault("NEO4J_URI", "bolt://localhost:7687");
        this.username = System.getenv().getOrDefault("NEO4J_USERNAME", "neo4j");
        this.password = System.getenv().getOrDefault("NEO4J_PASSWORD", "CardioFit2024!");
        this.database = System.getenv().getOrDefault("NEO4J_DATABASE", "cardiofit");
    }

    public synchronized Driver getDriver() {
        if (driver == null) {
            try {
                driver = GraphDatabase.driver(uri, AuthTokens.basic(username, password));
                LOG.info("Neo4j driver initialized for {}", uri);
            } catch (Exception e) {
                LOG.error("Failed to initialize Neo4j driver", e);
                throw new RuntimeException("Failed to connect to Neo4j", e);
            }
        }
        return driver;
    }

    public Session getSession() {
        return getDriver().session(SessionConfig.forDatabase(database));
    }

    public void close() {
        if (driver != null) {
            try {
                driver.close();
                LOG.info("Neo4j driver closed");
            } catch (Exception e) {
                LOG.error("Error closing Neo4j driver", e);
            }
        }
    }

    public boolean testConnection() {
        try (Session session = getSession()) {
            session.run("RETURN 1").consume();
            return true;
        } catch (Exception e) {
            LOG.error("Neo4j connection test failed", e);
            return false;
        }
    }
}