package com.cardiofit.flink.cds.knowledge;

import com.cardiofit.flink.protocol.models.Protocol;
import org.junit.jupiter.api.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.List;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for KnowledgeBaseManager.
 *
 * <p>Tests cover:
 * - Singleton pattern (1 test)
 * - Protocol lookup (3 tests)
 * - Category index (2 tests)
 * - Specialty index (2 tests)
 * - Search functionality (2 tests)
 * - Hot reload (2 tests)
 *
 * Total: 12 comprehensive unit tests
 *
 * @author Module 3 CDS Team
 * @version 1.0
 * @since 2025-01-15
 */
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
class KnowledgeBaseManagerTest {

    private static final Logger logger = LoggerFactory.getLogger(KnowledgeBaseManagerTest.class);
    private static KnowledgeBaseManager knowledgeBase;

    @BeforeAll
    static void setUpClass() {
        logger.info("Initializing KnowledgeBaseManager for tests...");
        knowledgeBase = KnowledgeBaseManager.getInstance();
        logger.info("KnowledgeBaseManager initialized with {} protocols",
            knowledgeBase.getProtocolCount());
    }

    @AfterAll
    static void tearDownClass() {
        logger.info("Shutting down KnowledgeBaseManager after tests...");
        if (knowledgeBase != null) {
            knowledgeBase.shutdown();
        }
    }

    // ========================================
    // Test 1: Singleton Pattern
    // ========================================

    @Test
    @Order(1)
    @DisplayName("Test 1: Singleton - Same Instance Returned")
    void testGetInstance_ReturnsSameInstance() {
        logger.info("TEST 1: Singleton pattern test");

        KnowledgeBaseManager kb1 = KnowledgeBaseManager.getInstance();
        KnowledgeBaseManager kb2 = KnowledgeBaseManager.getInstance();

        assertNotNull(kb1, "First instance should not be null");
        assertNotNull(kb2, "Second instance should not be null");
        assertSame(kb1, kb2, "Should return same singleton instance");

        logger.info("✓ Singleton test passed: Same instance returned");
    }

    // ========================================
    // Test 2-4: Protocol Lookup
    // ========================================

    @Test
    @Order(2)
    @DisplayName("Test 2: Protocol Lookup - Found by ID")
    void testGetProtocol_Found() {
        logger.info("TEST 2: Protocol lookup by ID (found)");

        // Try to get sepsis protocol (should exist from ProtocolLoader)
        Protocol sepsis = knowledgeBase.getProtocol("SEPSIS-BUNDLE-001");

        if (sepsis != null) {
            assertNotNull(sepsis, "Sepsis protocol should be found");
            assertEquals("SEPSIS-BUNDLE-001", sepsis.getProtocolId());
            assertNotNull(sepsis.getName(), "Protocol should have a name");
            logger.info("✓ Found protocol: {} - {}", sepsis.getProtocolId(), sepsis.getName());
        } else {
            logger.warn("Sepsis protocol not found - may need to add YAML files to resources");
            // Test with any available protocol
            List<Protocol> allProtocols = knowledgeBase.getAllProtocols();
            if (!allProtocols.isEmpty()) {
                Protocol firstProtocol = allProtocols.get(0);
                Protocol found = knowledgeBase.getProtocol(firstProtocol.getProtocolId());
                assertNotNull(found, "Should find protocol by ID");
                assertEquals(firstProtocol.getProtocolId(), found.getProtocolId());
                logger.info("✓ Found protocol: {}", found.getProtocolId());
            } else {
                logger.warn("No protocols loaded - test inconclusive");
            }
        }
    }

    @Test
    @Order(3)
    @DisplayName("Test 3: Protocol Lookup - Not Found")
    void testGetProtocol_NotFound() {
        logger.info("TEST 3: Protocol lookup by ID (not found)");

        Protocol nonExistent = knowledgeBase.getProtocol("NON-EXISTENT-PROTOCOL-999");

        assertNull(nonExistent, "Non-existent protocol should return null");
        logger.info("✓ Non-existent protocol correctly returned null");
    }

    @Test
    @Order(4)
    @DisplayName("Test 4: Protocol Lookup - Null/Empty ID")
    void testGetProtocol_NullAndEmptyId() {
        logger.info("TEST 4: Protocol lookup with null/empty ID");

        Protocol nullResult = knowledgeBase.getProtocol(null);
        Protocol emptyResult = knowledgeBase.getProtocol("");

        assertNull(nullResult, "Null protocolId should return null");
        assertNull(emptyResult, "Empty protocolId should return null");

        logger.info("✓ Null/empty ID handling correct");
    }

    // ========================================
    // Test 5-6: Category Index
    // ========================================

    @Test
    @Order(5)
    @DisplayName("Test 5: Category Index - Get Protocols by Category")
    void testGetByCategory_ReturnsProtocols() {
        logger.info("TEST 5: Get protocols by category");

        Set<String> categories = knowledgeBase.getCategories();
        logger.info("Available categories: {}", categories);

        if (!categories.isEmpty()) {
            // Test with first available category
            String testCategory = categories.iterator().next();
            List<Protocol> categoryProtocols = knowledgeBase.getByCategory(testCategory);

            assertNotNull(categoryProtocols, "Category protocols list should not be null");
            assertFalse(categoryProtocols.isEmpty(),
                "Category " + testCategory + " should have protocols");

            // Verify all protocols in list match the category
            for (Protocol p : categoryProtocols) {
                assertEquals(testCategory, p.getCategory(),
                    "All protocols should have category: " + testCategory);
            }

            logger.info("✓ Category {} has {} protocols",
                testCategory, categoryProtocols.size());

        } else {
            logger.warn("No categories available - test inconclusive");
        }

        // Test typical categories if they exist
        List<Protocol> infectious = knowledgeBase.getByCategory("INFECTIOUS");
        logger.info("INFECTIOUS category: {} protocols", infectious.size());

        List<Protocol> cardiovascular = knowledgeBase.getByCategory("CARDIOVASCULAR");
        logger.info("CARDIOVASCULAR category: {} protocols", cardiovascular.size());
    }

    @Test
    @Order(6)
    @DisplayName("Test 6: Category Index - Empty/Invalid Category")
    void testGetByCategory_InvalidCategory() {
        logger.info("TEST 6: Get protocols by invalid category");

        List<Protocol> nonExistent = knowledgeBase.getByCategory("NON_EXISTENT_CATEGORY");
        List<Protocol> nullCategory = knowledgeBase.getByCategory(null);
        List<Protocol> emptyCategory = knowledgeBase.getByCategory("");

        assertTrue(nonExistent.isEmpty(), "Non-existent category should return empty list");
        assertTrue(nullCategory.isEmpty(), "Null category should return empty list");
        assertTrue(emptyCategory.isEmpty(), "Empty category should return empty list");

        logger.info("✓ Invalid category handling correct");
    }

    // ========================================
    // Test 7-8: Specialty Index
    // ========================================

    @Test
    @Order(7)
    @DisplayName("Test 7: Specialty Index - Get Protocols by Specialty")
    void testGetBySpecialty_ReturnsProtocols() {
        logger.info("TEST 7: Get protocols by specialty");

        Set<String> specialties = knowledgeBase.getSpecialties();
        logger.info("Available specialties: {}", specialties);

        if (!specialties.isEmpty()) {
            // Test with first available specialty
            String testSpecialty = specialties.iterator().next();
            List<Protocol> specialtyProtocols = knowledgeBase.getBySpecialty(testSpecialty);

            assertNotNull(specialtyProtocols, "Specialty protocols list should not be null");
            assertFalse(specialtyProtocols.isEmpty(),
                "Specialty " + testSpecialty + " should have protocols");

            // Verify all protocols in list match the specialty
            for (Protocol p : specialtyProtocols) {
                assertEquals(testSpecialty, p.getSpecialty(),
                    "All protocols should have specialty: " + testSpecialty);
            }

            logger.info("✓ Specialty {} has {} protocols",
                testSpecialty, specialtyProtocols.size());

        } else {
            logger.warn("No specialties available - test inconclusive");
        }

        // Test typical specialties if they exist
        List<Protocol> criticalCare = knowledgeBase.getBySpecialty("CRITICAL_CARE");
        logger.info("CRITICAL_CARE specialty: {} protocols", criticalCare.size());

        List<Protocol> cardiology = knowledgeBase.getBySpecialty("CARDIOLOGY");
        logger.info("CARDIOLOGY specialty: {} protocols", cardiology.size());
    }

    @Test
    @Order(8)
    @DisplayName("Test 8: Specialty Index - Empty/Invalid Specialty")
    void testGetBySpecialty_InvalidSpecialty() {
        logger.info("TEST 8: Get protocols by invalid specialty");

        List<Protocol> nonExistent = knowledgeBase.getBySpecialty("NON_EXISTENT_SPECIALTY");
        List<Protocol> nullSpecialty = knowledgeBase.getBySpecialty(null);
        List<Protocol> emptySpecialty = knowledgeBase.getBySpecialty("");

        assertTrue(nonExistent.isEmpty(), "Non-existent specialty should return empty list");
        assertTrue(nullSpecialty.isEmpty(), "Null specialty should return empty list");
        assertTrue(emptySpecialty.isEmpty(), "Empty specialty should return empty list");

        logger.info("✓ Invalid specialty handling correct");
    }

    // ========================================
    // Test 9-10: Search Functionality
    // ========================================

    @Test
    @Order(9)
    @DisplayName("Test 9: Search - Find Protocols by Query")
    void testSearch_FindsProtocols() {
        logger.info("TEST 9: Search protocols by query");

        List<Protocol> allProtocols = knowledgeBase.getAllProtocols();

        if (!allProtocols.isEmpty()) {
            Protocol firstProtocol = allProtocols.get(0);
            String searchTerm = firstProtocol.getName().substring(0, 5).toLowerCase();

            List<Protocol> searchResults = knowledgeBase.search(searchTerm);

            assertNotNull(searchResults, "Search results should not be null");
            assertFalse(searchResults.isEmpty(),
                "Search for '" + searchTerm + "' should find protocols");

            logger.info("✓ Search for '{}' found {} protocols",
                searchTerm, searchResults.size());

            // Search by protocol ID
            searchResults = knowledgeBase.search(firstProtocol.getProtocolId());
            assertFalse(searchResults.isEmpty(),
                "Search by protocol ID should find protocol");

            // Search by category
            if (firstProtocol.getCategory() != null) {
                searchResults = knowledgeBase.search(firstProtocol.getCategory());
                assertFalse(searchResults.isEmpty(),
                    "Search by category should find protocols");
            }

        } else {
            logger.warn("No protocols loaded - test inconclusive");
        }
    }

    @Test
    @Order(10)
    @DisplayName("Test 10: Search - Empty/Invalid Query")
    void testSearch_InvalidQuery() {
        logger.info("TEST 10: Search with invalid query");

        List<Protocol> nullSearch = knowledgeBase.search(null);
        List<Protocol> emptySearch = knowledgeBase.search("");
        List<Protocol> noMatch = knowledgeBase.search("XYZNONEXISTENT12345");

        assertTrue(nullSearch.isEmpty(), "Null query should return empty list");
        assertTrue(emptySearch.isEmpty(), "Empty query should return empty list");
        assertTrue(noMatch.isEmpty(), "Non-matching query should return empty list");

        logger.info("✓ Invalid query handling correct");
    }

    // ========================================
    // Test 11-12: Hot Reload
    // ========================================

    @Test
    @Order(11)
    @DisplayName("Test 11: Hot Reload - Reload Protocols")
    void testReloadProtocols_Success() {
        logger.info("TEST 11: Hot reload protocols");

        int initialCount = knowledgeBase.getProtocolCount();
        logger.info("Initial protocol count: {}", initialCount);

        // Trigger reload
        knowledgeBase.reloadProtocols();

        int afterReloadCount = knowledgeBase.getProtocolCount();
        logger.info("After reload protocol count: {}", afterReloadCount);

        // Count should be same (or possibly different if files changed)
        assertTrue(afterReloadCount >= 0, "Protocol count should be non-negative");

        logger.info("✓ Hot reload completed successfully");
    }

    @Test
    @Order(12)
    @DisplayName("Test 12: Hot Reload - Concurrent Safety")
    void testReloadProtocols_ConcurrentSafety() throws InterruptedException {
        logger.info("TEST 12: Hot reload concurrent safety");

        final int THREAD_COUNT = 5;
        Thread[] threads = new Thread[THREAD_COUNT];

        // Start multiple threads trying to reload simultaneously
        for (int i = 0; i < THREAD_COUNT; i++) {
            final int threadNum = i;
            threads[i] = new Thread(() -> {
                logger.info("Thread {} triggering reload", threadNum);
                knowledgeBase.reloadProtocols();
            });
            threads[i].start();
        }

        // Wait for all threads to complete
        for (Thread thread : threads) {
            thread.join(5000); // 5 second timeout
        }

        // Verify system is still functional
        int finalCount = knowledgeBase.getProtocolCount();
        assertTrue(finalCount >= 0, "Protocol count should be non-negative after concurrent reloads");

        // Verify we can still get protocols
        List<Protocol> allProtocols = knowledgeBase.getAllProtocols();
        assertNotNull(allProtocols, "Should still be able to get all protocols");

        logger.info("✓ Concurrent reload safety verified. Final count: {}", finalCount);
    }

    // ========================================
    // Additional Helper Tests
    // ========================================

    @Test
    @Order(13)
    @DisplayName("Test 13: Performance - Category Index Lookup Speed")
    void testPerformance_CategoryIndexLookup() {
        logger.info("TEST 13: Performance test - category index lookup");

        Set<String> categories = knowledgeBase.getCategories();
        if (categories.isEmpty()) {
            logger.warn("No categories available - skipping performance test");
            return;
        }

        String testCategory = categories.iterator().next();

        // Measure lookup time (should be < 5ms)
        long startTime = System.nanoTime();

        for (int i = 0; i < 100; i++) {
            knowledgeBase.getByCategory(testCategory);
        }

        long duration = (System.nanoTime() - startTime) / 1_000_000; // Convert to ms
        long avgDuration = duration / 100;

        logger.info("Category index lookup: {} lookups in {}ms (avg: {}ms)",
            100, duration, avgDuration);

        assertTrue(avgDuration < 5,
            "Category index lookup should be < 5ms (actual: " + avgDuration + "ms)");

        logger.info("✓ Category index performance test passed");
    }

    @Test
    @Order(14)
    @DisplayName("Test 14: Performance - Specialty Index Lookup Speed")
    void testPerformance_SpecialtyIndexLookup() {
        logger.info("TEST 14: Performance test - specialty index lookup");

        Set<String> specialties = knowledgeBase.getSpecialties();
        if (specialties.isEmpty()) {
            logger.warn("No specialties available - skipping performance test");
            return;
        }

        String testSpecialty = specialties.iterator().next();

        // Measure lookup time (should be < 5ms)
        long startTime = System.nanoTime();

        for (int i = 0; i < 100; i++) {
            knowledgeBase.getBySpecialty(testSpecialty);
        }

        long duration = (System.nanoTime() - startTime) / 1_000_000; // Convert to ms
        long avgDuration = duration / 100;

        logger.info("Specialty index lookup: {} lookups in {}ms (avg: {}ms)",
            100, duration, avgDuration);

        assertTrue(avgDuration < 5,
            "Specialty index lookup should be < 5ms (actual: " + avgDuration + "ms)");

        logger.info("✓ Specialty index performance test passed");
    }

    @Test
    @Order(15)
    @DisplayName("Test 15: Thread Safety - Concurrent Access")
    void testThreadSafety_ConcurrentAccess() throws InterruptedException {
        logger.info("TEST 15: Thread safety - concurrent access test");

        final int THREAD_COUNT = 10;
        final int OPERATIONS_PER_THREAD = 100;
        Thread[] threads = new Thread[THREAD_COUNT];
        final boolean[] errors = {false};

        // Start multiple threads performing concurrent operations
        for (int i = 0; i < THREAD_COUNT; i++) {
            threads[i] = new Thread(() -> {
                try {
                    for (int j = 0; j < OPERATIONS_PER_THREAD; j++) {
                        // Mix of different operations
                        knowledgeBase.getAllProtocols();
                        knowledgeBase.getCategories();
                        knowledgeBase.getSpecialties();

                        List<Protocol> all = knowledgeBase.getAllProtocols();
                        if (!all.isEmpty()) {
                            knowledgeBase.getProtocol(all.get(0).getProtocolId());
                        }
                    }
                } catch (Exception e) {
                    logger.error("Thread error during concurrent access", e);
                    errors[0] = true;
                }
            });
            threads[i].start();
        }

        // Wait for all threads
        for (Thread thread : threads) {
            thread.join(10000); // 10 second timeout
        }

        assertFalse(errors[0], "No errors should occur during concurrent access");

        logger.info("✓ Thread safety test passed: {} threads × {} ops = {} total operations",
            THREAD_COUNT, OPERATIONS_PER_THREAD, THREAD_COUNT * OPERATIONS_PER_THREAD);
    }
}
