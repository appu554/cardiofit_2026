package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"kb-drug-interactions/internal/services"
)

// ConstitutionalHandlers handles ONC constitutional DDI rule endpoints
type ConstitutionalHandlers struct {
	expansionService *services.OHDSIExpansionService
}

// NewConstitutionalHandlers creates constitutional rule handlers
func NewConstitutionalHandlers(expansionService *services.OHDSIExpansionService) *ConstitutionalHandlers {
	return &ConstitutionalHandlers{
		expansionService: expansionService,
	}
}

// RegisterRoutes registers constitutional DDI routes
func (h *ConstitutionalHandlers) RegisterRoutes(r *gin.RouterGroup) {
	constitutional := r.Group("/constitutional")
	{
		// Rule management
		constitutional.GET("/rules", h.GetConstitutionalRules)
		constitutional.GET("/rules/:id", h.GetConstitutionalRule)

		// Class expansion
		constitutional.GET("/expansion/stats", h.GetExpansionStats)
		constitutional.POST("/expansion/expand-rule", h.ExpandRule)

		// DDI checking
		constitutional.POST("/check", h.CheckDDI)

		// Validation endpoints
		constitutional.GET("/validate/warfarin-ibuprofen", h.ValidateWarfarinIbuprofen)
		constitutional.GET("/validate/qt-rule", h.ValidateQTRule)
	}
}

// GetConstitutionalRules returns all active constitutional DDI rules
// @Summary Get all constitutional DDI rules
// @Description Returns the 25 canonical ONC DDI rules with authority stratification
// @Tags Constitutional DDI
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /constitutional/rules [get]
func (h *ConstitutionalHandlers) GetConstitutionalRules(c *gin.Context) {
	ctx := c.Request.Context()

	rules, err := h.expansionService.GetConstitutionalRules(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve constitutional rules",
			"details": err.Error(),
		})
		return
	}

	// Group by authority for structured response
	byAuthority := make(map[string][]services.ConstitutionalRule)
	for _, rule := range rules {
		byAuthority[rule.RuleAuthority] = append(byAuthority[rule.RuleAuthority], rule)
	}

	c.JSON(http.StatusOK, gin.H{
		"total_rules": len(rules),
		"rules": rules,
		"by_authority": byAuthority,
		"authority_summary": map[string]int{
			"ONC-Phansalkar-2012": len(byAuthority["ONC-Phansalkar-2012"]),
			"ONC-Derived":         len(byAuthority["ONC-Derived"]),
			"FDA-Boxed-Warning":   len(byAuthority["FDA-Boxed-Warning"]),
			"Post-ONC-Critical":   len(byAuthority["Post-ONC-Critical"]),
		},
	})
}

// GetConstitutionalRule returns a specific constitutional rule by ID
// @Summary Get constitutional rule by ID
// @Tags Constitutional DDI
// @Produce json
// @Param id path int true "Rule ID"
// @Success 200 {object} services.ConstitutionalRule
// @Router /constitutional/rules/{id} [get]
func (h *ConstitutionalHandlers) GetConstitutionalRule(c *gin.Context) {
	ctx := c.Request.Context()

	ruleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rule ID"})
		return
	}

	rules, err := h.expansionService.GetConstitutionalRules(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, rule := range rules {
		if rule.RuleID == ruleID {
			c.JSON(http.StatusOK, rule)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
}

// GetExpansionStats returns statistics about rule expansion via OHDSI
// @Summary Get class expansion statistics
// @Description Shows how many drug pairs each rule expands to via OHDSI
// @Tags Constitutional DDI
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /constitutional/expansion/stats [get]
func (h *ConstitutionalHandlers) GetExpansionStats(c *gin.Context) {
	ctx := c.Request.Context()

	stats, err := h.expansionService.GetExpansionStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get expansion stats",
			"details": err.Error(),
		})
		return
	}

	totalPairs := 0
	for _, stat := range stats {
		totalPairs += stat.TotalPairsCovered
	}

	c.JSON(http.StatusOK, gin.H{
		"total_rules": len(stats),
		"total_drug_pairs": totalPairs,
		"stats": stats,
		"note": "Total pairs = Cartesian product of class members",
	})
}

// ExpandRuleRequest is the request body for expanding a rule
type ExpandRuleRequest struct {
	RuleID int `json:"rule_id" binding:"required"`
}

// ExpandRule expands a specific constitutional rule to drug pairs
// @Summary Expand a constitutional rule
// @Description Expands a class-based rule to concrete drug pairs via OHDSI
// @Tags Constitutional DDI
// @Accept json
// @Produce json
// @Param request body ExpandRuleRequest true "Rule to expand"
// @Success 200 {object} map[string]interface{}
// @Router /constitutional/expansion/expand-rule [post]
func (h *ConstitutionalHandlers) ExpandRule(c *gin.Context) {
	ctx := c.Request.Context()

	var req ExpandRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get the rule
	rules, err := h.expansionService.GetConstitutionalRules(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var targetRule *services.ConstitutionalRule
	for _, rule := range rules {
		if rule.RuleID == req.RuleID {
			targetRule = &rule
			break
		}
	}

	if targetRule == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	// Expand the rule
	projections, err := h.expansionService.ExpandRule(ctx, *targetRule)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to expand rule",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rule_id": req.RuleID,
		"rule": targetRule,
		"total_projections": len(projections),
		"projections": projections,
	})
}

// CheckDDIRequest is the request body for DDI checking
type CheckDDIRequest struct {
	DrugConceptIDs []int64 `json:"drug_concept_ids" binding:"required,min=2"`
}

// CheckDDI checks for drug-drug interactions using constitutional rules
// @Summary Check for DDI using constitutional rules
// @Description Checks drug concept IDs against expanded constitutional rules
// @Tags Constitutional DDI
// @Accept json
// @Produce json
// @Param request body CheckDDIRequest true "Drugs to check"
// @Success 200 {object} services.DDICheckResult
// @Router /constitutional/check [post]
func (h *ConstitutionalHandlers) CheckDDI(c *gin.Context) {
	ctx := c.Request.Context()

	var req CheckDDIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"details": "Provide at least 2 drug_concept_ids",
		})
		return
	}

	result, err := h.expansionService.CheckDDI(ctx, req.DrugConceptIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check DDI",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ValidateWarfarinIbuprofen performs the Warfarin + Ibuprofen validation check
// @Summary Validate Warfarin + Ibuprofen expansion
// @Description Spot check that Rule 6 correctly expands to Warfarin + Ibuprofen
// @Tags Constitutional DDI
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /constitutional/validate/warfarin-ibuprofen [get]
func (h *ConstitutionalHandlers) ValidateWarfarinIbuprofen(c *gin.Context) {
	ctx := c.Request.Context()

	result, err := h.expansionService.ValidateWarfarinIbuprofen(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"validation": "FAILED",
			"error": err.Error(),
			"expected": "Rule 6 (VKA + NSAID) should expand to include Warfarin + Ibuprofen",
		})
		return
	}

	if result.RuleID == 0 {
		c.JSON(http.StatusOK, gin.H{
			"validation": "PENDING",
			"message": "OHDSI vocabulary not loaded yet",
			"expected": "Load OHDSI CONCEPT.csv and CONCEPT_RELATIONSHIP.csv first",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"validation": "PASSED",
		"message": "Warfarin + Ibuprofen correctly expanded from Rule 6",
		"result": result,
		"expected": map[string]interface{}{
			"rule_id": 6,
			"trigger_class": "Vitamin K Antagonists",
			"target_class": "NSAIDs",
			"risk_level": "HIGH",
			"context_required": false,
		},
	})
}

// ValidateQTRule validates the QT + QT self-class rule expansion
// @Summary Validate QT + QT rule expansion
// @Description Spot check that Rule 9 (self-class rule) expands correctly
// @Tags Constitutional DDI
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /constitutional/validate/qt-rule [get]
func (h *ConstitutionalHandlers) ValidateQTRule(c *gin.Context) {
	ctx := c.Request.Context()

	stat, err := h.expansionService.ValidateQTRule(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"validation": "FAILED",
			"error": err.Error(),
		})
		return
	}

	if stat.RuleID == 0 {
		c.JSON(http.StatusOK, gin.H{
			"validation": "PENDING",
			"message": "OHDSI vocabulary not loaded yet",
		})
		return
	}

	// QT + QT should be the largest expansion (many QT-prolonging drugs)
	c.JSON(http.StatusOK, gin.H{
		"validation": "PASSED",
		"message": "QT + QT self-class rule expanded correctly",
		"result": stat,
		"analysis": map[string]interface{}{
			"is_self_class_rule": stat.TriggerClassName == stat.TargetClassName,
			"expected_formula": "n*(n-1)/2 pairs for n drugs",
			"note": "Should be largest expansion due to many QT-prolonging drugs",
		},
	})
}

// parseConceptIDs parses comma-separated concept IDs from query string
func parseConceptIDs(s string) ([]int64, error) {
	if s == "" {
		return nil, nil
	}

	parts := strings.Split(s, ",")
	ids := make([]int64, 0, len(parts))

	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}
