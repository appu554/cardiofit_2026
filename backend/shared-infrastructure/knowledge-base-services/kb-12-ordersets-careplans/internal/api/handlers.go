// Package api provides the HTTP API server for KB-12
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kb-12-ordersets-careplans/internal/models"
)

// Template Handlers

// listTemplates handles GET /api/v1/templates
func (s *Server) listTemplates(c *gin.Context) {
	ctx := c.Request.Context()

	category := c.Query("category")
	active := c.DefaultQuery("active", "true")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	query := s.db.WithContext(ctx).Model(&models.OrderSetTemplate{})

	if category != "" {
		query = query.Where("category = ?", category)
	}
	if active == "true" {
		query = query.Where("active = ?", true)
	}

	var total int64
	query.Count(&total)

	var templates []models.OrderSetTemplate
	if err := query.Limit(limit).Offset(offset).Order("name ASC").Find(&templates).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch templates", "DB_ERROR")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"templates": templates,
			"total":     total,
			"limit":     limit,
			"offset":    offset,
		},
	})
}

// getTemplate handles GET /api/v1/templates/:id
func (s *Server) getTemplate(c *gin.Context) {
	ctx := c.Request.Context()
	templateID := c.Param("id")

	// Try cache first
	if cached, err := s.cache.GetOrderSetTemplate(ctx, templateID); err == nil {
		var template models.OrderSetTemplate
		if json.Unmarshal(cached, &template) == nil {
			respondSuccess(c, template)
			return
		}
	}

	var template models.OrderSetTemplate
	if err := s.db.WithContext(ctx).Where("template_id = ?", templateID).First(&template).Error; err != nil {
		respondError(c, http.StatusNotFound, "Template not found", "NOT_FOUND")
		return
	}

	// Cache the result
	if data, err := json.Marshal(template); err == nil {
		s.cache.SetOrderSetTemplate(ctx, templateID, data)
	}

	respondSuccess(c, template)
}

// searchTemplates handles GET /api/v1/templates/search
func (s *Server) searchTemplates(c *gin.Context) {
	ctx := c.Request.Context()

	q := c.Query("q")
	category := c.Query("category")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	query := s.db.WithContext(ctx).Model(&models.OrderSetTemplate{}).Where("active = ?", true)

	if q != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+q+"%", "%"+q+"%")
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}

	var templates []models.OrderSetTemplate
	if err := query.Limit(limit).Order("name ASC").Find(&templates).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "Search failed", "DB_ERROR")
		return
	}

	respondSuccess(c, templates)
}

// getTemplatesByCategory handles GET /api/v1/templates/category/:category
func (s *Server) getTemplatesByCategory(c *gin.Context) {
	ctx := c.Request.Context()
	category := c.Param("category")

	var templates []models.OrderSetTemplate
	if err := s.db.WithContext(ctx).Where("category = ? AND active = ?", category, true).Find(&templates).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch templates", "DB_ERROR")
		return
	}

	respondSuccess(c, templates)
}

// Instance Handlers

// activateOrderSet handles POST /api/v1/activate
func (s *Server) activateOrderSet(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.OrderSetActivationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	// Get template
	var template models.OrderSetTemplate
	if err := s.db.WithContext(ctx).Where("template_id = ? AND active = ?", req.TemplateID, true).First(&template).Error; err != nil {
		respondError(c, http.StatusNotFound, "Template not found", "NOT_FOUND")
		return
	}

	// Get orders from template
	orders, err := template.GetOrders()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to parse template orders", "PARSE_ERROR")
		return
	}

	// Filter selected orders if specified
	if len(req.SelectedOrders) > 0 {
		selectedMap := make(map[string]bool)
		for _, id := range req.SelectedOrders {
			selectedMap[id] = true
		}
		var filtered []models.Order
		for _, order := range orders {
			if selectedMap[order.OrderID] || order.Required {
				order.Selected = true
				filtered = append(filtered, order)
			}
		}
		orders = filtered
	} else {
		// Select all orders by default
		for i := range orders {
			orders[i].Selected = true
		}
	}

	// Create instance
	instance := models.OrderSetInstance{
		TemplateID:  req.TemplateID,
		PatientID:   req.PatientID,
		EncounterID: req.EncounterID,
		ActivatedBy: req.ActivatedBy,
		Status:      models.OrderStatusActive,
		ActivatedAt: time.Now(),
	}

	if err := instance.SetOrders(orders); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to set orders", "INTERNAL_ERROR")
		return
	}

	// Initialize constraint status for time-critical protocols
	constraints, _ := template.GetTimeConstraints()
	if len(constraints) > 0 {
		var statuses []models.ConstraintStatus
		for _, c := range constraints {
			statuses = append(statuses, models.ConstraintStatus{
				ConstraintID:    c.ConstraintID,
				Action:          c.Action,
				Status:          "pending",
				StartTime:       instance.ActivatedAt,
				Deadline:        instance.ActivatedAt.Add(c.Deadline),
				Severity:        c.Severity,
				PercentComplete: 0,
			})
		}
		instance.SetConstraintStatus(statuses)
	}

	if err := s.db.WithContext(ctx).Create(&instance).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to create instance", "DB_ERROR")
		return
	}

	// Invalidate patient cache
	s.cache.InvalidatePatientOrderSets(ctx, req.PatientID)

	response := models.OrderSetActivationResponse{
		Success:      true,
		InstanceID:   instance.InstanceID,
		TemplateID:   template.TemplateID,
		TemplateName: template.Name,
		PatientID:    req.PatientID,
		EncounterID:  req.EncounterID,
		Orders:       orders,
		ActivatedAt:  instance.ActivatedAt,
	}

	if statuses, _ := instance.GetConstraintStatus(); len(statuses) > 0 {
		response.Constraints = statuses
	}

	respondCreated(c, response)
}

// listInstances handles GET /api/v1/instances
func (s *Server) listInstances(c *gin.Context) {
	ctx := c.Request.Context()

	patientID := c.Query("patient_id")
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	query := s.db.WithContext(ctx).Model(&models.OrderSetInstance{})

	if patientID != "" {
		query = query.Where("patient_id = ?", patientID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var instances []models.OrderSetInstance
	if err := query.Limit(limit).Offset(offset).Order("activated_at DESC").Find(&instances).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch instances", "DB_ERROR")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"instances": instances,
			"total":     total,
			"limit":     limit,
			"offset":    offset,
		},
	})
}

// getInstance handles GET /api/v1/instances/:id
func (s *Server) getInstance(c *gin.Context) {
	ctx := c.Request.Context()
	instanceID := c.Param("id")

	var instance models.OrderSetInstance
	if err := s.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&instance).Error; err != nil {
		respondError(c, http.StatusNotFound, "Instance not found", "NOT_FOUND")
		return
	}

	respondSuccess(c, instance)
}

// updateInstanceStatus handles PUT /api/v1/instances/:id/status
func (s *Server) updateInstanceStatus(c *gin.Context) {
	ctx := c.Request.Context()
	instanceID := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	var instance models.OrderSetInstance
	if err := s.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&instance).Error; err != nil {
		respondError(c, http.StatusNotFound, "Instance not found", "NOT_FOUND")
		return
	}

	instance.Status = models.OrderStatus(req.Status)
	if req.Status == string(models.OrderStatusCompleted) {
		now := time.Now()
		instance.CompletedAt = &now
	}

	if err := s.db.WithContext(ctx).Save(&instance).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to update instance", "DB_ERROR")
		return
	}

	s.cache.InvalidateOrderSetInstance(ctx, instanceID)
	s.cache.InvalidatePatientOrderSets(ctx, instance.PatientID)

	respondSuccess(c, instance)
}

// getInstanceConstraints handles GET /api/v1/instances/:id/constraints
func (s *Server) getInstanceConstraints(c *gin.Context) {
	ctx := c.Request.Context()
	instanceID := c.Param("id")

	var instance models.OrderSetInstance
	if err := s.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&instance).Error; err != nil {
		respondError(c, http.StatusNotFound, "Instance not found", "NOT_FOUND")
		return
	}

	statuses, err := instance.GetConstraintStatus()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to parse constraints", "PARSE_ERROR")
		return
	}

	// Update time remaining for each constraint
	now := time.Now()
	for i := range statuses {
		if statuses[i].CompletedAt == nil {
			statuses[i].TimeElapsed = now.Sub(statuses[i].StartTime)
			statuses[i].TimeRemaining = statuses[i].Deadline.Sub(now)
			if statuses[i].TimeRemaining < 0 {
				statuses[i].TimeRemaining = 0
				statuses[i].Status = "overdue"
			}
			deadline := statuses[i].Deadline.Sub(statuses[i].StartTime)
			if deadline > 0 {
				statuses[i].PercentComplete = float64(statuses[i].TimeElapsed) / float64(deadline) * 100
				if statuses[i].PercentComplete > 100 {
					statuses[i].PercentComplete = 100
				}
			}
		}
	}

	respondSuccess(c, statuses)
}

// updateOrderStatus handles PUT /api/v1/instances/:id/order/:order_id
func (s *Server) updateOrderStatus(c *gin.Context) {
	ctx := c.Request.Context()
	instanceID := c.Param("id")
	orderID := c.Param("order_id")

	var req struct {
		Status string `json:"status" binding:"required"`
		Notes  string `json:"notes,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	var instance models.OrderSetInstance
	if err := s.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&instance).Error; err != nil {
		respondError(c, http.StatusNotFound, "Instance not found", "NOT_FOUND")
		return
	}

	orders, err := instance.GetOrders()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to parse orders", "PARSE_ERROR")
		return
	}

	found := false
	for i := range orders {
		if orders[i].OrderID == orderID {
			found = true
			// Update order status (simplified - in production would track status properly)
			if req.Notes != "" {
				orders[i].Notes = req.Notes
			}
			break
		}
	}

	if !found {
		respondError(c, http.StatusNotFound, "Order not found", "NOT_FOUND")
		return
	}

	if err := instance.SetOrders(orders); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to update orders", "INTERNAL_ERROR")
		return
	}

	if err := s.db.WithContext(ctx).Save(&instance).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to save instance", "DB_ERROR")
		return
	}

	s.cache.InvalidateOrderSetInstance(ctx, instanceID)

	respondSuccess(c, gin.H{"order_id": orderID, "status": req.Status})
}

// Care Plan Handlers

// listCarePlanTemplates handles GET /api/v1/careplans
func (s *Server) listCarePlanTemplates(c *gin.Context) {
	ctx := c.Request.Context()

	condition := c.Query("condition")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	query := s.db.WithContext(ctx).Model(&models.CarePlanTemplate{}).Where("active = ?", true)

	if condition != "" {
		query = query.Where("condition ILIKE ?", "%"+condition+"%")
	}

	var total int64
	query.Count(&total)

	var templates []models.CarePlanTemplate
	if err := query.Limit(limit).Offset(offset).Order("name ASC").Find(&templates).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch care plans", "DB_ERROR")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"careplans": templates,
			"total":     total,
			"limit":     limit,
			"offset":    offset,
		},
	})
}

// getCarePlanTemplate handles GET /api/v1/careplans/:id
func (s *Server) getCarePlanTemplate(c *gin.Context) {
	ctx := c.Request.Context()
	planID := c.Param("id")

	var template models.CarePlanTemplate
	if err := s.db.WithContext(ctx).Where("plan_id = ?", planID).First(&template).Error; err != nil {
		respondError(c, http.StatusNotFound, "Care plan template not found", "NOT_FOUND")
		return
	}

	respondSuccess(c, template)
}

// activateCarePlan handles POST /api/v1/careplans
func (s *Server) activateCarePlan(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.CarePlanActivationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	var template models.CarePlanTemplate
	if err := s.db.WithContext(ctx).Where("plan_id = ? AND active = ?", req.PlanID, true).First(&template).Error; err != nil {
		respondError(c, http.StatusNotFound, "Care plan template not found", "NOT_FOUND")
		return
	}

	startDate := req.StartDate
	if startDate.IsZero() {
		startDate = time.Now()
	}

	instance := models.CarePlanInstance{
		TemplateID: req.PlanID,
		PatientID:  req.PatientID,
		Status:     models.CarePlanStatusActive,
		StartDate:  startDate,
		EndDate:    req.EndDate,
	}

	// Initialize goals progress
	goals, _ := template.GetGoals()
	var goalsProgress []models.GoalProgress
	for _, g := range goals {
		goalsProgress = append(goalsProgress, models.GoalProgress{
			GoalID:      g.GoalID,
			Status:      models.GoalStatusInProgress,
			ProgressPct: 0,
		})
	}
	instance.SetGoalsProgress(goalsProgress)

	// Initialize activities completed
	activities, _ := template.GetActivities()
	var activitiesCompleted []models.ActivityCompletion
	for _, a := range activities {
		activitiesCompleted = append(activitiesCompleted, models.ActivityCompletion{
			ActivityID:      a.ActivityID,
			Status:          models.ActivityStatusScheduled,
			CompletionCount: 0,
		})
	}
	instance.SetActivitiesCompleted(activitiesCompleted)

	if err := s.db.WithContext(ctx).Create(&instance).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to create care plan instance", "DB_ERROR")
		return
	}

	s.cache.InvalidatePatientCarePlans(ctx, req.PatientID)

	monitoring, _ := template.GetMonitoringItems()

	response := models.CarePlanActivationResponse{
		Success:         true,
		InstanceID:      instance.InstanceID,
		PlanID:          template.PlanID,
		PlanName:        template.Name,
		PatientID:       req.PatientID,
		Status:          instance.Status,
		Goals:           goals,
		Activities:      activities,
		MonitoringItems: monitoring,
		StartDate:       instance.StartDate,
		EndDate:         instance.EndDate,
	}

	respondCreated(c, response)
}

// Care Plan Instance Handlers

// listCarePlanInstances handles GET /api/v1/careplan-instances
func (s *Server) listCarePlanInstances(c *gin.Context) {
	ctx := c.Request.Context()

	patientID := c.Query("patient_id")
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	query := s.db.WithContext(ctx).Model(&models.CarePlanInstance{})

	if patientID != "" {
		query = query.Where("patient_id = ?", patientID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var instances []models.CarePlanInstance
	if err := query.Limit(limit).Offset(offset).Order("start_date DESC").Find(&instances).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch care plan instances", "DB_ERROR")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"instances": instances,
			"total":     total,
			"limit":     limit,
			"offset":    offset,
		},
	})
}

// getCarePlanInstance handles GET /api/v1/careplan-instances/:id
func (s *Server) getCarePlanInstance(c *gin.Context) {
	ctx := c.Request.Context()
	instanceID := c.Param("id")

	var instance models.CarePlanInstance
	if err := s.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&instance).Error; err != nil {
		respondError(c, http.StatusNotFound, "Care plan instance not found", "NOT_FOUND")
		return
	}

	respondSuccess(c, instance)
}

// updateCarePlanStatus handles PUT /api/v1/careplan-instances/:id/status
func (s *Server) updateCarePlanStatus(c *gin.Context) {
	ctx := c.Request.Context()
	instanceID := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	var instance models.CarePlanInstance
	if err := s.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&instance).Error; err != nil {
		respondError(c, http.StatusNotFound, "Care plan instance not found", "NOT_FOUND")
		return
	}

	instance.Status = models.CarePlanStatus(req.Status)
	if req.Status == string(models.CarePlanStatusCompleted) {
		now := time.Now()
		instance.EndDate = &now
	}

	if err := s.db.WithContext(ctx).Save(&instance).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to update care plan", "DB_ERROR")
		return
	}

	s.cache.InvalidateCarePlanInstance(ctx, instanceID)
	s.cache.InvalidatePatientCarePlans(ctx, instance.PatientID)

	respondSuccess(c, instance)
}

// updateCarePlanProgress handles PUT /api/v1/careplan-instances/:id/progress
func (s *Server) updateCarePlanProgress(c *gin.Context) {
	ctx := c.Request.Context()
	instanceID := c.Param("id")

	var req struct {
		Goals      []models.GoalProgress      `json:"goals,omitempty"`
		Activities []models.ActivityCompletion `json:"activities,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	var instance models.CarePlanInstance
	if err := s.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&instance).Error; err != nil {
		respondError(c, http.StatusNotFound, "Care plan instance not found", "NOT_FOUND")
		return
	}

	if len(req.Goals) > 0 {
		instance.SetGoalsProgress(req.Goals)
	}
	if len(req.Activities) > 0 {
		instance.SetActivitiesCompleted(req.Activities)
	}

	if err := s.db.WithContext(ctx).Save(&instance).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to update progress", "DB_ERROR")
		return
	}

	s.cache.InvalidateCarePlanInstance(ctx, instanceID)

	respondSuccess(c, instance)
}

// getCarePlanGoals handles GET /api/v1/careplan-instances/:id/goals
func (s *Server) getCarePlanGoals(c *gin.Context) {
	ctx := c.Request.Context()
	instanceID := c.Param("id")

	var instance models.CarePlanInstance
	if err := s.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&instance).Error; err != nil {
		respondError(c, http.StatusNotFound, "Care plan instance not found", "NOT_FOUND")
		return
	}

	progress, err := instance.GetGoalsProgress()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to parse goals", "PARSE_ERROR")
		return
	}

	respondSuccess(c, progress)
}

// updateGoalProgress handles PUT /api/v1/careplan-instances/:id/goals/:goal_id
func (s *Server) updateGoalProgress(c *gin.Context) {
	ctx := c.Request.Context()
	instanceID := c.Param("id")
	goalID := c.Param("goal_id")

	var req models.GoalProgress
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	var instance models.CarePlanInstance
	if err := s.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&instance).Error; err != nil {
		respondError(c, http.StatusNotFound, "Care plan instance not found", "NOT_FOUND")
		return
	}

	progress, _ := instance.GetGoalsProgress()
	found := false
	for i := range progress {
		if progress[i].GoalID == goalID {
			found = true
			progress[i] = req
			break
		}
	}

	if !found {
		respondError(c, http.StatusNotFound, "Goal not found", "NOT_FOUND")
		return
	}

	instance.SetGoalsProgress(progress)

	if err := s.db.WithContext(ctx).Save(&instance).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to update goal", "DB_ERROR")
		return
	}

	s.cache.InvalidateCarePlanInstance(ctx, instanceID)

	respondSuccess(c, req)
}

// getCarePlanActivities handles GET /api/v1/careplan-instances/:id/activities
func (s *Server) getCarePlanActivities(c *gin.Context) {
	ctx := c.Request.Context()
	instanceID := c.Param("id")

	var instance models.CarePlanInstance
	if err := s.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&instance).Error; err != nil {
		respondError(c, http.StatusNotFound, "Care plan instance not found", "NOT_FOUND")
		return
	}

	completions, err := instance.GetActivitiesCompleted()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to parse activities", "PARSE_ERROR")
		return
	}

	respondSuccess(c, completions)
}

// updateActivityStatus handles PUT /api/v1/careplan-instances/:id/activities/:activity_id
func (s *Server) updateActivityStatus(c *gin.Context) {
	ctx := c.Request.Context()
	instanceID := c.Param("id")
	activityID := c.Param("activity_id")

	var req models.ActivityCompletion
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	var instance models.CarePlanInstance
	if err := s.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&instance).Error; err != nil {
		respondError(c, http.StatusNotFound, "Care plan instance not found", "NOT_FOUND")
		return
	}

	completions, _ := instance.GetActivitiesCompleted()
	found := false
	for i := range completions {
		if completions[i].ActivityID == activityID {
			found = true
			completions[i] = req
			break
		}
	}

	if !found {
		respondError(c, http.StatusNotFound, "Activity not found", "NOT_FOUND")
		return
	}

	instance.SetActivitiesCompleted(completions)

	if err := s.db.WithContext(ctx).Save(&instance).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to update activity", "DB_ERROR")
		return
	}

	s.cache.InvalidateCarePlanInstance(ctx, instanceID)

	respondSuccess(c, req)
}

// Placeholder handlers for remaining endpoints

func (s *Server) listConstraints(c *gin.Context)       { respondSuccess(c, []interface{}{}) }
func (s *Server) evaluateConstraints(c *gin.Context)   { respondSuccess(c, gin.H{"evaluated": true}) }
func (s *Server) getOverdueConstraints(c *gin.Context) { respondSuccess(c, []interface{}{}) }

func (s *Server) getFHIRBundle(c *gin.Context)             { respondSuccess(c, gin.H{"resourceType": "Bundle"}) }
func (s *Server) getFHIRPlanDefinition(c *gin.Context)     { respondSuccess(c, gin.H{"resourceType": "PlanDefinition"}) }
func (s *Server) getFHIRCarePlan(c *gin.Context)           { respondSuccess(c, gin.H{"resourceType": "CarePlan"}) }
func (s *Server) getFHIRMedicationRequest(c *gin.Context)  { respondSuccess(c, gin.H{"resourceType": "MedicationRequest"}) }
func (s *Server) getFHIRServiceRequest(c *gin.Context)     { respondSuccess(c, gin.H{"resourceType": "ServiceRequest"}) }

func (s *Server) submitOrders(c *gin.Context)        { respondSuccess(c, gin.H{"submitted": true}) }
func (s *Server) createDraftSession(c *gin.Context)  { respondCreated(c, gin.H{"session_id": uuid.New().String()}) }
func (s *Server) getDraftSession(c *gin.Context)     { respondSuccess(c, gin.H{}) }
func (s *Server) updateDraftSession(c *gin.Context)  { respondSuccess(c, gin.H{"updated": true}) }
func (s *Server) submitDraftSession(c *gin.Context)  { respondSuccess(c, gin.H{"submitted": true}) }
func (s *Server) cancelDraftSession(c *gin.Context)  { respondSuccess(c, gin.H{"cancelled": true}) }
func (s *Server) performSafetyCheck(c *gin.Context)  { respondSuccess(c, gin.H{"safe": true, "alerts": []interface{}{}}) }

func (s *Server) getPatientOrderSets(c *gin.Context)   { respondSuccess(c, []interface{}{}) }
func (s *Server) getPatientCarePlans(c *gin.Context)   { respondSuccess(c, []interface{}{}) }
func (s *Server) getPatientOrders(c *gin.Context)      { respondSuccess(c, []interface{}{}) }
func (s *Server) getPatientConstraints(c *gin.Context) { respondSuccess(c, []interface{}{}) }

func (s *Server) listTasks(c *gin.Context)           { respondSuccess(c, []interface{}{}) }
func (s *Server) getTask(c *gin.Context)             { respondSuccess(c, gin.H{}) }
func (s *Server) updateTaskStatus(c *gin.Context)    { respondSuccess(c, gin.H{"updated": true}) }
func (s *Server) getOverdueTasks(c *gin.Context)     { respondSuccess(c, []interface{}{}) }
func (s *Server) getUpcomingDeadlines(c *gin.Context) { respondSuccess(c, []interface{}{}) }

func (s *Server) getCDSServices(c *gin.Context)          { respondSuccess(c, gin.H{"services": []interface{}{}}) }
func (s *Server) orderSelectHook(c *gin.Context)         { respondSuccess(c, gin.H{"cards": []interface{}{}}) }
func (s *Server) orderSignHook(c *gin.Context)           { respondSuccess(c, gin.H{"cards": []interface{}{}}) }
func (s *Server) patientViewHook(c *gin.Context)         { respondSuccess(c, gin.H{"cards": []interface{}{}}) }
func (s *Server) encounterStartHook(c *gin.Context)      { respondSuccess(c, gin.H{"cards": []interface{}{}}) }
func (s *Server) encounterDischargeHook(c *gin.Context)  { respondSuccess(c, gin.H{"cards": []interface{}{}}) }
