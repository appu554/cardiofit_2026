from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SafetyStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SAFETY_STATUS_UNSPECIFIED: _ClassVar[SafetyStatus]
    SAFETY_STATUS_SAFE: _ClassVar[SafetyStatus]
    SAFETY_STATUS_UNSAFE: _ClassVar[SafetyStatus]
    SAFETY_STATUS_WARNING: _ClassVar[SafetyStatus]
    SAFETY_STATUS_MANUAL_REVIEW: _ClassVar[SafetyStatus]
    SAFETY_STATUS_ERROR: _ClassVar[SafetyStatus]

class ExplanationLevel(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EXPLANATION_LEVEL_UNSPECIFIED: _ClassVar[ExplanationLevel]
    EXPLANATION_LEVEL_BASIC: _ClassVar[ExplanationLevel]
    EXPLANATION_LEVEL_DETAILED: _ClassVar[ExplanationLevel]
    EXPLANATION_LEVEL_EXPERT: _ClassVar[ExplanationLevel]

class OverrideLevel(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OVERRIDE_LEVEL_UNSPECIFIED: _ClassVar[OverrideLevel]
    OVERRIDE_LEVEL_RESIDENT: _ClassVar[OverrideLevel]
    OVERRIDE_LEVEL_ATTENDING: _ClassVar[OverrideLevel]
    OVERRIDE_LEVEL_PHARMACIST: _ClassVar[OverrideLevel]
    OVERRIDE_LEVEL_CHIEF: _ClassVar[OverrideLevel]

class EngineStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ENGINE_STATUS_UNSPECIFIED: _ClassVar[EngineStatus]
    ENGINE_STATUS_HEALTHY: _ClassVar[EngineStatus]
    ENGINE_STATUS_DEGRADED: _ClassVar[EngineStatus]
    ENGINE_STATUS_UNHEALTHY: _ClassVar[EngineStatus]

class HealthStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    HEALTH_STATUS_UNSPECIFIED: _ClassVar[HealthStatus]
    HEALTH_STATUS_HEALTHY: _ClassVar[HealthStatus]
    HEALTH_STATUS_DEGRADED: _ClassVar[HealthStatus]
    HEALTH_STATUS_UNHEALTHY: _ClassVar[HealthStatus]
SAFETY_STATUS_UNSPECIFIED: SafetyStatus
SAFETY_STATUS_SAFE: SafetyStatus
SAFETY_STATUS_UNSAFE: SafetyStatus
SAFETY_STATUS_WARNING: SafetyStatus
SAFETY_STATUS_MANUAL_REVIEW: SafetyStatus
SAFETY_STATUS_ERROR: SafetyStatus
EXPLANATION_LEVEL_UNSPECIFIED: ExplanationLevel
EXPLANATION_LEVEL_BASIC: ExplanationLevel
EXPLANATION_LEVEL_DETAILED: ExplanationLevel
EXPLANATION_LEVEL_EXPERT: ExplanationLevel
OVERRIDE_LEVEL_UNSPECIFIED: OverrideLevel
OVERRIDE_LEVEL_RESIDENT: OverrideLevel
OVERRIDE_LEVEL_ATTENDING: OverrideLevel
OVERRIDE_LEVEL_PHARMACIST: OverrideLevel
OVERRIDE_LEVEL_CHIEF: OverrideLevel
ENGINE_STATUS_UNSPECIFIED: EngineStatus
ENGINE_STATUS_HEALTHY: EngineStatus
ENGINE_STATUS_DEGRADED: EngineStatus
ENGINE_STATUS_UNHEALTHY: EngineStatus
HEALTH_STATUS_UNSPECIFIED: HealthStatus
HEALTH_STATUS_HEALTHY: HealthStatus
HEALTH_STATUS_DEGRADED: HealthStatus
HEALTH_STATUS_UNHEALTHY: HealthStatus

class SafetyRequest(_message.Message):
    __slots__ = ("request_id", "patient_id", "clinician_id", "action_type", "priority", "medication_ids", "condition_ids", "allergy_ids", "context", "timestamp", "source")
    class ContextEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    PATIENT_ID_FIELD_NUMBER: _ClassVar[int]
    CLINICIAN_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_TYPE_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    MEDICATION_IDS_FIELD_NUMBER: _ClassVar[int]
    CONDITION_IDS_FIELD_NUMBER: _ClassVar[int]
    ALLERGY_IDS_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    patient_id: str
    clinician_id: str
    action_type: str
    priority: str
    medication_ids: _containers.RepeatedScalarFieldContainer[str]
    condition_ids: _containers.RepeatedScalarFieldContainer[str]
    allergy_ids: _containers.RepeatedScalarFieldContainer[str]
    context: _containers.ScalarMap[str, str]
    timestamp: _timestamp_pb2.Timestamp
    source: str
    def __init__(self, request_id: _Optional[str] = ..., patient_id: _Optional[str] = ..., clinician_id: _Optional[str] = ..., action_type: _Optional[str] = ..., priority: _Optional[str] = ..., medication_ids: _Optional[_Iterable[str]] = ..., condition_ids: _Optional[_Iterable[str]] = ..., allergy_ids: _Optional[_Iterable[str]] = ..., context: _Optional[_Mapping[str, str]] = ..., timestamp: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., source: _Optional[str] = ...) -> None: ...

class SafetyResponse(_message.Message):
    __slots__ = ("request_id", "status", "risk_score", "critical_violations", "warnings", "engine_results", "engines_failed", "explanation", "override_token", "processing_time_ms", "context_version", "timestamp", "metadata")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RISK_SCORE_FIELD_NUMBER: _ClassVar[int]
    CRITICAL_VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    ENGINE_RESULTS_FIELD_NUMBER: _ClassVar[int]
    ENGINES_FAILED_FIELD_NUMBER: _ClassVar[int]
    EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    PROCESSING_TIME_MS_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_VERSION_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    status: SafetyStatus
    risk_score: float
    critical_violations: _containers.RepeatedScalarFieldContainer[str]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    engine_results: _containers.RepeatedCompositeFieldContainer[EngineResult]
    engines_failed: _containers.RepeatedScalarFieldContainer[str]
    explanation: Explanation
    override_token: OverrideToken
    processing_time_ms: int
    context_version: str
    timestamp: _timestamp_pb2.Timestamp
    metadata: _containers.ScalarMap[str, str]
    def __init__(self, request_id: _Optional[str] = ..., status: _Optional[_Union[SafetyStatus, str]] = ..., risk_score: _Optional[float] = ..., critical_violations: _Optional[_Iterable[str]] = ..., warnings: _Optional[_Iterable[str]] = ..., engine_results: _Optional[_Iterable[_Union[EngineResult, _Mapping]]] = ..., engines_failed: _Optional[_Iterable[str]] = ..., explanation: _Optional[_Union[Explanation, _Mapping]] = ..., override_token: _Optional[_Union[OverrideToken, _Mapping]] = ..., processing_time_ms: _Optional[int] = ..., context_version: _Optional[str] = ..., timestamp: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...

class EngineResult(_message.Message):
    __slots__ = ("engine_id", "engine_name", "status", "risk_score", "violations", "warnings", "confidence", "duration_ms", "tier", "error", "metadata")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ENGINE_ID_FIELD_NUMBER: _ClassVar[int]
    ENGINE_NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RISK_SCORE_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    engine_id: str
    engine_name: str
    status: SafetyStatus
    risk_score: float
    violations: _containers.RepeatedScalarFieldContainer[str]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    confidence: float
    duration_ms: int
    tier: int
    error: str
    metadata: _containers.ScalarMap[str, str]
    def __init__(self, engine_id: _Optional[str] = ..., engine_name: _Optional[str] = ..., status: _Optional[_Union[SafetyStatus, str]] = ..., risk_score: _Optional[float] = ..., violations: _Optional[_Iterable[str]] = ..., warnings: _Optional[_Iterable[str]] = ..., confidence: _Optional[float] = ..., duration_ms: _Optional[int] = ..., tier: _Optional[int] = ..., error: _Optional[str] = ..., metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...

class Explanation(_message.Message):
    __slots__ = ("level", "summary", "details", "confidence", "evidence", "actionable", "generated_at")
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    ACTIONABLE_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    level: ExplanationLevel
    summary: str
    details: _containers.RepeatedCompositeFieldContainer[ExplanationDetail]
    confidence: float
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    actionable: _containers.RepeatedCompositeFieldContainer[ActionableGuidance]
    generated_at: _timestamp_pb2.Timestamp
    def __init__(self, level: _Optional[_Union[ExplanationLevel, str]] = ..., summary: _Optional[str] = ..., details: _Optional[_Iterable[_Union[ExplanationDetail, _Mapping]]] = ..., confidence: _Optional[float] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ..., actionable: _Optional[_Iterable[_Union[ActionableGuidance, _Mapping]]] = ..., generated_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ExplanationDetail(_message.Message):
    __slots__ = ("category", "severity", "description", "clinical_rationale", "confidence", "engine_source", "recommended_action")
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CLINICAL_RATIONALE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    ENGINE_SOURCE_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_ACTION_FIELD_NUMBER: _ClassVar[int]
    category: str
    severity: str
    description: str
    clinical_rationale: str
    confidence: float
    engine_source: str
    recommended_action: str
    def __init__(self, category: _Optional[str] = ..., severity: _Optional[str] = ..., description: _Optional[str] = ..., clinical_rationale: _Optional[str] = ..., confidence: _Optional[float] = ..., engine_source: _Optional[str] = ..., recommended_action: _Optional[str] = ...) -> None: ...

class Evidence(_message.Message):
    __slots__ = ("type", "source", "description", "strength", "url", "metadata")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STRENGTH_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    type: str
    source: str
    description: str
    strength: str
    url: str
    metadata: _containers.ScalarMap[str, str]
    def __init__(self, type: _Optional[str] = ..., source: _Optional[str] = ..., description: _Optional[str] = ..., strength: _Optional[str] = ..., url: _Optional[str] = ..., metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...

class ActionableGuidance(_message.Message):
    __slots__ = ("action", "priority", "steps", "monitoring", "timeline", "responsible")
    ACTION_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    MONITORING_FIELD_NUMBER: _ClassVar[int]
    TIMELINE_FIELD_NUMBER: _ClassVar[int]
    RESPONSIBLE_FIELD_NUMBER: _ClassVar[int]
    action: str
    priority: str
    steps: _containers.RepeatedScalarFieldContainer[str]
    monitoring: _containers.RepeatedScalarFieldContainer[str]
    timeline: str
    responsible: str
    def __init__(self, action: _Optional[str] = ..., priority: _Optional[str] = ..., steps: _Optional[_Iterable[str]] = ..., monitoring: _Optional[_Iterable[str]] = ..., timeline: _Optional[str] = ..., responsible: _Optional[str] = ...) -> None: ...

class OverrideToken(_message.Message):
    __slots__ = ("token_id", "request_id", "patient_id", "decision_summary", "required_level", "expires_at", "context_hash", "created_at", "signature")
    TOKEN_ID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    PATIENT_ID_FIELD_NUMBER: _ClassVar[int]
    DECISION_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_LEVEL_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_HASH_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    token_id: str
    request_id: str
    patient_id: str
    decision_summary: DecisionSummary
    required_level: OverrideLevel
    expires_at: _timestamp_pb2.Timestamp
    context_hash: str
    created_at: _timestamp_pb2.Timestamp
    signature: str
    def __init__(self, token_id: _Optional[str] = ..., request_id: _Optional[str] = ..., patient_id: _Optional[str] = ..., decision_summary: _Optional[_Union[DecisionSummary, _Mapping]] = ..., required_level: _Optional[_Union[OverrideLevel, str]] = ..., expires_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., context_hash: _Optional[str] = ..., created_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., signature: _Optional[str] = ...) -> None: ...

class DecisionSummary(_message.Message):
    __slots__ = ("status", "critical_violations", "engines_failed", "risk_score", "explanation")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CRITICAL_VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    ENGINES_FAILED_FIELD_NUMBER: _ClassVar[int]
    RISK_SCORE_FIELD_NUMBER: _ClassVar[int]
    EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    status: SafetyStatus
    critical_violations: _containers.RepeatedScalarFieldContainer[str]
    engines_failed: _containers.RepeatedScalarFieldContainer[str]
    risk_score: float
    explanation: str
    def __init__(self, status: _Optional[_Union[SafetyStatus, str]] = ..., critical_violations: _Optional[_Iterable[str]] = ..., engines_failed: _Optional[_Iterable[str]] = ..., risk_score: _Optional[float] = ..., explanation: _Optional[str] = ...) -> None: ...

class EngineStatusRequest(_message.Message):
    __slots__ = ("engine_id",)
    ENGINE_ID_FIELD_NUMBER: _ClassVar[int]
    engine_id: str
    def __init__(self, engine_id: _Optional[str] = ...) -> None: ...

class EngineStatusResponse(_message.Message):
    __slots__ = ("engines", "metadata")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ENGINES_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    engines: _containers.RepeatedCompositeFieldContainer[EngineInfo]
    metadata: _containers.ScalarMap[str, str]
    def __init__(self, engines: _Optional[_Iterable[_Union[EngineInfo, _Mapping]]] = ..., metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...

class EngineInfo(_message.Message):
    __slots__ = ("id", "name", "capabilities", "tier", "priority", "timeout_ms", "status", "last_check", "failure_count", "metadata")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LAST_CHECK_FIELD_NUMBER: _ClassVar[int]
    FAILURE_COUNT_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    tier: int
    priority: int
    timeout_ms: int
    status: EngineStatus
    last_check: _timestamp_pb2.Timestamp
    failure_count: int
    metadata: _containers.ScalarMap[str, str]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ..., tier: _Optional[int] = ..., priority: _Optional[int] = ..., timeout_ms: _Optional[int] = ..., status: _Optional[_Union[EngineStatus, str]] = ..., last_check: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ..., failure_count: _Optional[int] = ..., metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...

class OverrideRequest(_message.Message):
    __slots__ = ("token_id", "clinician_id", "reason")
    TOKEN_ID_FIELD_NUMBER: _ClassVar[int]
    CLINICIAN_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    token_id: str
    clinician_id: str
    reason: str
    def __init__(self, token_id: _Optional[str] = ..., clinician_id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class OverrideResponse(_message.Message):
    __slots__ = ("valid", "reason", "token", "clinician_id", "validated_at")
    VALID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    CLINICIAN_ID_FIELD_NUMBER: _ClassVar[int]
    VALIDATED_AT_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    reason: str
    token: OverrideToken
    clinician_id: str
    validated_at: _timestamp_pb2.Timestamp
    def __init__(self, valid: bool = ..., reason: _Optional[str] = ..., token: _Optional[_Union[OverrideToken, _Mapping]] = ..., clinician_id: _Optional[str] = ..., validated_at: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class HealthRequest(_message.Message):
    __slots__ = ("detailed",)
    DETAILED_FIELD_NUMBER: _ClassVar[int]
    detailed: bool
    def __init__(self, detailed: bool = ...) -> None: ...

class HealthResponse(_message.Message):
    __slots__ = ("status", "message", "details", "timestamp")
    class DetailsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    status: HealthStatus
    message: str
    details: _containers.ScalarMap[str, str]
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, status: _Optional[_Union[HealthStatus, str]] = ..., message: _Optional[str] = ..., details: _Optional[_Mapping[str, str]] = ..., timestamp: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
