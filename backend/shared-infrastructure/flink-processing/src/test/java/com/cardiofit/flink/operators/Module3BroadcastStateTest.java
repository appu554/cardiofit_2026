package com.cardiofit.flink.operators;

import com.cardiofit.flink.cdc.ProtocolCDCEvent;
import com.cardiofit.flink.models.protocol.SimplifiedProtocol;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class Module3BroadcastStateTest {

    @Test
    void protocolStateDescriptor_hasCorrectName() {
        assertEquals("protocol-broadcast-state",
                Module3_ComprehensiveCDS_WithCDC.PROTOCOL_STATE_DESCRIPTOR.getName());
    }

    @Test
    void drugRuleStateDescriptor_exists() {
        assertNotNull(Module3_ComprehensiveCDS_WithCDC.DRUG_RULE_STATE_DESCRIPTOR);
        assertEquals("drug-rule-broadcast-state",
                Module3_ComprehensiveCDS_WithCDC.DRUG_RULE_STATE_DESCRIPTOR.getName());
    }

    @Test
    void drugInteractionStateDescriptor_exists() {
        assertNotNull(Module3_ComprehensiveCDS_WithCDC.DRUG_INTERACTION_STATE_DESCRIPTOR);
        assertEquals("drug-interaction-broadcast-state",
                Module3_ComprehensiveCDS_WithCDC.DRUG_INTERACTION_STATE_DESCRIPTOR.getName());
    }

    @Test
    void terminologyStateDescriptor_exists() {
        assertNotNull(Module3_ComprehensiveCDS_WithCDC.TERMINOLOGY_STATE_DESCRIPTOR);
        assertEquals("terminology-broadcast-state",
                Module3_ComprehensiveCDS_WithCDC.TERMINOLOGY_STATE_DESCRIPTOR.getName());
    }

    @Test
    void convertCDCToProtocol_parsesThresholdsFromContent() throws Exception {
        Module3_ComprehensiveCDS_WithCDC.CDSProcessorWithCDC processor =
                new Module3_ComprehensiveCDS_WithCDC.CDSProcessorWithCDC();

        // Use reflection to call private convertCDCToProtocol
        java.lang.reflect.Method method = processor.getClass()
                .getDeclaredMethod("convertCDCToProtocol", ProtocolCDCEvent.ProtocolData.class);
        method.setAccessible(true);

        ProtocolCDCEvent.ProtocolData data = new ProtocolCDCEvent.ProtocolData();
        data.setId(42);
        data.setProtocolName("Test HTN Protocol");
        data.setVersion("1.0");
        data.setSpecialty("Cardiology");
        data.setCategory("CARDIOLOGY");
        data.setContent("{\"triggerThresholds\":{\"systolicbloodpressure\":140.0,\"diastolicbloodpressure\":90.0}}");

        SimplifiedProtocol result = (SimplifiedProtocol) method.invoke(processor, data);

        assertNotNull(result.getTriggerThresholds());
        assertEquals(140.0, result.getTriggerThresholds().get("systolicbloodpressure"), 0.01);
        assertEquals(90.0, result.getTriggerThresholds().get("diastolicbloodpressure"), 0.01);
    }

    @Test
    void convertCDCToProtocol_fallsBackToDefaults_whenNoThresholdsInContent() throws Exception {
        Module3_ComprehensiveCDS_WithCDC.CDSProcessorWithCDC processor =
                new Module3_ComprehensiveCDS_WithCDC.CDSProcessorWithCDC();

        java.lang.reflect.Method method = processor.getClass()
                .getDeclaredMethod("convertCDCToProtocol", ProtocolCDCEvent.ProtocolData.class);
        method.setAccessible(true);

        ProtocolCDCEvent.ProtocolData data = new ProtocolCDCEvent.ProtocolData();
        data.setId(99);
        data.setProtocolName("Generic Protocol");
        data.setVersion("1.0");
        data.setSpecialty("General");
        data.setContent("Some plain text content without JSON");

        SimplifiedProtocol result = (SimplifiedProtocol) method.invoke(processor, data);

        // No thresholds parsed and no category default → empty map
        assertTrue(result.getTriggerThresholds() == null || result.getTriggerThresholds().isEmpty());
    }
}
