package com.cardiofit.flink.thresholds;

import org.apache.flink.api.common.state.BroadcastState;
import org.apache.flink.api.common.state.MapStateDescriptor;
import org.apache.flink.api.common.state.ReadOnlyBroadcastState;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.api.common.typeinfo.Types;
import org.apache.flink.streaming.api.functions.co.KeyedBroadcastProcessFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;

/**
 * Abstract base class for Flink operators that consume clinical thresholds
 * via BroadcastState.
 *
 * Architecture follows the same pattern as
 * {@link com.cardiofit.flink.operators.Module3_ComprehensiveCDS_WithCDC.CDSProcessorWithCDC}
 * but broadcasts {@link ClinicalThresholdSet} instead of protocol data.
 *
 * Usage:
 * <pre>
 *   // 1. Create a BroadcastStream from a threshold-update Kafka topic or timer source
 *   BroadcastStream&lt;ClinicalThresholdSet&gt; thresholdStream = ...
 *       .broadcast(ThresholdBroadcastFunction.THRESHOLD_STATE);
 *
 *   // 2. Connect your keyed stream and process
 *   patientStream
 *       .keyBy(...)
 *       .connect(thresholdStream)
 *       .process(new MyConcreteFunction());
 * </pre>
 *
 * Subclasses implement {@code processElement} and call
 * {@link #getThresholds(ReadOnlyBroadcastState)} to get the current thresholds
 * with automatic fallback to hardcoded defaults.
 *
 * @param <K>   key type
 * @param <IN>  input element type
 * @param <OUT> output element type
 */
public abstract class ThresholdBroadcastFunction<K, IN, OUT>
        extends KeyedBroadcastProcessFunction<K, IN, ClinicalThresholdSet, OUT>
        implements Serializable {

    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(ThresholdBroadcastFunction.class);

    /** Single key used in the MapState to store the current threshold set. */
    private static final String STATE_KEY = "current";

    /**
     * Shared MapStateDescriptor for clinical thresholds BroadcastState.
     * All operators that consume thresholds must reference this same descriptor
     * so that Flink can correctly route broadcast elements.
     */
    public static final MapStateDescriptor<String, ClinicalThresholdSet> THRESHOLD_STATE =
            new MapStateDescriptor<>(
                    "clinical-thresholds",
                    Types.STRING,
                    TypeInformation.of(ClinicalThresholdSet.class)
            );

    /**
     * Handles incoming threshold broadcast updates. Stores the new threshold set
     * into BroadcastState, making it immediately available to all parallel instances.
     */
    @Override
    public void processBroadcastElement(
            ClinicalThresholdSet thresholds,
            Context ctx,
            Collector<OUT> out) throws Exception {

        BroadcastState<String, ClinicalThresholdSet> state = ctx.getBroadcastState(THRESHOLD_STATE);
        state.put(STATE_KEY, thresholds);

        LOG.info("Clinical thresholds updated in BroadcastState: version={}",
                thresholds != null ? thresholds.getVersion() : "null");
    }

    /**
     * Retrieves the current {@link ClinicalThresholdSet} from BroadcastState
     * with automatic fallback to hardcoded defaults if state is empty.
     *
     * @param broadcastState the read-only broadcast state from processElement's context
     * @return never null -- always returns a valid threshold set
     */
    protected ClinicalThresholdSet getThresholds(
            ReadOnlyBroadcastState<String, ClinicalThresholdSet> broadcastState) throws Exception {

        ClinicalThresholdSet thresholds = broadcastState.get(STATE_KEY);
        if (thresholds != null) {
            return thresholds;
        }

        LOG.debug("BroadcastState empty, using hardcoded defaults");
        return ClinicalThresholdSet.hardcodedDefaults();
    }
}
