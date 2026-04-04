package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module11ExerciseBPAnalyzerTest {

    @Test
    void normalBPResponse_riseLessThan60() {
        Module11ExerciseBPAnalyzer.Result result =
                Module11ExerciseBPAnalyzer.analyze(120.0, 160.0, 125.0);
        assertEquals(ExerciseBPResponse.NORMAL, result.bpResponse);
        assertEquals(40.0, result.sbpRise);
        assertEquals(5.0, result.postExerciseDelta); // 125-120
    }

    @Test
    void exaggeratedResponse_riseOver60() {
        Module11ExerciseBPAnalyzer.Result result =
                Module11ExerciseBPAnalyzer.analyze(120.0, 195.0, 130.0);
        assertEquals(ExerciseBPResponse.EXAGGERATED, result.bpResponse);
        assertEquals(75.0, result.sbpRise);
        assertTrue(result.bpResponse.isPrognosticFlag());
    }

    @Test
    void exaggeratedResponse_peakOver210() {
        Module11ExerciseBPAnalyzer.Result result =
                Module11ExerciseBPAnalyzer.analyze(160.0, 215.0, 155.0);
        assertEquals(ExerciseBPResponse.EXAGGERATED, result.bpResponse);
    }

    @Test
    void postExerciseHypotension_beneficial() {
        // Post-exercise SBP drops 10 mmHg below pre (5-20 range → PEH)
        Module11ExerciseBPAnalyzer.Result result =
                Module11ExerciseBPAnalyzer.analyze(140.0, 170.0, 130.0);
        assertEquals(ExerciseBPResponse.POST_EXERCISE_HYPOTENSION, result.bpResponse);
        assertEquals(-10.0, result.postExerciseDelta);
    }

    @Test
    void hypotensiveResponse_dropOver20() {
        Module11ExerciseBPAnalyzer.Result result =
                Module11ExerciseBPAnalyzer.analyze(140.0, 170.0, 115.0);
        assertEquals(ExerciseBPResponse.HYPOTENSIVE_RESPONSE, result.bpResponse);
        assertTrue(result.bpResponse.isPrognosticFlag());
    }

    @Test
    void incomplete_missingPreExercise() {
        Module11ExerciseBPAnalyzer.Result result =
                Module11ExerciseBPAnalyzer.analyze(null, 160.0, 130.0);
        assertEquals(ExerciseBPResponse.INCOMPLETE, result.bpResponse);
    }

    @Test
    void rpp_computedCorrectly() {
        Module11ExerciseBPAnalyzer.Result result =
                Module11ExerciseBPAnalyzer.analyze(120.0, 180.0, 125.0);
        // RPP is computed externally from HR; BP analyzer provides peak SBP
        assertEquals(180.0, result.peakExerciseSBP);
    }
}
