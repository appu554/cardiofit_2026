package com.cardiofit.flink.operators;

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
}
