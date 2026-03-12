// Package models contains domain models for KB-14 Care Navigator
package models

import "time"

// FHIRTask represents a FHIR R4 Task resource
// Reference: https://www.hl7.org/fhir/task.html
type FHIRTask struct {
	ResourceType    string                `json:"resourceType"` // Always "Task"
	ID              string                `json:"id"`
	Meta            *FHIRMeta             `json:"meta,omitempty"`
	Identifier      []FHIRIdentifier      `json:"identifier,omitempty"`
	InstantiatesURI string                `json:"instantiatesUri,omitempty"`
	BasedOn         []FHIRReference       `json:"basedOn,omitempty"`
	GroupIdentifier *FHIRIdentifier       `json:"groupIdentifier,omitempty"`
	PartOf          []FHIRReference       `json:"partOf,omitempty"`
	Status          string                `json:"status"` // draft | requested | received | accepted | rejected | ready | cancelled | in-progress | on-hold | failed | completed | entered-in-error
	StatusReason    *FHIRCodeableConcept  `json:"statusReason,omitempty"`
	BusinessStatus  *FHIRCodeableConcept  `json:"businessStatus,omitempty"`
	Intent          string                `json:"intent"` // unknown | proposal | plan | order | original-order | reflex-order | filler-order | instance-order | option
	Priority        string                `json:"priority,omitempty"` // routine | urgent | asap | stat
	Code            *FHIRCodeableConcept  `json:"code,omitempty"`
	Description     string                `json:"description,omitempty"`
	Focus           *FHIRReference        `json:"focus,omitempty"`
	For             *FHIRReference        `json:"for,omitempty"` // Patient reference
	Encounter       *FHIRReference        `json:"encounter,omitempty"`
	ExecutionPeriod *FHIRPeriod           `json:"executionPeriod,omitempty"`
	AuthoredOn      *string               `json:"authoredOn,omitempty"` // dateTime
	LastModified    *string               `json:"lastModified,omitempty"` // dateTime
	Requester       *FHIRReference        `json:"requester,omitempty"`
	PerformerType   []FHIRCodeableConcept `json:"performerType,omitempty"`
	Owner           *FHIRReference        `json:"owner,omitempty"` // Assigned to
	Location        *FHIRReference        `json:"location,omitempty"`
	ReasonCode      *FHIRCodeableConcept  `json:"reasonCode,omitempty"`
	ReasonReference *FHIRReference        `json:"reasonReference,omitempty"`
	Insurance       []FHIRReference       `json:"insurance,omitempty"`
	Note            []FHIRAnnotation      `json:"note,omitempty"`
	RelevantHistory []FHIRReference       `json:"relevantHistory,omitempty"`
	Restriction     *FHIRTaskRestriction  `json:"restriction,omitempty"`
	Input           []FHIRTaskParameter   `json:"input,omitempty"`
	Output          []FHIRTaskParameter   `json:"output,omitempty"`
}

// FHIRMeta represents FHIR resource metadata
type FHIRMeta struct {
	VersionID   string   `json:"versionId,omitempty"`
	LastUpdated string   `json:"lastUpdated,omitempty"`
	Source      string   `json:"source,omitempty"`
	Profile     []string `json:"profile,omitempty"`
	Tag         []FHIRCoding `json:"tag,omitempty"`
}

// FHIRIdentifier represents a FHIR Identifier
type FHIRIdentifier struct {
	Use    string           `json:"use,omitempty"` // usual | official | temp | secondary | old
	Type   *FHIRCodeableConcept `json:"type,omitempty"`
	System string           `json:"system,omitempty"`
	Value  string           `json:"value,omitempty"`
	Period *FHIRPeriod      `json:"period,omitempty"`
}

// FHIRReference represents a FHIR Reference to another resource
type FHIRReference struct {
	Reference  string      `json:"reference,omitempty"`
	Type       string      `json:"type,omitempty"`
	Identifier *FHIRIdentifier `json:"identifier,omitempty"`
	Display    string      `json:"display,omitempty"`
}

// FHIRCodeableConcept represents a FHIR CodeableConcept
type FHIRCodeableConcept struct {
	Coding []FHIRCoding `json:"coding,omitempty"`
	Text   string       `json:"text,omitempty"`
}

// FHIRCoding represents a FHIR Coding
type FHIRCoding struct {
	System       string `json:"system,omitempty"`
	Version      string `json:"version,omitempty"`
	Code         string `json:"code,omitempty"`
	Display      string `json:"display,omitempty"`
	UserSelected bool   `json:"userSelected,omitempty"`
}

// FHIRPeriod represents a FHIR Period
type FHIRPeriod struct {
	Start string `json:"start,omitempty"` // dateTime
	End   string `json:"end,omitempty"`   // dateTime
}

// FHIRAnnotation represents a FHIR Annotation (note)
type FHIRAnnotation struct {
	AuthorReference *FHIRReference `json:"authorReference,omitempty"`
	AuthorString    string         `json:"authorString,omitempty"`
	Time            string         `json:"time,omitempty"` // dateTime
	Text            string         `json:"text"`
}

// FHIRTaskRestriction represents Task.restriction
type FHIRTaskRestriction struct {
	Repetitions int             `json:"repetitions,omitempty"`
	Period      *FHIRPeriod     `json:"period,omitempty"`
	Recipient   []FHIRReference `json:"recipient,omitempty"`
}

// FHIRTaskParameter represents Task.input or Task.output
type FHIRTaskParameter struct {
	Type  FHIRCodeableConcept `json:"type"`
	Value interface{}         `json:"value"` // Can be various types
}

// FHIRBundle represents a FHIR Bundle for search results
type FHIRBundle struct {
	ResourceType string           `json:"resourceType"` // Always "Bundle"
	ID           string           `json:"id,omitempty"`
	Type         string           `json:"type"` // searchset
	Total        int              `json:"total,omitempty"`
	Link         []FHIRBundleLink `json:"link,omitempty"`
	Entry        []FHIRBundleEntry `json:"entry,omitempty"`
}

// FHIRBundleLink represents a link in a FHIR Bundle
type FHIRBundleLink struct {
	Relation string `json:"relation"`
	URL      string `json:"url"`
}

// FHIRBundleEntry represents an entry in a FHIR Bundle
type FHIRBundleEntry struct {
	FullURL  string      `json:"fullUrl,omitempty"`
	Resource interface{} `json:"resource,omitempty"`
	Search   *FHIRBundleEntrySearch `json:"search,omitempty"`
}

// FHIRBundleEntrySearch represents search information for a bundle entry
type FHIRBundleEntrySearch struct {
	Mode  string  `json:"mode,omitempty"` // match | include | outcome
	Score float64 `json:"score,omitempty"`
}

// FHIR Task Status Mapping
var TaskStatusToFHIR = map[TaskStatus]string{
	TaskStatusCreated:    "requested",
	TaskStatusAssigned:   "accepted",
	TaskStatusInProgress: "in-progress",
	TaskStatusCompleted:  "completed",
	TaskStatusVerified:   "completed",
	TaskStatusDeclined:   "rejected",
	TaskStatusBlocked:    "on-hold",
	TaskStatusEscalated:  "in-progress",
	TaskStatusCancelled:  "cancelled",
}

// FHIR to Task Status Mapping
var FHIRStatusToTask = map[string]TaskStatus{
	"draft":      TaskStatusCreated,
	"requested":  TaskStatusCreated,
	"received":   TaskStatusCreated,
	"accepted":   TaskStatusAssigned,
	"rejected":   TaskStatusDeclined,
	"ready":      TaskStatusAssigned,
	"cancelled":  TaskStatusCancelled,
	"in-progress": TaskStatusInProgress,
	"on-hold":    TaskStatusBlocked,
	"failed":     TaskStatusCancelled,
	"completed":  TaskStatusCompleted,
}

// FHIR Priority Mapping
var TaskPriorityToFHIR = map[TaskPriority]string{
	TaskPriorityCritical: "stat",
	TaskPriorityHigh:     "asap",
	TaskPriorityMedium:   "urgent",
	TaskPriorityLow:      "routine",
}

var FHIRPriorityToTask = map[string]TaskPriority{
	"stat":    TaskPriorityCritical,
	"asap":    TaskPriorityHigh,
	"urgent":  TaskPriorityMedium,
	"routine": TaskPriorityLow,
}

// FormatFHIRDateTime formats a time.Time to FHIR dateTime format
func FormatFHIRDateTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// ParseFHIRDateTime parses a FHIR dateTime string to time.Time
func ParseFHIRDateTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// KB-14 Task Type Code System
const KB14TaskTypeSystem = "urn:kb14:task-type"

// KB-14 Task Identifier System
const KB14IdentifierSystem = "urn:kb14:task-id"
