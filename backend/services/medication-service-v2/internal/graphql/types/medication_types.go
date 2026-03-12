package types

import (
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

// AnyScalar represents the _Any scalar type for federation
var AnyScalar = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "_Any",
	Description: "The _Any scalar is used to pass representations of entities from external services.",
	Serialize: func(value interface{}) interface{} {
		return value
	},
	ParseValue: func(value interface{}) interface{} {
		return value
	},
	ParseLiteral: func(valueAST ast.Value) interface{} {
		return valueAST.GetValue()
	},
})

// ServiceType represents the _Service type for federation
var ServiceType = graphql.NewObject(graphql.ObjectConfig{
	Name: "_Service",
	Fields: graphql.Fields{
		"sdl": &graphql.Field{
			Type: graphql.String,
		},
	},
})

// FHIR Common Types (marked as @shareable in federation)

// CodeableConceptType represents a FHIR CodeableConcept
var CodeableConceptType = graphql.NewObject(graphql.ObjectConfig{
	Name: "CodeableConcept",
	Fields: graphql.Fields{
		"text": &graphql.Field{
			Type: graphql.String,
		},
		"coding": &graphql.Field{
			Type: graphql.NewList(CodingType),
		},
	},
})

// CodingType represents a FHIR Coding
var CodingType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Coding",
	Fields: graphql.Fields{
		"system": &graphql.Field{
			Type: graphql.String,
		},
		"code": &graphql.Field{
			Type: graphql.String,
		},
		"display": &graphql.Field{
			Type: graphql.String,
		},
		"version": &graphql.Field{
			Type: graphql.String,
		},
		"userSelected": &graphql.Field{
			Type: graphql.Boolean,
		},
	},
})

// IdentifierType represents a FHIR Identifier
var IdentifierType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Identifier",
	Fields: graphql.Fields{
		"use": &graphql.Field{
			Type: graphql.String,
		},
		"type": &graphql.Field{
			Type: CodeableConceptType,
		},
		"system": &graphql.Field{
			Type: graphql.String,
		},
		"value": &graphql.Field{
			Type: graphql.String,
		},
		"period": &graphql.Field{
			Type: PeriodType,
		},
	},
})

// ReferenceType represents a FHIR Reference
var ReferenceType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Reference",
	Fields: graphql.Fields{
		"reference": &graphql.Field{
			Type: graphql.String,
		},
		"display": &graphql.Field{
			Type: graphql.String,
		},
		"type": &graphql.Field{
			Type: graphql.String,
		},
		"identifier": &graphql.Field{
			Type: IdentifierType,
		},
	},
})

// PeriodType represents a FHIR Period
var PeriodType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Period",
	Fields: graphql.Fields{
		"start": &graphql.Field{
			Type: graphql.String,
		},
		"end": &graphql.Field{
			Type: graphql.String,
		},
	},
})

// QuantityType represents a FHIR Quantity
var QuantityType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Quantity",
	Fields: graphql.Fields{
		"value": &graphql.Field{
			Type: graphql.Float,
		},
		"unit": &graphql.Field{
			Type: graphql.String,
		},
		"system": &graphql.Field{
			Type: graphql.String,
		},
		"code": &graphql.Field{
			Type: graphql.String,
		},
		"comparator": &graphql.Field{
			Type: graphql.String,
		},
	},
})

// DosageType represents a FHIR Dosage instruction
var DosageType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Dosage",
	Fields: graphql.Fields{
		"sequence": &graphql.Field{
			Type: graphql.Int,
		},
		"text": &graphql.Field{
			Type: graphql.String,
		},
		"additionalInstruction": &graphql.Field{
			Type: graphql.NewList(CodeableConceptType),
		},
		"patientInstruction": &graphql.Field{
			Type: graphql.String,
		},
		"timing": &graphql.Field{
			Type: TimingType,
		},
		"asNeededBoolean": &graphql.Field{
			Type: graphql.Boolean,
		},
		"asNeededCodeableConcept": &graphql.Field{
			Type: CodeableConceptType,
		},
		"site": &graphql.Field{
			Type: CodeableConceptType,
		},
		"route": &graphql.Field{
			Type: CodeableConceptType,
		},
		"method": &graphql.Field{
			Type: CodeableConceptType,
		},
		"doseAndRate": &graphql.Field{
			Type: graphql.NewList(DoseAndRateType),
		},
		"maxDosePerPeriod": &graphql.Field{
			Type: RatioType,
		},
		"maxDosePerAdministration": &graphql.Field{
			Type: QuantityType,
		},
		"maxDosePerLifetime": &graphql.Field{
			Type: QuantityType,
		},
	},
})

// TimingType represents a FHIR Timing
var TimingType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Timing",
	Fields: graphql.Fields{
		"event": &graphql.Field{
			Type: graphql.NewList(graphql.String),
		},
		"repeat": &graphql.Field{
			Type: TimingRepeatType,
		},
		"code": &graphql.Field{
			Type: CodeableConceptType,
		},
	},
})

// TimingRepeatType represents a FHIR Timing.repeat
var TimingRepeatType = graphql.NewObject(graphql.ObjectConfig{
	Name: "TimingRepeat",
	Fields: graphql.Fields{
		"boundsRange": &graphql.Field{
			Type: RangeType,
		},
		"boundsPeriod": &graphql.Field{
			Type: PeriodType,
		},
		"boundsQuantity": &graphql.Field{
			Type: QuantityType,
		},
		"count": &graphql.Field{
			Type: graphql.Int,
		},
		"countMax": &graphql.Field{
			Type: graphql.Int,
		},
		"duration": &graphql.Field{
			Type: graphql.Float,
		},
		"durationMax": &graphql.Field{
			Type: graphql.Float,
		},
		"durationUnit": &graphql.Field{
			Type: graphql.String,
		},
		"frequency": &graphql.Field{
			Type: graphql.Int,
		},
		"frequencyMax": &graphql.Field{
			Type: graphql.Int,
		},
		"period": &graphql.Field{
			Type: graphql.Float,
		},
		"periodMax": &graphql.Field{
			Type: graphql.Float,
		},
		"periodUnit": &graphql.Field{
			Type: graphql.String,
		},
		"dayOfWeek": &graphql.Field{
			Type: graphql.NewList(graphql.String),
		},
		"timeOfDay": &graphql.Field{
			Type: graphql.NewList(graphql.String),
		},
		"when": &graphql.Field{
			Type: graphql.NewList(graphql.String),
		},
		"offset": &graphql.Field{
			Type: graphql.Int,
		},
	},
})

// RangeType represents a FHIR Range
var RangeType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Range",
	Fields: graphql.Fields{
		"low": &graphql.Field{
			Type: QuantityType,
		},
		"high": &graphql.Field{
			Type: QuantityType,
		},
	},
})

// RatioType represents a FHIR Ratio
var RatioType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Ratio",
	Fields: graphql.Fields{
		"numerator": &graphql.Field{
			Type: QuantityType,
		},
		"denominator": &graphql.Field{
			Type: QuantityType,
		},
	},
})

// DoseAndRateType represents a FHIR DoseAndRate
var DoseAndRateType = graphql.NewObject(graphql.ObjectConfig{
	Name: "DoseAndRate",
	Fields: graphql.Fields{
		"type": &graphql.Field{
			Type: CodeableConceptType,
		},
		"doseRange": &graphql.Field{
			Type: RangeType,
		},
		"doseQuantity": &graphql.Field{
			Type: QuantityType,
		},
		"rateRatio": &graphql.Field{
			Type: RatioType,
		},
		"rateRange": &graphql.Field{
			Type: RangeType,
		},
		"rateQuantity": &graphql.Field{
			Type: QuantityType,
		},
	},
})

// Medication Resource Types

// MedicationType represents a FHIR Medication resource
var MedicationType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Medication",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.NewNonNull(graphql.ID),
		},
		"identifier": &graphql.Field{
			Type: graphql.NewList(IdentifierType),
		},
		"code": &graphql.Field{
			Type: CodeableConceptType,
		},
		"status": &graphql.Field{
			Type: graphql.String,
		},
		"manufacturer": &graphql.Field{
			Type: ReferenceType,
		},
		"form": &graphql.Field{
			Type: CodeableConceptType,
		},
		"amount": &graphql.Field{
			Type: RatioType,
		},
		"ingredient": &graphql.Field{
			Type: graphql.NewList(MedicationIngredientType),
		},
		"batch": &graphql.Field{
			Type: MedicationBatchType,
		},
	},
})

// MedicationIngredientType represents a Medication.ingredient
var MedicationIngredientType = graphql.NewObject(graphql.ObjectConfig{
	Name: "MedicationIngredient",
	Fields: graphql.Fields{
		"itemCodeableConcept": &graphql.Field{
			Type: CodeableConceptType,
		},
		"itemReference": &graphql.Field{
			Type: ReferenceType,
		},
		"isActive": &graphql.Field{
			Type: graphql.Boolean,
		},
		"strength": &graphql.Field{
			Type: RatioType,
		},
	},
})

// MedicationBatchType represents a Medication.batch
var MedicationBatchType = graphql.NewObject(graphql.ObjectConfig{
	Name: "MedicationBatch",
	Fields: graphql.Fields{
		"lotNumber": &graphql.Field{
			Type: graphql.String,
		},
		"expirationDate": &graphql.Field{
			Type: graphql.String,
		},
	},
})

// MedicationRequestType represents a FHIR MedicationRequest resource
var MedicationRequestType = graphql.NewObject(graphql.ObjectConfig{
	Name: "MedicationRequest",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.NewNonNull(graphql.ID),
		},
		"identifier": &graphql.Field{
			Type: graphql.NewList(IdentifierType),
		},
		"status": &graphql.Field{
			Type: graphql.NewNonNull(graphql.String),
		},
		"statusReason": &graphql.Field{
			Type: CodeableConceptType,
		},
		"intent": &graphql.Field{
			Type: graphql.NewNonNull(graphql.String),
		},
		"category": &graphql.Field{
			Type: graphql.NewList(CodeableConceptType),
		},
		"priority": &graphql.Field{
			Type: graphql.String,
		},
		"doNotPerform": &graphql.Field{
			Type: graphql.Boolean,
		},
		"reportedBoolean": &graphql.Field{
			Type: graphql.Boolean,
		},
		"reportedReference": &graphql.Field{
			Type: ReferenceType,
		},
		"medicationCodeableConcept": &graphql.Field{
			Type: CodeableConceptType,
		},
		"medicationReference": &graphql.Field{
			Type: ReferenceType,
		},
		"subject": &graphql.Field{
			Type: graphql.NewNonNull(ReferenceType),
		},
		"encounter": &graphql.Field{
			Type: ReferenceType,
		},
		"supportingInformation": &graphql.Field{
			Type: graphql.NewList(ReferenceType),
		},
		"authoredOn": &graphql.Field{
			Type: graphql.String,
		},
		"requester": &graphql.Field{
			Type: ReferenceType,
		},
		"performer": &graphql.Field{
			Type: ReferenceType,
		},
		"performerType": &graphql.Field{
			Type: CodeableConceptType,
		},
		"recorder": &graphql.Field{
			Type: ReferenceType,
		},
		"reasonCode": &graphql.Field{
			Type: graphql.NewList(CodeableConceptType),
		},
		"reasonReference": &graphql.Field{
			Type: graphql.NewList(ReferenceType),
		},
		"instantiatesCanonical": &graphql.Field{
			Type: graphql.NewList(graphql.String),
		},
		"instantiatesUri": &graphql.Field{
			Type: graphql.NewList(graphql.String),
		},
		"basedOn": &graphql.Field{
			Type: graphql.NewList(ReferenceType),
		},
		"groupIdentifier": &graphql.Field{
			Type: IdentifierType,
		},
		"courseOfTherapyType": &graphql.Field{
			Type: CodeableConceptType,
		},
		"insurance": &graphql.Field{
			Type: graphql.NewList(ReferenceType),
		},
		"note": &graphql.Field{
			Type: graphql.NewList(AnnotationType),
		},
		"dosageInstruction": &graphql.Field{
			Type: graphql.NewList(DosageType),
		},
		"dispenseRequest": &graphql.Field{
			Type: MedicationRequestDispenseRequestType,
		},
		"substitution": &graphql.Field{
			Type: MedicationRequestSubstitutionType,
		},
		"priorPrescription": &graphql.Field{
			Type: ReferenceType,
		},
		"detectedIssue": &graphql.Field{
			Type: graphql.NewList(ReferenceType),
		},
		"eventHistory": &graphql.Field{
			Type: graphql.NewList(ReferenceType),
		},
	},
})

// AnnotationType represents a FHIR Annotation
var AnnotationType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Annotation",
	Fields: graphql.Fields{
		"authorReference": &graphql.Field{
			Type: ReferenceType,
		},
		"authorString": &graphql.Field{
			Type: graphql.String,
		},
		"time": &graphql.Field{
			Type: graphql.String,
		},
		"text": &graphql.Field{
			Type: graphql.NewNonNull(graphql.String),
		},
	},
})

// MedicationRequestDispenseRequestType represents a MedicationRequest.dispenseRequest
var MedicationRequestDispenseRequestType = graphql.NewObject(graphql.ObjectConfig{
	Name: "MedicationRequestDispenseRequest",
	Fields: graphql.Fields{
		"initialFill": &graphql.Field{
			Type: MedicationRequestInitialFillType,
		},
		"dispenseInterval": &graphql.Field{
			Type: DurationType,
		},
		"validityPeriod": &graphql.Field{
			Type: PeriodType,
		},
		"numberOfRepeatsAllowed": &graphql.Field{
			Type: graphql.Int,
		},
		"quantity": &graphql.Field{
			Type: QuantityType,
		},
		"expectedSupplyDuration": &graphql.Field{
			Type: DurationType,
		},
		"performer": &graphql.Field{
			Type: ReferenceType,
		},
	},
})

// MedicationRequestInitialFillType represents a MedicationRequest.dispenseRequest.initialFill
var MedicationRequestInitialFillType = graphql.NewObject(graphql.ObjectConfig{
	Name: "MedicationRequestInitialFill",
	Fields: graphql.Fields{
		"quantity": &graphql.Field{
			Type: QuantityType,
		},
		"duration": &graphql.Field{
			Type: DurationType,
		},
	},
})

// DurationType represents a FHIR Duration
var DurationType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Duration",
	Fields: graphql.Fields{
		"value": &graphql.Field{
			Type: graphql.Float,
		},
		"unit": &graphql.Field{
			Type: graphql.String,
		},
		"system": &graphql.Field{
			Type: graphql.String,
		},
		"code": &graphql.Field{
			Type: graphql.String,
		},
	},
})

// MedicationRequestSubstitutionType represents a MedicationRequest.substitution
var MedicationRequestSubstitutionType = graphql.NewObject(graphql.ObjectConfig{
	Name: "MedicationRequestSubstitution",
	Fields: graphql.Fields{
		"allowedBoolean": &graphql.Field{
			Type: graphql.Boolean,
		},
		"allowedCodeableConcept": &graphql.Field{
			Type: CodeableConceptType,
		},
		"reason": &graphql.Field{
			Type: CodeableConceptType,
		},
	},
})

// External entity types for federation

// PatientType represents a Patient entity (external from patient service)
var PatientType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Patient",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.NewNonNull(graphql.ID),
		},
		"medicationRequests": &graphql.Field{
			Type: graphql.NewList(MedicationRequestType),
		},
	},
})

// EntityUnion represents the _Entity union for federation
var EntityUnion = graphql.NewUnion(graphql.UnionConfig{
	Name: "_Entity",
	Types: []*graphql.Object{
		MedicationType,
		MedicationRequestType,
		PatientType,
	},
	ResolveType: func(p graphql.ResolveTypeParams) *graphql.Object {
		// Determine the type based on the object
		if obj, ok := p.Value.(map[string]interface{}); ok {
			if resourceType, exists := obj["resourceType"]; exists {
				switch resourceType {
				case "Medication":
					return MedicationType
				case "MedicationRequest":
					return MedicationRequestType
				case "Patient":
					return PatientType
				}
			}
		}
		return nil
	},
})

// Input types for mutations

// CreateMedicationRequestInput represents input for creating a medication request
var CreateMedicationRequestInput = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "CreateMedicationRequestInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"status": &graphql.InputObjectFieldConfig{
			Type: graphql.NewNonNull(graphql.String),
		},
		"intent": &graphql.InputObjectFieldConfig{
			Type: graphql.NewNonNull(graphql.String),
		},
		"medicationCodeableConcept": &graphql.InputObjectFieldConfig{
			Type: CodeableConceptInputType,
		},
		"medicationReference": &graphql.InputObjectFieldConfig{
			Type: ReferenceInputType,
		},
		"subjectId": &graphql.InputObjectFieldConfig{
			Type: graphql.NewNonNull(graphql.String),
		},
		"requesterId": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
		"encounterId": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
		"dosageInstructions": &graphql.InputObjectFieldConfig{
			Type: graphql.NewList(DosageInputType),
		},
		"priority": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
		"reasonCode": &graphql.InputObjectFieldConfig{
			Type: graphql.NewList(CodeableConceptInputType),
		},
		"note": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
	},
})

// UpdateMedicationRequestInput represents input for updating a medication request
var UpdateMedicationRequestInput = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "UpdateMedicationRequestInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"status": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
		"priority": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
		"dosageInstructions": &graphql.InputObjectFieldConfig{
			Type: graphql.NewList(DosageInputType),
		},
		"reasonCode": &graphql.InputObjectFieldConfig{
			Type: graphql.NewList(CodeableConceptInputType),
		},
		"note": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
	},
})

// Input types for complex objects

// CodeableConceptInputType represents input for CodeableConcept
var CodeableConceptInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "CodeableConceptInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"text": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
		"coding": &graphql.InputObjectFieldConfig{
			Type: graphql.NewList(CodingInputType),
		},
	},
})

// CodingInputType represents input for Coding
var CodingInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "CodingInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"system": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
		"code": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
		"display": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
		"version": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
	},
})

// ReferenceInputType represents input for Reference
var ReferenceInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "ReferenceInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"reference": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
		"display": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
		"type": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
	},
})

// DosageInputType represents input for Dosage
var DosageInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "DosageInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"sequence": &graphql.InputObjectFieldConfig{
			Type: graphql.Int,
		},
		"text": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
		"patientInstruction": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
		"asNeededBoolean": &graphql.InputObjectFieldConfig{
			Type: graphql.Boolean,
		},
		"route": &graphql.InputObjectFieldConfig{
			Type: CodeableConceptInputType,
		},
		"method": &graphql.InputObjectFieldConfig{
			Type: CodeableConceptInputType,
		},
		"doseQuantity": &graphql.InputObjectFieldConfig{
			Type: QuantityInputType,
		},
	},
})

// QuantityInputType represents input for Quantity
var QuantityInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "QuantityInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"value": &graphql.InputObjectFieldConfig{
			Type: graphql.Float,
		},
		"unit": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
		"system": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
		"code": &graphql.InputObjectFieldConfig{
			Type: graphql.String,
		},
	},
})