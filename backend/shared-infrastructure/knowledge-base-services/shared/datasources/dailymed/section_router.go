// Package dailymed provides section routing for SPL documents.
//
// Phase 3a.3: Section Router for DailyMed SPL
// Key Feature: Route parsed LOINC sections to appropriate Knowledge Bases
//
// This implements the Truth Sourcing Manifest which defines:
// - Which KBs receive each LOINC-coded section
// - Priority levels for extraction
// - Authority sources that may supersede SPL data
package dailymed

import (
	"fmt"
)

// =============================================================================
// LOINC ROUTING CONFIGURATION
// =============================================================================

// SectionRouting defines KB targets and metadata for a LOINC-coded section
type SectionRouting struct {
	LOINCCode          string   // LOINC code (e.g., "34068-7")
	LOINCDisplay       string   // Human-readable name
	TargetKBs          []string // KBs that should receive this section
	Priority           string   // P0_CRITICAL, P1_HIGH, P2_MEDIUM, P3_LOW
	RequiresAuthority  string   // If set, route to this authority instead (e.g., "LACTMED")
	ContentExpectation string   // TABLE, NARRATIVE, MIXED
	Description        string   // Description of section content
}

// Priority levels for section extraction
const (
	PriorityCritical = "P0_CRITICAL" // Safety-critical, must process first
	PriorityHigh     = "P1_HIGH"     // High importance, process second
	PriorityMedium   = "P2_MEDIUM"   // Moderate importance
	PriorityLow      = "P3_LOW"      // Lower importance, process last
)

// Content expectations for sections
const (
	ContentTable     = "TABLE"     // Primarily structured tables
	ContentNarrative = "NARRATIVE" // Primarily text
	ContentMixed     = "MIXED"     // Both tables and narrative
)

// Authority sources that may supersede SPL data
const (
	AuthorityLactMed  = "LACTMED"    // NIH LactMed for breastfeeding
	AuthorityBeers    = "BEERS_STOPP" // AGS Beers Criteria for geriatrics
	AuthorityCPIC     = "CPIC"       // CPIC for pharmacogenomics
	AuthorityCredible = "CREDIBLEMEDS" // CredibleMeds for QT risk
	AuthorityDrugBank = "DRUGBANK"   // DrugBank for PK data
	AuthorityLiverTox = "LIVERTOX"   // NIH LiverTox for hepatotoxicity
)

// =============================================================================
// DEFAULT ROUTING MAP (Truth Sourcing Manifest)
// =============================================================================

// DefaultRoutingMap is the Truth Sourcing Manifest for FDA SPL sections
// This defines which Knowledge Bases receive data from each LOINC-coded section
var DefaultRoutingMap = map[string]SectionRouting{
	// P0_CRITICAL: Safety-critical sections
	LOINCBoxedWarning: {
		LOINCCode:          LOINCBoxedWarning,
		LOINCDisplay:       "BOXED WARNING",
		TargetKBs:          []string{"KB-4"},
		Priority:           PriorityCritical,
		ContentExpectation: ContentNarrative,
		Description:        "Black box warnings - highest severity safety information",
	},
	LOINCDosageAdministration: {
		LOINCCode:          LOINCDosageAdministration,
		LOINCDisplay:       "DOSAGE AND ADMINISTRATION",
		TargetKBs:          []string{"KB-1", "KB-4"},
		Priority:           PriorityCritical,
		ContentExpectation: ContentMixed, // Tables for renal/hepatic + narrative
		Description:        "Dosing instructions including renal/hepatic adjustments",
	},
	LOINCContraindications: {
		LOINCCode:          LOINCContraindications,
		LOINCDisplay:       "CONTRAINDICATIONS",
		TargetKBs:          []string{"KB-4"},
		Priority:           PriorityCritical,
		ContentExpectation: ContentNarrative,
		Description:        "Conditions where drug should not be used",
	},
	LOINCWarningsPrecautions: {
		LOINCCode:          LOINCWarningsPrecautions,
		LOINCDisplay:       "WARNINGS AND PRECAUTIONS",
		TargetKBs:          []string{"KB-4", "KB-16"},
		Priority:           PriorityCritical,
		ContentExpectation: ContentNarrative,
		Description:        "Important warnings and precautionary measures",
	},

	// P1_HIGH: High importance sections
	LOINCDrugInteractions: {
		LOINCCode:          LOINCDrugInteractions,
		LOINCDisplay:       "DRUG INTERACTIONS",
		TargetKBs:          []string{"KB-5"},
		Priority:           PriorityHigh,
		ContentExpectation: ContentMixed,
		Description:        "Drug-drug, drug-food, and drug-lab interactions",
	},
	LOINCAdverseReactions: {
		LOINCCode:          LOINCAdverseReactions,
		LOINCDisplay:       "ADVERSE REACTIONS",
		TargetKBs:          []string{"KB-4"},
		Priority:           PriorityHigh,
		ContentExpectation: ContentMixed, // Often has incidence tables
		Description:        "Adverse drug reactions and their incidence",
	},
	LOINCPregnancy: {
		LOINCCode:          LOINCPregnancy,
		LOINCDisplay:       "PREGNANCY",
		TargetKBs:          []string{"KB-4"},
		Priority:           PriorityHigh,
		ContentExpectation: ContentNarrative,
		Description:        "Pregnancy risk category and considerations",
	},
	LOINCNursing: {
		LOINCCode:          LOINCNursing,
		LOINCDisplay:       "NURSING MOTHERS",
		TargetKBs:          []string{"KB-4"},
		Priority:           PriorityHigh,
		RequiresAuthority:  AuthorityLactMed, // LactMed has RID% data
		ContentExpectation: ContentNarrative,
		Description:        "Breastfeeding considerations and excretion data",
	},
	LOINCPediatricUse: {
		LOINCCode:          LOINCPediatricUse,
		LOINCDisplay:       "PEDIATRIC USE",
		TargetKBs:          []string{"KB-1", "KB-4"},
		Priority:           PriorityHigh,
		ContentExpectation: ContentNarrative,
		Description:        "Pediatric dosing and safety considerations",
	},
	LOINCGeriatricUse: {
		LOINCCode:          LOINCGeriatricUse,
		LOINCDisplay:       "GERIATRIC USE",
		TargetKBs:          []string{"KB-4"},
		Priority:           PriorityHigh,
		RequiresAuthority:  AuthorityBeers, // Beers/STOPP criteria
		ContentExpectation: ContentNarrative,
		Description:        "Geriatric considerations and Beers criteria relevance",
	},

	// P2_MEDIUM: Moderate importance sections
	LOINCClinicalPharm: {
		LOINCCode:          LOINCClinicalPharm,
		LOINCDisplay:       "CLINICAL PHARMACOLOGY",
		TargetKBs:          []string{"KB-1"},
		Priority:           PriorityMedium,
		RequiresAuthority:  AuthorityDrugBank, // DrugBank has detailed PK
		ContentExpectation: ContentTable,
		Description:        "Mechanism of action and pharmacokinetic parameters",
	},
	LOINCOverdosage: {
		LOINCCode:          LOINCOverdosage,
		LOINCDisplay:       "OVERDOSAGE",
		TargetKBs:          []string{"KB-4"},
		Priority:           PriorityMedium,
		ContentExpectation: ContentNarrative,
		Description:        "Signs, symptoms, and treatment of overdose",
	},
	LOINCHowSupplied: {
		LOINCCode:          LOINCHowSupplied,
		LOINCDisplay:       "HOW SUPPLIED",
		TargetKBs:          []string{"KB-6"}, // Formulary
		Priority:           PriorityMedium,
		ContentExpectation: ContentTable,
		Description:        "Available forms, strengths, and NDC codes",
	},

	// Subsections for organ impairment
	LOINCRenalImpairment: {
		LOINCCode:          LOINCRenalImpairment,
		LOINCDisplay:       "RENAL IMPAIRMENT",
		TargetKBs:          []string{"KB-1"},
		Priority:           PriorityCritical, // Critical for dosing
		ContentExpectation: ContentTable,
		Description:        "Dose adjustments based on kidney function",
	},
	LOINCHepaticImpairment: {
		LOINCCode:          LOINCHepaticImpairment,
		LOINCDisplay:       "HEPATIC IMPAIRMENT",
		TargetKBs:          []string{"KB-1"},
		Priority:           PriorityCritical, // Critical for dosing
		RequiresAuthority:  AuthorityLiverTox,
		ContentExpectation: ContentTable,
		Description:        "Dose adjustments based on liver function (Child-Pugh)",
	},
}

// =============================================================================
// SECTION ROUTER
// =============================================================================

// SectionRouter routes parsed SPL sections to appropriate Knowledge Bases
type SectionRouter struct {
	routingMap      map[string]SectionRouting
	tableClassifier *TableClassifier
}

// NewSectionRouter creates a new section router with default routing
func NewSectionRouter() *SectionRouter {
	return &SectionRouter{
		routingMap:      DefaultRoutingMap,
		tableClassifier: NewTableClassifier(),
	}
}

// NewSectionRouterWithCustomRouting creates a router with custom routing map
func NewSectionRouterWithCustomRouting(routingMap map[string]SectionRouting) *SectionRouter {
	return &SectionRouter{
		routingMap:      routingMap,
		tableClassifier: NewTableClassifier(),
	}
}

// =============================================================================
// ROUTING RESULT
// =============================================================================

// RoutedSection represents a section with routing information applied
type RoutedSection struct {
	// Original section data
	Section *SPLSection

	// Routing information
	TargetKBs         []string
	Priority          string
	RequiresAuthority string
	ContentExpectation string

	// Extracted data
	ExtractedTables []*ExtractedTable
	PlainText       string

	// Metadata
	HasTables     bool
	TableCount    int
	NestingLevel  int
	SourceSetID   string
	SourceVersion int
}

// RouteSection applies routing rules to a single section
func (sr *SectionRouter) RouteSection(section *SPLSection, setID string, version int) *RoutedSection {
	routed := &RoutedSection{
		Section:       section,
		PlainText:     section.GetRawText(),
		HasTables:     section.HasTables(),
		TableCount:    len(section.Text.Tables),
		SourceSetID:   setID,
		SourceVersion: version,
	}

	// Look up routing by LOINC code
	if routing, found := sr.routingMap[section.Code.Code]; found {
		routed.TargetKBs = routing.TargetKBs
		routed.Priority = routing.Priority
		routed.RequiresAuthority = routing.RequiresAuthority
		routed.ContentExpectation = routing.ContentExpectation
	} else {
		// Default routing for unknown sections
		routed.TargetKBs = []string{}
		routed.Priority = PriorityLow
		routed.ContentExpectation = ContentNarrative
	}

	// Extract and classify tables if present
	if routed.HasTables {
		routed.ExtractedTables = sr.tableClassifier.ExtractAndClassifyTables(section)

		// Add table-specific KBs to routing
		for _, table := range routed.ExtractedTables {
			for _, kb := range table.TargetKBs {
				if !contains(routed.TargetKBs, kb) {
					routed.TargetKBs = append(routed.TargetKBs, kb)
				}
			}
		}
	}

	return routed
}

// RouteDocument routes all sections in a document
func (sr *SectionRouter) RouteDocument(doc *SPLDocument) []*RoutedSection {
	var routed []*RoutedSection

	// Get SetID and version
	setID := doc.SetID.Root
	if doc.SetID.Extension != "" {
		setID = doc.SetID.Extension
	}

	// Process all sections recursively
	sr.routeSectionsRecursive(doc.Sections, &routed, setID, doc.VersionNumber.Value, 0)

	return routed
}

// routeSectionsRecursive processes sections and their subsections
func (sr *SectionRouter) routeSectionsRecursive(sections []SPLSection, routed *[]*RoutedSection, setID string, version int, level int) {
	for i := range sections {
		section := &sections[i]
		routedSection := sr.RouteSection(section, setID, version)
		routedSection.NestingLevel = level
		*routed = append(*routed, routedSection)

		// Process subsections
		if len(section.Subsections) > 0 {
			sr.routeSectionsRecursive(section.Subsections, routed, setID, version, level+1)
		}
	}
}

// =============================================================================
// ROUTING UTILITIES
// =============================================================================

// GetRoutingForLOINC returns the routing configuration for a LOINC code
func (sr *SectionRouter) GetRoutingForLOINC(loincCode string) (SectionRouting, bool) {
	routing, found := sr.routingMap[loincCode]
	return routing, found
}

// GetSectionsByPriority returns sections grouped by priority
func GetSectionsByPriority(sections []*RoutedSection) map[string][]*RoutedSection {
	grouped := make(map[string][]*RoutedSection)

	for _, section := range sections {
		priority := section.Priority
		grouped[priority] = append(grouped[priority], section)
	}

	return grouped
}

// GetSectionsByTargetKB returns sections grouped by target KB
func GetSectionsByTargetKB(sections []*RoutedSection) map[string][]*RoutedSection {
	grouped := make(map[string][]*RoutedSection)

	for _, section := range sections {
		for _, kb := range section.TargetKBs {
			grouped[kb] = append(grouped[kb], section)
		}
	}

	return grouped
}

// GetCriticalSections returns only P0_CRITICAL priority sections
func GetCriticalSections(sections []*RoutedSection) []*RoutedSection {
	var critical []*RoutedSection

	for _, section := range sections {
		if section.Priority == PriorityCritical {
			critical = append(critical, section)
		}
	}

	return critical
}

// GetTablesForKB returns all extracted tables targeted at a specific KB
func GetTablesForKB(sections []*RoutedSection, kbName string) []*ExtractedTable {
	var tables []*ExtractedTable

	for _, section := range sections {
		for _, table := range section.ExtractedTables {
			if contains(table.TargetKBs, kbName) {
				tables = append(tables, table)
			}
		}
	}

	return tables
}

// =============================================================================
// ROUTING EVENTS
// =============================================================================

// RoutingEvent represents an event for downstream processing
type RoutingEvent struct {
	EventType     string // SECTION_ROUTED, TABLE_EXTRACTED
	SourceSetID   string
	SourceVersion int
	LOINCCode     string
	TargetKBs     []string
	Priority      string
	HasTables     bool
	TableTypes    []TableType
	Timestamp     int64
}

// GenerateRoutingEvents creates events for downstream KB processors
func GenerateRoutingEvents(routed []*RoutedSection) []RoutingEvent {
	var events []RoutingEvent

	for _, section := range routed {
		if len(section.TargetKBs) == 0 {
			continue // Skip unrouted sections
		}

		event := RoutingEvent{
			EventType:     "SECTION_ROUTED",
			SourceSetID:   section.SourceSetID,
			SourceVersion: section.SourceVersion,
			LOINCCode:     section.Section.Code.Code,
			TargetKBs:     section.TargetKBs,
			Priority:      section.Priority,
			HasTables:     section.HasTables,
		}

		// Add table types
		for _, table := range section.ExtractedTables {
			event.TableTypes = append(event.TableTypes, table.TableType)
		}

		events = append(events, event)
	}

	return events
}

// =============================================================================
// VALIDATION
// =============================================================================

// RoutingValidation contains validation results for routing
type RoutingValidation struct {
	TotalSections    int
	RoutedSections   int
	UnroutedSections int
	CriticalSections int
	TablesExtracted  int
	MissingLOINC     []string
	Warnings         []string
}

// ValidateRouting checks routing completeness for a document
func (sr *SectionRouter) ValidateRouting(routed []*RoutedSection) *RoutingValidation {
	validation := &RoutingValidation{
		TotalSections:  len(routed),
		MissingLOINC:   make([]string, 0),
		Warnings:       make([]string, 0),
	}

	for _, section := range routed {
		if len(section.TargetKBs) > 0 {
			validation.RoutedSections++
		} else {
			validation.UnroutedSections++
			if section.Section.Code.Code != "" {
				validation.MissingLOINC = append(validation.MissingLOINC, section.Section.Code.Code)
			}
		}

		if section.Priority == PriorityCritical {
			validation.CriticalSections++
		}

		validation.TablesExtracted += len(section.ExtractedTables)

		// Check for potential issues
		if section.ContentExpectation == ContentTable && !section.HasTables {
			validation.Warnings = append(validation.Warnings,
				fmt.Sprintf("Section %s expected tables but none found", section.Section.Code.Code))
		}
	}

	return validation
}

// =============================================================================
// HELPERS
// =============================================================================

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
