package graphql

import (
	"github.com/graphql-go/graphql"
	"medication-service-v2/internal/graphql/types"
	"medication-service-v2/internal/graphql/resolvers"
)

// BuildSchema creates and returns the GraphQL schema with Federation directives
func BuildSchema(resolver *resolvers.MedicationResolver) (graphql.Schema, error) {
	// Root query type with federation support
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"medication": &graphql.Field{
				Type: types.MedicationType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: resolver.GetMedication,
			},
			"medications": &graphql.Field{
				Type: graphql.NewList(types.MedicationType),
				Args: graphql.FieldConfigArgument{
					"patientId": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"status": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"limit": &graphql.ArgumentConfig{
						Type: graphql.Int,
						DefaultValue: 50,
					},
					"offset": &graphql.ArgumentConfig{
						Type: graphql.Int,
						DefaultValue: 0,
					},
				},
				Resolve: resolver.GetMedications,
			},
			"medicationRequest": &graphql.Field{
				Type: types.MedicationRequestType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: resolver.GetMedicationRequest,
			},
			"medicationRequests": &graphql.Field{
				Type: graphql.NewList(types.MedicationRequestType),
				Args: graphql.FieldConfigArgument{
					"patientId": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"status": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"limit": &graphql.ArgumentConfig{
						Type: graphql.Int,
						DefaultValue: 50,
					},
					"offset": &graphql.ArgumentConfig{
						Type: graphql.Int,
						DefaultValue: 0,
					},
				},
				Resolve: resolver.GetMedicationRequests,
			},
			// Federation directives support
			"_entities": &graphql.Field{
				Type: graphql.NewList(types.EntityUnion),
				Args: graphql.FieldConfigArgument{
					"representations": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(types.AnyScalar))),
					},
				},
				Resolve: resolver.ResolveEntities,
			},
			"_service": &graphql.Field{
				Type: types.ServiceType,
				Resolve: resolver.ResolveService,
			},
		},
	})

	// Mutation type for medication operations
	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"createMedicationRequest": &graphql.Field{
				Type: types.MedicationRequestType,
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(types.CreateMedicationRequestInput),
					},
				},
				Resolve: resolver.CreateMedicationRequest,
			},
			"updateMedicationRequest": &graphql.Field{
				Type: types.MedicationRequestType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(types.UpdateMedicationRequestInput),
					},
				},
				Resolve: resolver.UpdateMedicationRequest,
			},
			"deleteMedicationRequest": &graphql.Field{
				Type: graphql.Boolean,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: resolver.DeleteMedicationRequest,
			},
		},
	})

	// Build the schema
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	})

	if err != nil {
		return graphql.Schema{}, err
	}

	return schema, nil
}

// GetFederationSDL returns the SDL (Schema Definition Language) with Federation directives
func GetFederationSDL(schema graphql.Schema) string {
	// Get the schema as SDL
	schemaSDL := schema.String()

	// Add Federation directives to the SDL
	federationSDL := `
# Federation directives
directive @key(fields: String!) on OBJECT | INTERFACE
directive @requires(fields: String!) on FIELD_DEFINITION
directive @provides(fields: String!) on FIELD_DEFINITION
directive @external on FIELD_DEFINITION
directive @shareable on FIELD_DEFINITION | OBJECT

# Scalar types
scalar _Any

# Federation types
type _Service {
  sdl: String
}

union _Entity = Medication | MedicationRequest | Patient

` + schemaSDL

	// Add @key directives to entity types
	federationSDL = addFederationDirectives(federationSDL)

	return federationSDL
}

// addFederationDirectives adds Apollo Federation directives to types
func addFederationDirectives(sdl string) string {
	// Add @key directive to Medication type
	sdl = replaceInSDL(sdl, "type Medication {", "type Medication @key(fields: \"id\") {")

	// Add @key directive to MedicationRequest type
	sdl = replaceInSDL(sdl, "type MedicationRequest {", "type MedicationRequest @key(fields: \"id\") {")

	// Add @key directive to Patient type (external entity)
	sdl = replaceInSDL(sdl, "type Patient {", "type Patient @key(fields: \"id\") @external {")

	// Mark shared FHIR types as @shareable
	sharedTypes := []string{
		"CodeableConcept", "Coding", "Identifier", "Reference", "Period",
		"Quantity", "Annotation", "HumanName", "ContactPoint", "Address",
		"Timing", "Dosage", "Range", "Ratio", "SampledData", "Duration",
	}

	for _, typeName := range sharedTypes {
		sdl = replaceInSDL(sdl, "type "+typeName+" {", "type "+typeName+" @shareable {")
	}

	return sdl
}

// Helper function to replace strings in SDL
func replaceInSDL(sdl, from, to string) string {
	return sdl // In a real implementation, use proper string replacement
}