package com.cardiofit.flink.loader;

import com.cardiofit.flink.models.diagnostics.LabTest;
import com.cardiofit.flink.models.diagnostics.ImagingStudy;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.InputStream;
import java.io.Serializable;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.stream.Collectors;
import java.util.stream.Stream;

/**
 * Diagnostic Test Loader
 *
 * Loads and caches YAML definitions for laboratory tests and imaging studies.
 * Provides thread-safe access to diagnostic test knowledge base for intelligent
 * test ordering and recommendation in the CardioFit clinical decision support system.
 *
 * <p>Features:
 * - Singleton pattern for consistent caching
 * - Thread-safe concurrent access
 * - Hot reload capability for updated definitions
 * - Multiple lookup methods (by ID, LOINC, CPT)
 * - Comprehensive error handling
 *
 * <p>YAML File Locations:
 * - Lab Tests: /knowledge-base/diagnostic-tests/lab-tests/**\/*.yaml
 * - Imaging Studies: /knowledge-base/diagnostic-tests/imaging/**\/*.yaml
 *
 * @author CardioFit Platform - Module 3 Phase 4
 * @version 1.0
 * @since 2025-10-23
 */
public class DiagnosticTestLoader implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(DiagnosticTestLoader.class);

    // Singleton instance
    private static volatile DiagnosticTestLoader instance;

    // YAML Parser
    private final ObjectMapper yamlMapper;

    // Thread-safe caches
    private final ConcurrentHashMap<String, LabTest> labTestsById;
    private final ConcurrentHashMap<String, LabTest> labTestsByLoinc;
    private final ConcurrentHashMap<String, ImagingStudy> imagingStudiesById;
    private final ConcurrentHashMap<String, ImagingStudy> imagingStudiesByCpt;

    // Resource paths
    private static final String LAB_TESTS_BASE_PATH = "/knowledge-base/diagnostic-tests/lab-tests";
    private static final String IMAGING_BASE_PATH = "/knowledge-base/diagnostic-tests/imaging";

    /**
     * Private constructor for singleton pattern.
     * Initializes YAML mapper and loads all test definitions.
     */
    private DiagnosticTestLoader() {
        LOG.info("Initializing DiagnosticTestLoader...");

        // Initialize Jackson YAML mapper
        this.yamlMapper = new ObjectMapper(new YAMLFactory());
        this.yamlMapper.registerModule(new JavaTimeModule());
        // Ignore unknown properties in YAML files (e.g., subcategory, loincDisplay, description)
        this.yamlMapper.configure(com.fasterxml.jackson.databind.DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
        // Allow Jackson to use setters and all-args constructors for deserialization
        this.yamlMapper.configure(com.fasterxml.jackson.databind.MapperFeature.USE_ANNOTATIONS, true);

        // Initialize concurrent caches
        this.labTestsById = new ConcurrentHashMap<>();
        this.labTestsByLoinc = new ConcurrentHashMap<>();
        this.imagingStudiesById = new ConcurrentHashMap<>();
        this.imagingStudiesByCpt = new ConcurrentHashMap<>();

        // Load all tests on initialization
        loadAllTests();
    }

    /**
     * Get singleton instance.
     * Double-checked locking for thread-safe lazy initialization.
     *
     * @return DiagnosticTestLoader singleton instance
     */
    public static DiagnosticTestLoader getInstance() {
        if (instance == null) {
            synchronized (DiagnosticTestLoader.class) {
                if (instance == null) {
                    instance = new DiagnosticTestLoader();
                }
            }
        }
        return instance;
    }

    /**
     * Load all lab tests and imaging studies from YAML files.
     * Scans resource directories and parses all YAML definitions.
     */
    private void loadAllTests() {
        LOG.info("Loading diagnostic test definitions from YAML files...");

        long startTime = System.currentTimeMillis();
        int labCount = 0;
        int imagingCount = 0;

        try {
            // Load lab tests
            labCount = loadLabTests();
            LOG.info("Loaded {} lab tests", labCount);

            // Load imaging studies
            imagingCount = loadImagingStudies();
            LOG.info("Loaded {} imaging studies", imagingCount);

            long elapsedMs = System.currentTimeMillis() - startTime;
            LOG.info("DiagnosticTestLoader initialization complete: {} lab tests, {} imaging studies in {}ms",
                    labCount, imagingCount, elapsedMs);

        } catch (Exception e) {
            LOG.error("Failed to load diagnostic test definitions: {}", e.getMessage(), e);
        }
    }

    /**
     * Load all lab test definitions from YAML files.
     *
     * @return Number of lab tests loaded
     */
    private int loadLabTests() {
        int count = 0;

        // Subdirectories under lab-tests (updated to match actual directory structure)
        String[] categories = {"chemistry", "hematology", "microbiology", "cardiac-markers", "urinalysis"};

        for (String category : categories) {
            String resourcePath = LAB_TESTS_BASE_PATH + "/" + category;
            List<String> yamlFiles = findYamlFiles(resourcePath);

            for (String yamlFile : yamlFiles) {
                try {
                    LabTest labTest = loadLabTest(yamlFile);
                    if (labTest != null) {
                        cacheLabTest(labTest);
                        count++;
                    }
                } catch (Exception e) {
                    LOG.error("Failed to load lab test from {}: {}", yamlFile, e.getMessage(), e);
                }
            }
        }

        return count;
    }

    /**
     * Load all imaging study definitions from YAML files.
     *
     * @return Number of imaging studies loaded
     */
    private int loadImagingStudies() {
        int count = 0;

        // Subdirectories under imaging (updated to match actual directory structure)
        String[] categories = {"radiology", "cardiac", "ultrasound", "nuclear", "mri"};

        for (String category : categories) {
            String resourcePath = IMAGING_BASE_PATH + "/" + category;
            List<String> yamlFiles = findYamlFiles(resourcePath);

            for (String yamlFile : yamlFiles) {
                try {
                    ImagingStudy imagingStudy = loadImagingStudy(yamlFile);
                    if (imagingStudy != null) {
                        cacheImagingStudy(imagingStudy);
                        count++;
                    }
                } catch (Exception e) {
                    LOG.error("Failed to load imaging study from {}: {}", yamlFile, e.getMessage(), e);
                }
            }
        }

        return count;
    }

    /**
     * Find all YAML files in a resource directory.
     *
     * @param resourcePath Resource path to scan
     * @return List of resource paths to YAML files
     */
    private List<String> findYamlFiles(String resourcePath) {
        List<String> yamlFiles = new ArrayList<>();

        try {
            // Try to get resource as stream to check if directory exists
            InputStream testStream = getClass().getResourceAsStream(resourcePath);
            if (testStream != null) {
                testStream.close();

                // Get all .yaml files in directory
                // Note: In JAR context, we need to list files differently
                // For now, we'll use a predefined list approach since we know the files
                yamlFiles = getKnownYamlFiles(resourcePath);
            } else {
                LOG.warn("Resource path not found: {}", resourcePath);
            }
        } catch (Exception e) {
            LOG.error("Error finding YAML files in {}: {}", resourcePath, e.getMessage());
        }

        return yamlFiles;
    }

    /**
     * Get known YAML files for a resource path.
     * This is a workaround for JAR resource listing limitations.
     *
     * @param resourcePath Resource path
     * @return List of known YAML file paths
     */
    private List<String> getKnownYamlFiles(String resourcePath) {
        List<String> files = new ArrayList<>();

        // Chemistry tests (25 files)
        if (resourcePath.contains("chemistry")) {
            files.add(resourcePath + "/abg-panel.yaml");
            files.add(resourcePath + "/albumin.yaml");
            files.add(resourcePath + "/alkaline-phosphatase.yaml");
            files.add(resourcePath + "/alt.yaml");
            files.add(resourcePath + "/ast.yaml");
            files.add(resourcePath + "/bicarbonate.yaml");
            files.add(resourcePath + "/bilirubin-total.yaml");
            files.add(resourcePath + "/blood-alcohol.yaml");
            files.add(resourcePath + "/bun.yaml");
            files.add(resourcePath + "/calcium.yaml");
            files.add(resourcePath + "/chloride.yaml");
            files.add(resourcePath + "/cortisol.yaml");
            files.add(resourcePath + "/creatinine.yaml");
            files.add(resourcePath + "/crp.yaml");
            files.add(resourcePath + "/esr.yaml");
            files.add(resourcePath + "/ferritin.yaml");
            files.add(resourcePath + "/free-t4.yaml");
            files.add(resourcePath + "/glucose.yaml");
            files.add(resourcePath + "/lactate.yaml");
            files.add(resourcePath + "/magnesium.yaml");
            files.add(resourcePath + "/potassium.yaml");
            files.add(resourcePath + "/procalcitonin.yaml");
            files.add(resourcePath + "/sodium.yaml");
            files.add(resourcePath + "/tsh.yaml");
            files.add(resourcePath + "/urine-drug-screen.yaml");
        }
        // Hematology tests (10 files)
        else if (resourcePath.contains("hematology")) {
            files.add(resourcePath + "/bleeding-time.yaml");
            files.add(resourcePath + "/d-dimer.yaml");
            files.add(resourcePath + "/differential.yaml");
            files.add(resourcePath + "/fibrinogen.yaml");
            files.add(resourcePath + "/hematocrit.yaml");
            files.add(resourcePath + "/hemoglobin.yaml");
            files.add(resourcePath + "/platelets.yaml");
            files.add(resourcePath + "/pt-inr.yaml");
            files.add(resourcePath + "/ptt.yaml");
            files.add(resourcePath + "/wbc.yaml");
        }
        // Microbiology tests (8 files)
        else if (resourcePath.contains("microbiology")) {
            files.add(resourcePath + "/blood-culture-aerobic.yaml");
            files.add(resourcePath + "/blood-culture-anaerobic.yaml");
            files.add(resourcePath + "/csf-culture.yaml");
            files.add(resourcePath + "/rapid-flu-rsv.yaml");
            files.add(resourcePath + "/sputum-culture.yaml");
            files.add(resourcePath + "/stool-culture.yaml");
            files.add(resourcePath + "/urine-culture.yaml");
            files.add(resourcePath + "/wound-culture.yaml");
        }
        // Cardiac markers (5 files)
        else if (resourcePath.contains("cardiac-markers")) {
            files.add(resourcePath + "/bnp.yaml");
            files.add(resourcePath + "/ck-mb.yaml");
            files.add(resourcePath + "/nt-probnp.yaml");
            files.add(resourcePath + "/troponin-i.yaml");
            files.add(resourcePath + "/troponin-t.yaml");
        }
        // Urinalysis tests (2 files)
        else if (resourcePath.contains("urinalysis")) {
            files.add(resourcePath + "/urinalysis.yaml");
            files.add(resourcePath + "/urine-protein-creatinine.yaml");
        }
        // Radiology imaging (5 files)
        else if (resourcePath.contains("radiology")) {
            files.add(resourcePath + "/chest-xray.yaml");
            files.add(resourcePath + "/ct-abdomen-pelvis.yaml");
            files.add(resourcePath + "/ct-chest.yaml");
            files.add(resourcePath + "/ct-head.yaml");
            files.add(resourcePath + "/ct-pulmonary-angiogram.yaml");
        }
        // Cardiac imaging (3 files)
        else if (resourcePath.contains("cardiac")) {
            files.add(resourcePath + "/cardiac-ct-angiography.yaml");
            files.add(resourcePath + "/echocardiogram.yaml");
            files.add(resourcePath + "/stress-echo.yaml");
        }
        // Ultrasound imaging (5 files)
        else if (resourcePath.contains("ultrasound")) {
            files.add(resourcePath + "/abdominal-ultrasound.yaml");
            files.add(resourcePath + "/carotid-doppler.yaml");
            files.add(resourcePath + "/pelvic-ultrasound.yaml");
            files.add(resourcePath + "/renal-ultrasound.yaml");
            files.add(resourcePath + "/venous-doppler-leg.yaml");
        }
        // Nuclear imaging (1 file)
        else if (resourcePath.contains("nuclear")) {
            files.add(resourcePath + "/vq-scan.yaml");
        }
        // MRI imaging (1 file)
        else if (resourcePath.contains("mri")) {
            files.add(resourcePath + "/mri-brain.yaml");
        }

        return files;
    }

    /**
     * Load a single lab test from YAML file.
     *
     * @param resourcePath Path to YAML file
     * @return Parsed LabTest object
     */
    private LabTest loadLabTest(String resourcePath) throws Exception {
        InputStream inputStream = getClass().getResourceAsStream(resourcePath);

        if (inputStream == null) {
            LOG.warn("Lab test YAML not found: {}", resourcePath);
            return null;
        }

        try {
            LabTest labTest = yamlMapper.readValue(inputStream, LabTest.class);
            LOG.debug("Loaded lab test: {} ({})", labTest.getTestName(), labTest.getTestId());
            return labTest;
        } finally {
            inputStream.close();
        }
    }

    /**
     * Load a single imaging study from YAML file.
     *
     * @param resourcePath Path to YAML file
     * @return Parsed ImagingStudy object
     */
    private ImagingStudy loadImagingStudy(String resourcePath) throws Exception {
        InputStream inputStream = getClass().getResourceAsStream(resourcePath);

        if (inputStream == null) {
            LOG.warn("Imaging study YAML not found: {}", resourcePath);
            return null;
        }

        try {
            ImagingStudy imagingStudy = yamlMapper.readValue(inputStream, ImagingStudy.class);
            LOG.debug("Loaded imaging study: {} ({})", imagingStudy.getStudyName(), imagingStudy.getStudyId());
            return imagingStudy;
        } finally {
            inputStream.close();
        }
    }

    /**
     * Cache a lab test in all lookup maps.
     *
     * @param labTest Lab test to cache
     */
    private void cacheLabTest(LabTest labTest) {
        if (labTest == null) return;

        // Cache by test ID
        if (labTest.getTestId() != null) {
            labTestsById.put(labTest.getTestId(), labTest);
        }

        // Cache by LOINC code
        if (labTest.getLoincCode() != null) {
            labTestsByLoinc.put(labTest.getLoincCode(), labTest);
        }
    }

    /**
     * Cache an imaging study in all lookup maps.
     *
     * @param imagingStudy Imaging study to cache
     */
    private void cacheImagingStudy(ImagingStudy imagingStudy) {
        if (imagingStudy == null) return;

        // Cache by study ID
        if (imagingStudy.getStudyId() != null) {
            imagingStudiesById.put(imagingStudy.getStudyId(), imagingStudy);
        }

        // Cache by CPT code
        if (imagingStudy.getCptCode() != null) {
            imagingStudiesByCpt.put(imagingStudy.getCptCode(), imagingStudy);
        }
    }

    // ================================================================
    // PUBLIC API METHODS
    // ================================================================

    /**
     * Get lab test by test ID.
     *
     * @param testId Test identifier (e.g., "LAB-LACTATE-001")
     * @return LabTest object or null if not found
     */
    public LabTest getLabTest(String testId) {
        if (testId == null) return null;

        LabTest labTest = labTestsById.get(testId);
        if (labTest == null) {
            LOG.debug("Lab test not found: {}", testId);
        }
        return labTest;
    }

    /**
     * Get lab test by LOINC code.
     *
     * @param loincCode LOINC code (e.g., "2524-7" for lactate)
     * @return LabTest object or null if not found
     */
    public LabTest getLabTestByLoinc(String loincCode) {
        if (loincCode == null) return null;

        LabTest labTest = labTestsByLoinc.get(loincCode);
        if (labTest == null) {
            LOG.debug("Lab test not found for LOINC: {}", loincCode);
        }
        return labTest;
    }

    /**
     * Get imaging study by study ID.
     *
     * @param studyId Study identifier (e.g., "IMG-CXR-001")
     * @return ImagingStudy object or null if not found
     */
    public ImagingStudy getImagingStudy(String studyId) {
        if (studyId == null) return null;

        ImagingStudy imagingStudy = imagingStudiesById.get(studyId);
        if (imagingStudy == null) {
            LOG.debug("Imaging study not found: {}", studyId);
        }
        return imagingStudy;
    }

    /**
     * Get imaging study by CPT code.
     *
     * @param cptCode CPT code (e.g., "71046" for chest x-ray)
     * @return ImagingStudy object or null if not found
     */
    public ImagingStudy getImagingStudyByCpt(String cptCode) {
        if (cptCode == null) return null;

        ImagingStudy imagingStudy = imagingStudiesByCpt.get(cptCode);
        if (imagingStudy == null) {
            LOG.debug("Imaging study not found for CPT: {}", cptCode);
        }
        return imagingStudy;
    }

    /**
     * Get all lab tests.
     *
     * @return Unmodifiable list of all lab tests
     */
    public List<LabTest> getAllLabTests() {
        return Collections.unmodifiableList(
            new ArrayList<>(labTestsById.values())
        );
    }

    /**
     * Get all imaging studies.
     *
     * @return Unmodifiable list of all imaging studies
     */
    public List<ImagingStudy> getAllImagingStudies() {
        return Collections.unmodifiableList(
            new ArrayList<>(imagingStudiesById.values())
        );
    }

    /**
     * Get lab tests by category.
     *
     * @param category Category (CHEMISTRY, HEMATOLOGY, etc.)
     * @return List of lab tests in that category
     */
    public List<LabTest> getLabTestsByCategory(String category) {
        if (category == null) return Collections.emptyList();

        return labTestsById.values().stream()
            .filter(test -> category.equalsIgnoreCase(test.getCategory()))
            .collect(Collectors.toList());
    }

    /**
     * Get imaging studies by modality.
     *
     * @param studyType Study type (XRAY, CT, MRI, etc.)
     * @return List of imaging studies of that type
     */
    public List<ImagingStudy> getImagingStudiesByType(ImagingStudy.StudyType studyType) {
        if (studyType == null) return Collections.emptyList();

        return imagingStudiesById.values().stream()
            .filter(study -> studyType.equals(study.getStudyType()))
            .collect(Collectors.toList());
    }

    /**
     * Reload all test definitions from YAML files.
     * Clears caches and reloads everything.
     * Thread-safe operation.
     */
    public synchronized void reload() {
        LOG.info("Reloading diagnostic test definitions...");

        // Clear all caches
        labTestsById.clear();
        labTestsByLoinc.clear();
        imagingStudiesById.clear();
        imagingStudiesByCpt.clear();

        // Reload all tests
        loadAllTests();

        LOG.info("Reload complete: {} lab tests, {} imaging studies",
                labTestsById.size(), imagingStudiesById.size());
    }

    /**
     * Get loader statistics.
     *
     * @return Map of statistics
     */
    public Map<String, Object> getStatistics() {
        Map<String, Object> stats = new HashMap<>();
        stats.put("labTestCount", labTestsById.size());
        stats.put("imagingStudyCount", imagingStudiesById.size());
        stats.put("labTestsByLoincCount", labTestsByLoinc.size());
        stats.put("imagingStudiesByCptCount", imagingStudiesByCpt.size());
        return stats;
    }

    /**
     * Check if loader is initialized and ready.
     *
     * @return true if tests are loaded
     */
    public boolean isInitialized() {
        return !labTestsById.isEmpty() || !imagingStudiesById.isEmpty();
    }
}
