// Package models defines data structures for KB-12 Order Sets & Care Plans
package models

import "time"

// FHIR R4 Resource Structures for Order Set Output

// FHIRBundle represents a FHIR Bundle resource
type FHIRBundle struct {
	ResourceType string        `json:"resourceType"`
	ID           string        `json:"id,omitempty"`
	Type         string        `json:"type"` // collection, transaction, batch
	Timestamp    time.Time     `json:"timestamp,omitempty"`
	Total        int           `json:"total,omitempty"`
	Entry        []BundleEntry `json:"entry,omitempty"`
}

// BundleEntry represents an entry in a FHIR Bundle
type BundleEntry struct {
	FullURL  string       `json:"fullUrl,omitempty"`
	Resource interface{}  `json:"resource"`
	Request  *BundleRequest `json:"request,omitempty"`
}

// BundleRequest represents the request details for a bundle entry
type BundleRequest struct {
	Method string `json:"method"` // GET, POST, PUT, DELETE
	URL    string `json:"url"`
}

// FHIRMedicationRequest represents a FHIR MedicationRequest resource
type FHIRMedicationRequest struct {
	ResourceType         string              `json:"resourceType"`
	ID                   string              `json:"id,omitempty"`
	Status               string              `json:"status"` // active, on-hold, cancelled, completed, entered-in-error, stopped, draft, unknown
	Intent               string              `json:"intent"` // proposal, plan, order, original-order, reflex-order, filler-order, instance-order, option
	Priority             string              `json:"priority,omitempty"` // routine, urgent, asap, stat
	MedicationCodeableConcept *CodeableConcept `json:"medicationCodeableConcept,omitempty"`
	Subject              Reference           `json:"subject"`
	Encounter            *Reference          `json:"encounter,omitempty"`
	AuthoredOn           time.Time           `json:"authoredOn,omitempty"`
	Requester            *Reference          `json:"requester,omitempty"`
	DosageInstruction    []DosageInstruction `json:"dosageInstruction,omitempty"`
	DispenseRequest      *DispenseRequest    `json:"dispenseRequest,omitempty"`
	Note                 []Annotation        `json:"note,omitempty"`
}

// FHIRServiceRequest represents a FHIR ServiceRequest resource
type FHIRServiceRequest struct {
	ResourceType    string           `json:"resourceType"`
	ID              string           `json:"id,omitempty"`
	Status          string           `json:"status"` // draft, active, on-hold, revoked, completed, entered-in-error, unknown
	Intent          string           `json:"intent"` // proposal, plan, directive, order, original-order, reflex-order, filler-order, instance-order, option
	Priority        string           `json:"priority,omitempty"` // routine, urgent, asap, stat
	Category        []CodeableConcept `json:"category,omitempty"`
	Code            *CodeableConcept `json:"code,omitempty"`
	Subject         Reference        `json:"subject"`
	Encounter       *Reference       `json:"encounter,omitempty"`
	AuthoredOn      time.Time        `json:"authoredOn,omitempty"`
	Requester       *Reference       `json:"requester,omitempty"`
	PerformerType   *CodeableConcept `json:"performerType,omitempty"`
	Performer       []Reference      `json:"performer,omitempty"`
	ReasonCode      []CodeableConcept `json:"reasonCode,omitempty"`
	BodySite        []CodeableConcept `json:"bodySite,omitempty"`
	Note            []Annotation     `json:"note,omitempty"`
}

// FHIRCarePlan represents a FHIR CarePlan resource
type FHIRCarePlan struct {
	ResourceType   string           `json:"resourceType"`
	ID             string           `json:"id,omitempty"`
	Status         string           `json:"status"` // draft, active, on-hold, revoked, completed, entered-in-error, unknown
	Intent         string           `json:"intent"` // proposal, plan, order, option
	Category       []CodeableConcept `json:"category,omitempty"`
	Title          string           `json:"title,omitempty"`
	Description    string           `json:"description,omitempty"`
	Subject        Reference        `json:"subject"`
	Encounter      *Reference       `json:"encounter,omitempty"`
	Period         *Period          `json:"period,omitempty"`
	Created        time.Time        `json:"created,omitempty"`
	Author         *Reference       `json:"author,omitempty"`
	Contributor    []Reference      `json:"contributor,omitempty"`
	CareTeam       []Reference      `json:"careTeam,omitempty"`
	Addresses      []CodeableConcept `json:"addresses,omitempty"`
	Goal           []Reference      `json:"goal,omitempty"`
	Activity       []CarePlanActivity `json:"activity,omitempty"`
	Note           []Annotation     `json:"note,omitempty"`
}

// CarePlanActivity represents an activity in a FHIR CarePlan
type CarePlanActivity struct {
	OutcomeCodeableConcept []CodeableConcept         `json:"outcomeCodeableConcept,omitempty"`
	OutcomeReference       []Reference               `json:"outcomeReference,omitempty"`
	Progress               []Annotation              `json:"progress,omitempty"`
	Reference              *Reference                `json:"reference,omitempty"`
	Detail                 *CarePlanActivityDetail   `json:"detail,omitempty"`
}

// CarePlanActivityDetail represents the detail of a care plan activity
type CarePlanActivityDetail struct {
	Kind                   string           `json:"kind,omitempty"` // Appointment, CommunicationRequest, DeviceRequest, MedicationRequest, etc.
	InstantiatesCanonical  []string         `json:"instantiatesCanonical,omitempty"`
	Code                   *CodeableConcept `json:"code,omitempty"`
	ReasonCode             []CodeableConcept `json:"reasonCode,omitempty"`
	Goal                   []Reference      `json:"goal,omitempty"`
	Status                 string           `json:"status"` // not-started, scheduled, in-progress, on-hold, completed, cancelled, stopped, unknown, entered-in-error
	StatusReason           *CodeableConcept `json:"statusReason,omitempty"`
	DoNotPerform           bool             `json:"doNotPerform,omitempty"`
	ScheduledTiming        *Timing          `json:"scheduledTiming,omitempty"`
	ScheduledPeriod        *Period          `json:"scheduledPeriod,omitempty"`
	ScheduledString        string           `json:"scheduledString,omitempty"`
	Location               *Reference       `json:"location,omitempty"`
	Performer              []Reference      `json:"performer,omitempty"`
	ProductCodeableConcept *CodeableConcept `json:"productCodeableConcept,omitempty"`
	DailyAmount            *Quantity        `json:"dailyAmount,omitempty"`
	Quantity               *Quantity        `json:"quantity,omitempty"`
	Description            string           `json:"description,omitempty"`
}

// FHIRTask represents a FHIR Task resource
type FHIRTask struct {
	ResourceType   string           `json:"resourceType"`
	ID             string           `json:"id,omitempty"`
	Status         string           `json:"status"` // draft, requested, received, accepted, rejected, ready, cancelled, in-progress, on-hold, failed, completed, entered-in-error
	Intent         string           `json:"intent"` // unknown, proposal, plan, order, original-order, reflex-order, filler-order, instance-order, option
	Priority       string           `json:"priority,omitempty"` // routine, urgent, asap, stat
	Code           *CodeableConcept `json:"code,omitempty"`
	Description    string           `json:"description,omitempty"`
	Focus          *Reference       `json:"focus,omitempty"`
	For            *Reference       `json:"for,omitempty"`
	Encounter      *Reference       `json:"encounter,omitempty"`
	ExecutionPeriod *Period         `json:"executionPeriod,omitempty"`
	AuthoredOn     time.Time        `json:"authoredOn,omitempty"`
	LastModified   time.Time        `json:"lastModified,omitempty"`
	Requester      *Reference       `json:"requester,omitempty"`
	Owner          *Reference       `json:"owner,omitempty"`
	ReasonCode     *CodeableConcept `json:"reasonCode,omitempty"`
	Note           []Annotation     `json:"note,omitempty"`
	Restriction    *TaskRestriction `json:"restriction,omitempty"`
	Input          []TaskInput      `json:"input,omitempty"`
	Output         []TaskOutput     `json:"output,omitempty"`
}

// TaskRestriction represents restrictions on a FHIR Task
type TaskRestriction struct {
	Repetitions int        `json:"repetitions,omitempty"`
	Period      *Period    `json:"period,omitempty"`
	Recipient   []Reference `json:"recipient,omitempty"`
}

// TaskInput represents input to a FHIR Task
type TaskInput struct {
	Type  CodeableConcept `json:"type"`
	Value interface{}     `json:"valueString,omitempty"`
}

// TaskOutput represents output from a FHIR Task
type TaskOutput struct {
	Type  CodeableConcept `json:"type"`
	Value interface{}     `json:"valueString,omitempty"`
}

// FHIRPlanDefinition represents a FHIR PlanDefinition resource
type FHIRPlanDefinition struct {
	ResourceType string           `json:"resourceType"`
	ID           string           `json:"id,omitempty"`
	URL          string           `json:"url,omitempty"`
	Version      string           `json:"version,omitempty"`
	Name         string           `json:"name,omitempty"`
	Title        string           `json:"title,omitempty"`
	Status       string           `json:"status"` // draft, active, retired, unknown
	Experimental bool             `json:"experimental,omitempty"`
	Date         time.Time        `json:"date,omitempty"`
	Publisher    string           `json:"publisher,omitempty"`
	Description  string           `json:"description,omitempty"`
	UseContext   []UsageContext   `json:"useContext,omitempty"`
	Purpose      string           `json:"purpose,omitempty"`
	Goal         []PlanDefinitionGoal `json:"goal,omitempty"`
	Action       []PlanDefinitionAction `json:"action,omitempty"`
}

// PlanDefinitionGoal represents a goal in a PlanDefinition
type PlanDefinitionGoal struct {
	Category    *CodeableConcept `json:"category,omitempty"`
	Description CodeableConcept  `json:"description"`
	Priority    *CodeableConcept `json:"priority,omitempty"`
	Start       *CodeableConcept `json:"start,omitempty"`
	Addresses   []CodeableConcept `json:"addresses,omitempty"`
	Target      []GoalTargetFHIR `json:"target,omitempty"`
}

// GoalTargetFHIR represents a target in a FHIR goal
type GoalTargetFHIR struct {
	Measure      *CodeableConcept `json:"measure,omitempty"`
	DetailQuantity *Quantity       `json:"detailQuantity,omitempty"`
	DetailRange    *Range          `json:"detailRange,omitempty"`
	DetailString   string          `json:"detailString,omitempty"`
	Due            *Duration       `json:"due,omitempty"`
}

// PlanDefinitionAction represents an action in a PlanDefinition
type PlanDefinitionAction struct {
	Prefix           string           `json:"prefix,omitempty"`
	Title            string           `json:"title,omitempty"`
	Description      string           `json:"description,omitempty"`
	TextEquivalent   string           `json:"textEquivalent,omitempty"`
	Priority         string           `json:"priority,omitempty"`
	Code             []CodeableConcept `json:"code,omitempty"`
	Reason           []CodeableConcept `json:"reason,omitempty"`
	Trigger          []TriggerDefinition `json:"trigger,omitempty"`
	Condition        []ActionCondition `json:"condition,omitempty"`
	Input            []DataRequirement `json:"input,omitempty"`
	Output           []DataRequirement `json:"output,omitempty"`
	RelatedAction    []RelatedAction  `json:"relatedAction,omitempty"`
	TimingTiming     *Timing          `json:"timingTiming,omitempty"`
	Participant      []ActionParticipant `json:"participant,omitempty"`
	Type             *CodeableConcept `json:"type,omitempty"`
	GroupingBehavior string           `json:"groupingBehavior,omitempty"`
	SelectionBehavior string          `json:"selectionBehavior,omitempty"`
	RequiredBehavior string           `json:"requiredBehavior,omitempty"`
	PrecheckBehavior string           `json:"precheckBehavior,omitempty"`
	CardinalityBehavior string        `json:"cardinalityBehavior,omitempty"`
	DefinitionCanonical string        `json:"definitionCanonical,omitempty"`
	Action           []PlanDefinitionAction `json:"action,omitempty"`
}

// Common FHIR data types

// CodeableConcept represents a FHIR CodeableConcept
type CodeableConcept struct {
	Coding []Coding `json:"coding,omitempty"`
	Text   string   `json:"text,omitempty"`
}

// Coding represents a FHIR Coding
type Coding struct {
	System  string `json:"system,omitempty"`
	Version string `json:"version,omitempty"`
	Code    string `json:"code,omitempty"`
	Display string `json:"display,omitempty"`
}

// Reference represents a FHIR Reference
type Reference struct {
	Reference string `json:"reference,omitempty"`
	Type      string `json:"type,omitempty"`
	Display   string `json:"display,omitempty"`
}

// Period represents a FHIR Period
type Period struct {
	Start time.Time `json:"start,omitempty"`
	End   time.Time `json:"end,omitempty"`
}

// Quantity represents a FHIR Quantity
type Quantity struct {
	Value  float64 `json:"value,omitempty"`
	Unit   string  `json:"unit,omitempty"`
	System string  `json:"system,omitempty"`
	Code   string  `json:"code,omitempty"`
}

// Range represents a FHIR Range
type Range struct {
	Low  *Quantity `json:"low,omitempty"`
	High *Quantity `json:"high,omitempty"`
}

// Duration represents a FHIR Duration
type Duration struct {
	Value  float64 `json:"value,omitempty"`
	Unit   string  `json:"unit,omitempty"`
	System string  `json:"system,omitempty"`
	Code   string  `json:"code,omitempty"`
}

// Timing represents a FHIR Timing
type Timing struct {
	Event  []time.Time   `json:"event,omitempty"`
	Repeat *TimingRepeat `json:"repeat,omitempty"`
	Code   *CodeableConcept `json:"code,omitempty"`
}

// TimingRepeat represents the repeat component of a Timing
type TimingRepeat struct {
	BoundsDuration *Duration `json:"boundsDuration,omitempty"`
	BoundsPeriod   *Period   `json:"boundsPeriod,omitempty"`
	Count          int       `json:"count,omitempty"`
	CountMax       int       `json:"countMax,omitempty"`
	Duration       float64   `json:"duration,omitempty"`
	DurationMax    float64   `json:"durationMax,omitempty"`
	DurationUnit   string    `json:"durationUnit,omitempty"`
	Frequency      int       `json:"frequency,omitempty"`
	FrequencyMax   int       `json:"frequencyMax,omitempty"`
	Period         float64   `json:"period,omitempty"`
	PeriodMax      float64   `json:"periodMax,omitempty"`
	PeriodUnit     string    `json:"periodUnit,omitempty"`
	DayOfWeek      []string  `json:"dayOfWeek,omitempty"`
	TimeOfDay      []string  `json:"timeOfDay,omitempty"`
	When           []string  `json:"when,omitempty"`
	Offset         int       `json:"offset,omitempty"`
}

// DosageInstruction represents a FHIR Dosage
type DosageInstruction struct {
	Sequence           int              `json:"sequence,omitempty"`
	Text               string           `json:"text,omitempty"`
	AdditionalInstruction []CodeableConcept `json:"additionalInstruction,omitempty"`
	PatientInstruction string           `json:"patientInstruction,omitempty"`
	Timing             *Timing          `json:"timing,omitempty"`
	AsNeededBoolean    bool             `json:"asNeededBoolean,omitempty"`
	AsNeededCodeableConcept *CodeableConcept `json:"asNeededCodeableConcept,omitempty"`
	Site               *CodeableConcept `json:"site,omitempty"`
	Route              *CodeableConcept `json:"route,omitempty"`
	Method             *CodeableConcept `json:"method,omitempty"`
	DoseAndRate        []DoseAndRate    `json:"doseAndRate,omitempty"`
	MaxDosePerPeriod   *Ratio           `json:"maxDosePerPeriod,omitempty"`
	MaxDosePerAdministration *Quantity  `json:"maxDosePerAdministration,omitempty"`
	MaxDosePerLifetime *Quantity        `json:"maxDosePerLifetime,omitempty"`
}

// DoseAndRate represents dose and rate in a dosage
type DoseAndRate struct {
	Type         *CodeableConcept `json:"type,omitempty"`
	DoseQuantity *Quantity        `json:"doseQuantity,omitempty"`
	DoseRange    *Range           `json:"doseRange,omitempty"`
	RateQuantity *Quantity        `json:"rateQuantity,omitempty"`
	RateRange    *Range           `json:"rateRange,omitempty"`
	RateRatio    *Ratio           `json:"rateRatio,omitempty"`
}

// Ratio represents a FHIR Ratio
type Ratio struct {
	Numerator   *Quantity `json:"numerator,omitempty"`
	Denominator *Quantity `json:"denominator,omitempty"`
}

// DispenseRequest represents dispense details in MedicationRequest
type DispenseRequest struct {
	InitialFill           *InitialFill `json:"initialFill,omitempty"`
	DispenseInterval      *Duration    `json:"dispenseInterval,omitempty"`
	ValidityPeriod        *Period      `json:"validityPeriod,omitempty"`
	NumberOfRepeatsAllowed int         `json:"numberOfRepeatsAllowed,omitempty"`
	Quantity              *Quantity    `json:"quantity,omitempty"`
	ExpectedSupplyDuration *Duration   `json:"expectedSupplyDuration,omitempty"`
	Performer             *Reference   `json:"performer,omitempty"`
}

// InitialFill represents initial fill details
type InitialFill struct {
	Quantity *Quantity `json:"quantity,omitempty"`
	Duration *Duration `json:"duration,omitempty"`
}

// Annotation represents a FHIR Annotation
type Annotation struct {
	AuthorReference *Reference `json:"authorReference,omitempty"`
	AuthorString    string     `json:"authorString,omitempty"`
	Time            time.Time  `json:"time,omitempty"`
	Text            string     `json:"text"`
}

// UsageContext represents a FHIR UsageContext
type UsageContext struct {
	Code                 Coding           `json:"code"`
	ValueCodeableConcept *CodeableConcept `json:"valueCodeableConcept,omitempty"`
	ValueQuantity        *Quantity        `json:"valueQuantity,omitempty"`
	ValueRange           *Range           `json:"valueRange,omitempty"`
	ValueReference       *Reference       `json:"valueReference,omitempty"`
}

// TriggerDefinition represents a FHIR TriggerDefinition
type TriggerDefinition struct {
	Type      string           `json:"type"` // named-event, periodic, data-changed, data-added, data-modified, data-removed, data-accessed, data-access-ended
	Name      string           `json:"name,omitempty"`
	TimingTiming *Timing       `json:"timingTiming,omitempty"`
	Data      []DataRequirement `json:"data,omitempty"`
	Condition *Expression      `json:"condition,omitempty"`
}

// ActionCondition represents a condition for an action
type ActionCondition struct {
	Kind       string      `json:"kind"` // applicability, start, stop
	Expression *Expression `json:"expression,omitempty"`
}

// Expression represents a FHIR Expression
type Expression struct {
	Description string `json:"description,omitempty"`
	Name        string `json:"name,omitempty"`
	Language    string `json:"language,omitempty"`
	Expression  string `json:"expression,omitempty"`
	Reference   string `json:"reference,omitempty"`
}

// DataRequirement represents a FHIR DataRequirement
type DataRequirement struct {
	Type             string           `json:"type"`
	Profile          []string         `json:"profile,omitempty"`
	SubjectCodeableConcept *CodeableConcept `json:"subjectCodeableConcept,omitempty"`
	MustSupport      []string         `json:"mustSupport,omitempty"`
	CodeFilter       []CodeFilter     `json:"codeFilter,omitempty"`
	DateFilter       []DateFilter     `json:"dateFilter,omitempty"`
}

// CodeFilter represents a code filter in DataRequirement
type CodeFilter struct {
	Path         string   `json:"path,omitempty"`
	SearchParam  string   `json:"searchParam,omitempty"`
	ValueSet     string   `json:"valueSet,omitempty"`
	Code         []Coding `json:"code,omitempty"`
}

// DateFilter represents a date filter in DataRequirement
type DateFilter struct {
	Path        string   `json:"path,omitempty"`
	SearchParam string   `json:"searchParam,omitempty"`
	ValueDateTime time.Time `json:"valueDateTime,omitempty"`
	ValuePeriod   *Period   `json:"valuePeriod,omitempty"`
	ValueDuration *Duration `json:"valueDuration,omitempty"`
}

// RelatedAction represents a related action
type RelatedAction struct {
	ActionID     string    `json:"actionId"`
	Relationship string    `json:"relationship"` // before-start, before, before-end, concurrent-with-start, concurrent, concurrent-with-end, after-start, after, after-end
	OffsetDuration *Duration `json:"offsetDuration,omitempty"`
	OffsetRange    *Range    `json:"offsetRange,omitempty"`
}

// ActionParticipant represents a participant in an action
type ActionParticipant struct {
	Type string           `json:"type"` // patient, practitioner, related-person, device
	Role *CodeableConcept `json:"role,omitempty"`
}
