// KB-1 Drug Dosing Rule Approval Flow Demo
// This demonstrates the complete governance workflow for high-risk clinical content

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"kb-0-governance-platform/internal/models"
	"kb-0-governance-platform/internal/workflow"
)

// Colors for terminal output
const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	Bold    = "\033[1m"
)

func main() {
	fmt.Println(Bold + Cyan + `
╔══════════════════════════════════════════════════════════════════════════════╗
║                    KB-0 UNIFIED GOVERNANCE PLATFORM                          ║
║                    KB-1 Drug Dosing Rule Approval Demo                       ║
╚══════════════════════════════════════════════════════════════════════════════╝` + Reset)

	// Initialize components
	store := newDemoStore()
	audit := &demoAuditLogger{}
	notifier := &demoNotifier{}
	engine := workflow.NewEngine(store, audit, notifier)
	ctx := context.Background()

	// Create Warfarin dosing rule (high-risk anticoagulant)
	item := createWarfarinDosingRule()
	store.items[item.ID] = item

	printSection("DRUG RULE SUBMITTED FOR GOVERNANCE")
	printDrugInfo(item)

	fmt.Println(Yellow + "\n📋 Workflow Template: " + Bold + "CLINICAL_HIGH" + Reset)
	fmt.Println(Yellow + "   Required Path: DRAFT → PRIMARY_REVIEW → SECONDARY_REVIEW → CMO_APPROVAL → APPROVED → ACTIVE" + Reset)
	fmt.Println(Yellow + "   Dual Review: " + Bold + "REQUIRED" + Reset + Yellow + " (high-alert medication)" + Reset)

	// ═══════════════════════════════════════════════════════════════════════════
	// STEP 1: Primary Pharmacist Review
	// ═══════════════════════════════════════════════════════════════════════════
	printStep(1, "PRIMARY PHARMACIST REVIEW", "Dr. Sarah Chen, PharmD, BCPS")

	result, err := engine.SubmitReview(ctx, &workflow.ReviewRequest{
		ItemID:       item.ID,
		ReviewerID:   "pharmacist-001",
		ReviewerName: "Dr. Sarah Chen",
		ReviewerRole: "pharmacist",
		Credentials:  "PharmD, BCPS - Board Certified Pharmacotherapy Specialist",
		Checklist: &models.ReviewChecklist{
			Items: []models.ChecklistItem{
				{ID: "dose_verification", Label: "Dose verified against regulatory label", Required: true, Verified: true},
				{ID: "renal_adjustment", Label: "Renal adjustments verified", Required: true, Verified: true},
				{ID: "hepatic_adjustment", Label: "Hepatic adjustments verified", Required: true, Verified: true},
				{ID: "interactions_checked", Label: "Drug interactions reviewed", Required: true, Verified: true},
				{ID: "monitoring_validated", Label: "Monitoring requirements validated", Required: true, Verified: true},
				{ID: "contraindications_verified", Label: "Contraindications verified", Required: true, Verified: true},
			},
		},
		Notes: "Verified warfarin dosing against FDA label. INR monitoring requirements confirmed. Drug interactions documented.",
	})
	handleResult(result, err, "Primary Review")

	// ═══════════════════════════════════════════════════════════════════════════
	// STEP 2: Secondary Pharmacist Review (Dual Review Requirement)
	// ═══════════════════════════════════════════════════════════════════════════
	printStep(2, "SECONDARY PHARMACIST REVIEW (DUAL REVIEW)", "Dr. Michael Torres, PharmD, CACP")

	result, err = engine.SubmitReview(ctx, &workflow.ReviewRequest{
		ItemID:       item.ID,
		ReviewerID:   "pharmacist-002",
		ReviewerName: "Dr. Michael Torres",
		ReviewerRole: "pharmacist",
		Credentials:  "PharmD, CACP - Certified Anticoagulation Care Provider",
		Checklist: &models.ReviewChecklist{
			Items: []models.ChecklistItem{
				{ID: "dose_verification", Label: "Dose verified against regulatory label", Required: true, Verified: true},
				{ID: "renal_adjustment", Label: "Renal adjustments verified", Required: true, Verified: true},
				{ID: "hepatic_adjustment", Label: "Hepatic adjustments verified", Required: true, Verified: true},
				{ID: "interactions_checked", Label: "Drug interactions reviewed", Required: true, Verified: true},
				{ID: "monitoring_validated", Label: "Monitoring requirements validated", Required: true, Verified: true},
				{ID: "contraindications_verified", Label: "Contraindications verified", Required: true, Verified: true},
			},
		},
		Notes: "Confirmed dosing algorithms align with CHEST guidelines. Bridging protocols verified.",
	})
	handleResult(result, err, "Secondary Review")

	// ═══════════════════════════════════════════════════════════════════════════
	// STEP 3: Route to CMO Approval
	// ═══════════════════════════════════════════════════════════════════════════
	printStep(3, "ROUTING TO CMO APPROVAL", "Workflow System")

	result, err = engine.Transition(ctx, &workflow.TransitionRequest{
		ItemID:    item.ID,
		Action:    "route_to_approval",
		ActorID:   "system",
		ActorName: "Workflow System",
		ActorRole: "system",
	})
	handleResult(result, err, "Route to CMO")

	// ═══════════════════════════════════════════════════════════════════════════
	// STEP 4: CMO Approval with Attestations
	// ═══════════════════════════════════════════════════════════════════════════
	printStep(4, "CHIEF MEDICAL OFFICER APPROVAL", "Dr. Elizabeth Warren, MD, MBA, FACP")

	result, err = engine.Approve(ctx, &workflow.ApprovalRequest{
		ItemID:       item.ID,
		ApproverID:   "cmo-001",
		ApproverName: "Dr. Elizabeth Warren",
		ApproverRole: "cmo",
		Credentials:  "MD, MBA, FACP - Chief Medical Officer",
		Attestations: map[string]bool{
			"medical_responsibility": true,
			"clinical_standards":     true,
		},
		Notes: "Approved for clinical use. Dosing aligns with evidence-based guidelines.",
	})
	handleResult(result, err, "CMO Approval")

	// ═══════════════════════════════════════════════════════════════════════════
	// STEP 5: Activation for Clinical Use
	// ═══════════════════════════════════════════════════════════════════════════
	printStep(5, "ACTIVATION FOR CLINICAL USE", "Activation Engine")

	result, err = engine.Activate(ctx, item.ID)
	handleResult(result, err, "Activation")

	// ═══════════════════════════════════════════════════════════════════════════
	// FINAL: Display Complete Governance Trail
	// ═══════════════════════════════════════════════════════════════════════════
	printSection("COMPLETE GOVERNANCE TRAIL")

	item = store.items[item.ID]
	printGovernanceTrail(item)

	printSection("AUDIT LOG (IMMUTABLE)")
	printAuditLog(audit.entries)

	fmt.Println(Bold + Green + `
╔══════════════════════════════════════════════════════════════════════════════╗
║  ✅ WARFARIN DOSING RULE NOW ACTIVE FOR CLINICAL USE                        ║
║     Full governance trail preserved for regulatory compliance                ║
╚══════════════════════════════════════════════════════════════════════════════╝` + Reset)
}

// createWarfarinDosingRule creates a KB-1 warfarin dosing rule for the demo
func createWarfarinDosingRule() *models.KnowledgeItem {
	dosingRule := map[string]interface{}{
		"drug": map[string]interface{}{
			"rxnorm_code":  "11289",
			"name":         "Warfarin Sodium",
			"generic_name": "warfarin",
			"drug_class":   "Anticoagulant - Vitamin K Antagonist",
			"atc_code":     "B01AA03",
		},
		"dosing": map[string]interface{}{
			"primary_method": "INDICATION_BASED",
			"adult": map[string]interface{}{
				"standard": []map[string]interface{}{
					{
						"indication":   "Atrial Fibrillation",
						"initial_dose": 5.0,
						"unit":         "mg",
						"frequency":    "DAILY",
						"target_inr":   "2.0-3.0",
					},
					{
						"indication":   "DVT/PE Treatment",
						"initial_dose": 5.0,
						"unit":         "mg",
						"frequency":    "DAILY",
						"target_inr":   "2.0-3.0",
						"duration":     "3-6 months",
					},
				},
			},
			"geriatric": map[string]interface{}{
				"start_low":    true,
				"initial_dose": 2.5,
				"unit":         "mg",
			},
		},
		"safety": map[string]interface{}{
			"high_alert_drug":          true,
			"narrow_therapeutic_index": true,
			"black_box_warning":        true,
			"black_box_text":           "WARNING: BLEEDING RISK",
			"monitoring":               []string{"INR", "Signs of bleeding"},
			"contraindications":        []string{"Active bleeding", "Pregnancy"},
			"major_interactions":       []string{"NSAIDs", "Aspirin", "Amiodarone"},
		},
	}

	contentJSON, _ := json.Marshal(dosingRule)
	hash := sha256.Sum256(contentJSON)
	contentHash := hex.EncodeToString(hash[:])

	return &models.KnowledgeItem{
		ID:                 "kb1:drug:warfarin:11289",
		Type:               models.TypeDosingRule,
		KB:                 models.KB1,
		Name:               "Warfarin Sodium Dosing Rule",
		Description:        "Comprehensive warfarin dosing guidance for anticoagulation therapy",
		ContentRef:         "kb1/drugs/warfarin/11289.yaml",
		ContentHash:        contentHash,
		Version:            "1.0.0",
		State:              models.StateDraft,
		WorkflowTemplate:   models.TemplateClinicalHigh,
		RequiresDualReview: true,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		Governance: models.GovernanceTrail{
			CreatedBy: "FDA DailyMed Ingestion",
			Reviews:   []models.Review{},
		},
		Source: models.SourceAttribution{
			Authority:    models.AuthorityFDA,
			Document:     "DailyMed SPL - Warfarin Sodium Tablets",
			URL:          "https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid=e8b794d7",
			Jurisdiction: models.JurisdictionUS,
		},
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// DEMO INFRASTRUCTURE
// ═══════════════════════════════════════════════════════════════════════════════

type demoStore struct {
	items map[string]*models.KnowledgeItem
}

func newDemoStore() *demoStore {
	return &demoStore{items: make(map[string]*models.KnowledgeItem)}
}

func (s *demoStore) GetItem(ctx context.Context, id string) (*models.KnowledgeItem, error) {
	if item, ok := s.items[id]; ok {
		return item, nil
	}
	return nil, fmt.Errorf("item not found: %s", id)
}

func (s *demoStore) UpdateItem(ctx context.Context, item *models.KnowledgeItem) error {
	s.items[item.ID] = item
	return nil
}

func (s *demoStore) GetItemsByState(ctx context.Context, kb models.KB, states []models.ItemState) ([]*models.KnowledgeItem, error) {
	var result []*models.KnowledgeItem
	for _, item := range s.items {
		if item.KB == kb {
			for _, state := range states {
				if item.State == state {
					result = append(result, item)
					break
				}
			}
		}
	}
	return result, nil
}

type demoAuditLogger struct {
	entries []*models.AuditEntry
}

func (l *demoAuditLogger) Log(ctx context.Context, entry *models.AuditEntry) error {
	l.entries = append(l.entries, entry)
	return nil
}

type demoNotifier struct{}

func (n *demoNotifier) NotifyReviewRequired(ctx context.Context, item *models.KnowledgeItem, reviewerRoles []string) error {
	return nil
}

func (n *demoNotifier) NotifyApprovalRequired(ctx context.Context, item *models.KnowledgeItem, approverRole string) error {
	fmt.Printf("   %s📧 Notification sent to %s for approval%s\n", Cyan, approverRole, Reset)
	return nil
}

func (n *demoNotifier) NotifySLABreach(ctx context.Context, item *models.KnowledgeItem, breachType string) error {
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// OUTPUT HELPERS
// ═══════════════════════════════════════════════════════════════════════════════

func printSection(title string) {
	fmt.Printf("\n%s%s═══ %s ═══%s\n", Bold, Magenta, title, Reset)
}

func printStep(num int, title, actor string) {
	fmt.Printf("\n%s%s┌─────────────────────────────────────────────────────────────────────────────┐%s\n", Bold, Blue, Reset)
	fmt.Printf("%s%s│ STEP %d: %-67s │%s\n", Bold, Blue, num, title, Reset)
	fmt.Printf("%s%s│ Actor: %-68s │%s\n", Bold, Blue, actor, Reset)
	fmt.Printf("%s%s└─────────────────────────────────────────────────────────────────────────────┘%s\n", Bold, Blue, Reset)
}

func printDrugInfo(item *models.KnowledgeItem) {
	fmt.Printf("\n   %s💊 Drug:%s %s\n", Bold, Reset, item.Name)
	fmt.Printf("   %s📦 Type:%s %s\n", Bold, Reset, item.Type)
	fmt.Printf("   %s🏷️  KB:%s %s\n", Bold, Reset, item.KB)
	fmt.Printf("   %s🔐 Content Hash:%s %s...\n", Bold, Reset, item.ContentHash[:16])

	fmt.Printf("   %s%s⚠️  HIGH-ALERT MEDICATION (RequiresDualReview: %v)%s\n", Bold, Red, item.RequiresDualReview, Reset)

	fmt.Printf("\n   %s📄 Source:%s %s\n", Bold, Reset, item.Source.Authority)
	fmt.Printf("   %s🔗 Document:%s %s\n", Bold, Reset, item.Source.Document)
}

func handleResult(result *workflow.TransitionResult, err error, step string) {
	if err != nil {
		fmt.Printf("   %s❌ %s FAILED: %v%s\n", Red, step, err, Reset)
		return
	}

	fmt.Printf("   %s✅ %s → %s%s\n", Green, result.PreviousState, result.NewState, Reset)
	fmt.Printf("   %s📝 %s%s\n", Yellow, result.Message, Reset)
}

func printGovernanceTrail(item *models.KnowledgeItem) {
	gov := item.Governance

	fmt.Printf("\n   %s📋 Item:%s %s\n", Bold, Reset, item.ID)
	fmt.Printf("   %s📊 Final State:%s %s%s%s\n", Bold, Reset, Green, item.State, Reset)
	fmt.Printf("   %s📅 Activated:%s %v\n", Bold, Reset, item.ActiveAt)

	fmt.Printf("\n   %s👥 REVIEWS (%d):%s\n", Bold, len(gov.Reviews), Reset)
	for i, r := range gov.Reviews {
		fmt.Printf("      %d. [%s] %s (%s)\n", i+1, r.ReviewType, r.ReviewerName, r.Credentials)
		fmt.Printf("         Decision: %s%s%s | %s\n", Green, r.Decision, Reset, r.ReviewedAt.Format("2006-01-02 15:04"))
		if r.Notes != "" {
			fmt.Printf("         Notes: %s\n", truncate(r.Notes, 70))
		}
	}

	if gov.Approval != nil {
		fmt.Printf("\n   %s✍️  APPROVAL:%s\n", Bold, Reset)
		fmt.Printf("      Approver: %s (%s)\n", gov.Approval.ApproverName, gov.Approval.Credentials)
		fmt.Printf("      Decision: %s%s%s | %s\n", Green, gov.Approval.Decision, Reset, gov.Approval.ApprovedAt.Format("2006-01-02 15:04"))
		if len(gov.Approval.Attestations) > 0 {
			fmt.Printf("      Attestations: ")
			first := true
			for k, v := range gov.Approval.Attestations {
				if !first {
					fmt.Printf(", ")
				}
				fmt.Printf("%s=%v", k, v)
				first = false
			}
			fmt.Println()
		}
	}

	if gov.ActivatedAt != nil {
		fmt.Printf("\n   %s🚀 ACTIVATION:%s\n", Bold, Reset)
		fmt.Printf("      Activated: %s\n", gov.ActivatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("      By: %s\n", gov.ActivatedBy)
	}
}

func printAuditLog(entries []*models.AuditEntry) {
	fmt.Println()
	for i, e := range entries {
		icon := "📝"
		switch e.Action {
		case models.AuditItemReviewed:
			icon = "🔍"
		case models.AuditItemApproved:
			icon = "✅"
		case models.AuditItemActivated:
			icon = "🚀"
		}

		fmt.Printf("   %s %d. [%s] %s → %s\n", icon, i+1, e.Action, e.PreviousState, e.NewState)
		fmt.Printf("      Actor: %s (%s) | %s\n", e.ActorName, e.ActorRole, e.Timestamp.Format("15:04:05"))
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
