package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.EnrichedPatientContext;
import org.apache.flink.util.OutputTag;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class Module3DLQTest {

    @Test
    void dlqOutputTag_exists() {
        OutputTag<EnrichedPatientContext> tag = Module3_ComprehensiveCDS_WithCDC.DLQ_OUTPUT_TAG;
        assertNotNull(tag);
        assertEquals("dlq-cds-events", tag.getId());
    }

    @Test
    void dlqOutputTag_typeIsEnrichedPatientContext() {
        OutputTag<EnrichedPatientContext> tag = Module3_ComprehensiveCDS_WithCDC.DLQ_OUTPUT_TAG;
        assertNotNull(tag.getTypeInfo());
    }
}
