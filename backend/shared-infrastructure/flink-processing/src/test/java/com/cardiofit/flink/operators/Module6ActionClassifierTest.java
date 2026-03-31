package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module6ActionClassifierTest {

    @Test
    void clinicalEvent_fromCDSEvent_extractsNews2Score() {
        CDSEvent cds = new CDSEvent();
        PatientContextState state = new PatientContextState("P001");
        state.setNews2Score(13);
        state.setQsofaScore(2);
        cds.setPatientId("P001");
        cds.setPatientState(state);

        ClinicalEvent event = ClinicalEvent.fromCDS(cds);

        assertEquals("P001", event.getPatientId());
        assertEquals(13, event.getNews2Score());
        assertEquals(2, event.getQsofaScore());
        assertEquals(ClinicalEvent.Source.CDS, event.getSource());
    }
}
