package au.vaidshala.cqlruntime.config;

import au.vaidshala.cqlruntime.external.SubstrateClient;
import au.vaidshala.cqlruntime.external.SubstrateExternalFunctions;
import au.vaidshala.cqlruntime.loader.CqlLibraryLoader;
import au.vaidshala.cqlruntime.loader.CqlLibraryRegistry;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.nio.file.Path;
import java.util.Arrays;
import java.util.List;

/**
 * Spring configuration for the substrate-aware beans:
 *
 * <ul>
 *   <li>{@link SubstrateClient} — HTTP client over kb-20 /v2/runtime/*</li>
 *   <li>{@link SubstrateExternalFunctions} — Java-side bodies of the
 *       Vaidshala.Substrate.* CQL external functions</li>
 *   <li>{@link CqlLibraryRegistry} — populated at startup by walking the
 *       configured library roots</li>
 * </ul>
 *
 * <p>Plan 0.5 Task 5 of 8.
 */
@Configuration
public class SubstrateConfig {

    private static final Logger log = LoggerFactory.getLogger(SubstrateConfig.class);

    @Bean
    public SubstrateClient substrateClient(
            @Value("${substrate.base-url}") String baseUrl) {
        log.info("kb-cql-runtime: SubstrateClient -> {}", baseUrl);
        return new SubstrateClient(baseUrl);
    }

    @Bean
    public SubstrateExternalFunctions substrateExternalFunctions(SubstrateClient client) {
        return new SubstrateExternalFunctions(client);
    }

    @Bean
    public CqlLibraryRegistry cqlLibraryRegistry(
            @Value("${cql.library.roots}") String rootsCsv) {
        List<Path> roots = Arrays.stream(rootsCsv.split(","))
            .map(String::trim)
            .filter(s -> !s.isEmpty())
            .map(Path::of)
            .toList();

        CqlLibraryLoader loader = new CqlLibraryLoader(roots);
        CqlLibraryRegistry registry = new CqlLibraryRegistry();
        int count = loader.loadAll(registry);
        log.info("kb-cql-runtime: loaded {} CQL libraries from {}", count, roots);
        return registry;
    }
}
