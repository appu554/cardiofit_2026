package services

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"text/template"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// MaskedHTNCard represents a decision card for masked/white-coat hypertension phenotypes.
type MaskedHTNCard struct {
	CardType       string                `json:"card_type"`
	Urgency        string                `json:"urgency"`
	Title          string                `json:"title"`
	Rationale      string                `json:"rationale"`
	Actions        []string              `json:"actions"`
	ConfidenceTier models.ConfidenceTier `json:"confidence_tier"`
}

// confidenceStringToTier maps the BP context classifier's string-based
// confidence ("HIGH"/"MODERATE"/"LOW"/"DAMPED") to KB-23's ConfidenceTier
// enum. Used only by masked HTN cards — existing HTN safety templates
// retain bypasses_confidence_gate=true and are unchanged.
func confidenceStringToTier(confidence string) models.ConfidenceTier {
	switch confidence {
	case "HIGH":
		return models.TierFirm
	case "MODERATE":
		return models.TierProbable
	case "LOW":
		return models.TierPossible
	case "DAMPED":
		return models.TierUncertain
	default:
		return models.TierUncertain
	}
}

// MaskedHTNCardBuilder renders masked/white-coat cards using YAML templates
// for clinician/patient fragments and recommendations. When the loader
// dependencies are nil, the builder falls back to the hardcoded text so the
// API retains the pre-Phase-4 behaviour.
type MaskedHTNCardBuilder struct {
	templateLoader *TemplateLoader
	fragmentLoader *FragmentLoader
	log            *zap.Logger
}

// NewMaskedHTNCardBuilder constructs a builder backed by the provided loaders.
func NewMaskedHTNCardBuilder(loader *TemplateLoader, fragments *FragmentLoader, log *zap.Logger) *MaskedHTNCardBuilder {
	if log == nil {
		log = zap.NewNop()
	}
	return &MaskedHTNCardBuilder{
		templateLoader: loader,
		fragmentLoader: fragments,
		log:            log,
	}
}

// Evaluate builds the set of cards for a BP context classification using the
// YAML template fragments when available.
func (b *MaskedHTNCardBuilder) Evaluate(c *models.BPContextClassification) []MaskedHTNCard {
	if c == nil {
		return nil
	}

	baseData := newMaskedTemplateData(c)
	confidenceTier := confidenceStringToTier(c.Confidence)
	var cards []MaskedHTNCard

	// 1. Compound risk: masked HTN + morning surge — highest urgency.
	if c.MorningSurgeCompound &&
		(c.Phenotype == models.PhenotypeMaskedHTN || c.Phenotype == models.PhenotypeMaskedUncontrolled) {
		fallback := fmt.Sprintf(
			"Clinic BP %.0f/%.0f mmHg; home mean %.0f mmHg. Morning surge compounds masked "+
				"hypertension — combined risk exceeds either condition alone. Peak cardiovascular "+
				"event risk window (06:00-12:00) coincides with uncontrolled BP period.",
			c.ClinicSBPMean, c.ClinicDBPMean, c.HomeSBPMean)
		actions := []string{
			"24-hour ABPM to characterise morning surge amplitude",
			"Review medication timing — consider evening/bedtime dosing of long-acting agent",
			"Urgent cardiology review if home SBP >160 mmHg in morning window",
		}
		card := MaskedHTNCard{
			CardType:       "MASKED_HTN_MORNING_SURGE_COMPOUND",
			Urgency:        "IMMEDIATE",
			Title:          "Masked HTN + Morning Surge — Compound CV Risk",
			Rationale:      fallback,
			Actions:        actions,
			ConfidenceTier: confidenceTier,
		}
		cards = append(cards, b.applyTemplate(card, baseData))
	}

	// 1b. Sustained HTN + morning surge — URGENT (not IMMEDIATE)
	if c.MorningSurgeCompound && c.Phenotype == models.PhenotypeSustainedHTN {
		fallback := fmt.Sprintf(
			"Both clinic (%.0f) and home (%.0f) BP elevated with abnormal morning surge (>20 mmHg). "+
				"Morning surge on top of sustained hypertension significantly increases stroke risk "+
				"during the morning cardiovascular event window (06:00-12:00) — Kario 2019 (JACC).",
			c.ClinicSBPMean, c.HomeSBPMean)
		actions := []string{
			"Consider bedtime dosing of long-acting antihypertensive (chronotherapy)",
			"Evaluate for obstructive sleep apnea — strong association with exaggerated morning surge",
			"Prefer 24-hour-coverage agents: long-acting ARB or dihydropyridine CCB",
			"Consider 24-hour ABPM to characterise surge amplitude",
		}
		card := MaskedHTNCard{
			CardType:       "SUSTAINED_HTN_MORNING_SURGE",
			Urgency:        "URGENT",
			Title:          "Sustained Hypertension with Abnormal Morning Surge",
			Rationale:      fallback,
			Actions:        actions,
			ConfidenceTier: confidenceTier,
		}
		cards = append(cards, b.applyTemplate(card, baseData))
	}

	// 2. Masked hypertension phenotype.
	if c.Phenotype == models.PhenotypeMaskedHTN {
		urgency := "URGENT"
		riskMultiplier := ""
		if c.DiabetesAmplification || c.CKDAmplification {
			urgency = "IMMEDIATE"
			if c.DiabetesAmplification {
				riskMultiplier = " Diabetes amplification: 3.2x CV risk multiplier vs masked HTN alone (Leitao 2015, Diabetologia)."
			}
		}

		demotedDueToBias := false
		if c.SelectionBiasRisk {
			switch urgency {
			case "IMMEDIATE":
				urgency = "URGENT"
				demotedDueToBias = true
			case "URGENT":
				urgency = "ROUTINE"
				demotedDueToBias = true
			}
		}
		if demotedDueToBias {
			riskMultiplier += " Urgency reduced due to selection bias risk — verify with structured monitoring before acting."
		}

		title := fmt.Sprintf("Masked Hypertension — Clinic BP Normal, Home Mean %.0f mmHg", c.HomeSBPMean)
		fallback := fmt.Sprintf(
			"Clinic BP %.0f/%.0f mmHg (normal) but home mean %.0f mmHg (elevated). "+
				"Home BP exceeds clinic by %.0f mmHg. Masked hypertension carries higher CV risk than "+
				"sustained hypertension because treatment is deferred.%s",
			c.ClinicSBPMean, c.ClinicDBPMean, c.HomeSBPMean, math.Abs(c.ClinicHomeGapSBP), riskMultiplier)
		actions := []string{
			"Do not rely on clinic BP alone — initiate or intensify antihypertensive therapy",
			"Target home BP <130/80 mmHg (AHA/ACC 2023)",
			"Assess for end-organ damage: renal function, retinopathy, LVH",
			"Review home monitoring technique and device accuracy",
		}
		cardData := baseData
		cardData.RiskMultiplier = riskMultiplier
		card := MaskedHTNCard{
			CardType:       "MASKED_HYPERTENSION",
			Urgency:        urgency,
			Title:          title,
			Rationale:      fallback,
			Actions:        actions,
			ConfidenceTier: confidenceTier,
		}
		cards = append(cards, b.applyTemplate(card, cardData))
	}

	// 3. Masked uncontrolled hypertension (treated but not controlled at home).
	if c.Phenotype == models.PhenotypeMaskedUncontrolled {
		fallback := fmt.Sprintf(
			"Patient is controlled in clinic but not at home. Clinic BP %.0f/%.0f mmHg; home mean %.0f mmHg.",
			c.ClinicSBPMean, c.ClinicDBPMean, c.HomeSBPMean)
		if c.CKDAmplification {
			fallback += " CKD co-presence accelerates renal progression with uncontrolled BP."
		}
		actions := []string{
			"Review current antihypertensive regimen — dose or agent adjustment likely required",
			"Check medication adherence: home readings pattern vs dosing schedule",
			"Consider ambulatory BP monitoring (ABPM) to quantify 24-hour burden",
		}
		cardData := baseData
		if c.CKDAmplification {
			cardData.CKDNote = " CKD co-presence accelerates renal progression with uncontrolled BP."
		}
		card := MaskedHTNCard{
			CardType:       "MASKED_UNCONTROLLED",
			Urgency:        "URGENT",
			Title:          "Masked Uncontrolled HTN — Therapy Appears Inadequate at Home",
			Rationale:      fallback,
			Actions:        actions,
			ConfidenceTier: confidenceTier,
		}
		cards = append(cards, b.applyTemplate(card, cardData))
	}

	// 4. White-coat hypertension — avoid overtreatment.
	if c.Phenotype == models.PhenotypeWhiteCoatHTN {
		fallback := fmt.Sprintf(
			"Clinic BP %.0f/%.0f mmHg elevated; home mean %.0f mmHg normal. "+
				"White-coat effect: %.0f mmHg. Risk of overtreatment: initiating or escalating "+
				"antihypertensives based on clinic-only readings may cause iatrogenic hypotension. "+
				"White-coat HTN does not carry the same CV risk as true sustained HTN.",
			c.ClinicSBPMean, c.ClinicDBPMean, c.HomeSBPMean, c.WhiteCoatEffect)
		actions := []string{
			"Do NOT intensify antihypertensive therapy based on this clinic reading alone",
			"Continue home monitoring — reassess if home readings consistently rise above 135/85 mmHg",
			"Lifestyle counselling (sodium, exercise, stress) remains appropriate",
			"Consider ABPM for formal confirmation if clinical decision is complex",
		}
		card := MaskedHTNCard{
			CardType:       "WHITE_COAT_HYPERTENSION",
			Urgency:        "ROUTINE",
			Title:          fmt.Sprintf("White-Coat HTN — Clinic BP Elevated, Home Mean %.0f mmHg", c.HomeSBPMean),
			Rationale:      fallback,
			Actions:        actions,
			ConfidenceTier: confidenceTier,
		}
		cards = append(cards, b.applyTemplate(card, baseData))
	}

	// 4b. White-coat uncontrolled — treated patient with elevated clinic BP but controlled home BP.
	if c.Phenotype == models.PhenotypeWhiteCoatUncontrolled {
		fallback := fmt.Sprintf(
			"Patient on antihypertensive therapy. Clinic BP %.0f/%.0f mmHg elevated; home mean %.0f mmHg normal. "+
				"White-coat effect: %.0f mmHg. Home readings confirm treatment is adequate — the clinic elevation reflects "+
				"the clinic environment, not true treatment failure.",
			c.ClinicSBPMean, c.ClinicDBPMean, c.HomeSBPMean, c.WhiteCoatEffect)
		actions := []string{
			"Do NOT escalate antihypertensive therapy based on clinic reading alone",
			"Consider REDUCING dose if patient reports symptomatic hypotension (dizziness, fatigue)",
			"Continue structured home monitoring — reassess if home readings rise above 135/85 mmHg",
			"Document white-coat effect in chart to prevent future overtreatment",
		}
		card := MaskedHTNCard{
			CardType:       "WHITE_COAT_UNCONTROLLED",
			Urgency:        "ROUTINE",
			Title:          "White-Coat Effect — Clinic BP Elevated but Home BP Controlled on Therapy",
			Rationale:      fallback,
			Actions:        actions,
			ConfidenceTier: confidenceTier,
		}
		cards = append(cards, b.applyTemplate(card, baseData))
	}

	// 5. Selection bias warning for low-confidence / avoidant patients.
	if c.SelectionBiasRisk {
		engagementDetail := ""
		if c.EngagementPhenotype != "" {
			engagementDetail = fmt.Sprintf(" Patient engagement phenotype: %s.",
				strings.ToLower(strings.ReplaceAll(c.EngagementPhenotype, "_", " ")))
		}
		readingDetail := ""
		if c.HomeReadingCount > 0 {
			readingDetail = fmt.Sprintf(" Only %d home readings available.", c.HomeReadingCount)
		}
		fallback := fmt.Sprintf(
			"Home BP classification has LOW confidence due to potential selection bias.%s%s "+
				"Patients who measure only when symptomatic produce systematically higher readings; "+
				"measurement avoidant patients produce systematically lower readings — both distort "+
				"the clinic-home discordance analysis.",
			engagementDetail, readingDetail)
		actions := []string{
			"Discuss structured home monitoring protocol: morning and evening readings x 7 days",
			"Validate home BP device against clinic sphygmomanometer",
			"Do not change therapy based on this classification alone — repeat structured HBPM first",
		}
		cardData := baseData
		cardData.EngagementDetail = engagementDetail
		cardData.ReadingDetail = readingDetail
		card := MaskedHTNCard{
			CardType:       "SELECTION_BIAS_WARNING",
			Urgency:        "ROUTINE",
			Title:          "Home BP Reading Quality — Selection Bias Risk",
			Rationale:      fallback,
			Actions:        actions,
			ConfidenceTier: confidenceTier,
		}
		cards = append(cards, b.applyTemplate(card, cardData))
	}

	// 6. Medication timing hypothesis.
	if c.MedicationTimingHypothesis != "" && c.OnAntihypertensives {
		fallback := fmt.Sprintf(
			"BP pattern suggests suboptimal medication timing. %s Chronotherapy — matching dosing time to BP circadian pattern — can reduce 24-hour BP burden without dose escalation.",
			c.MedicationTimingHypothesis)
		actions := []string{
			"Consider switching once-daily agent to evening dosing",
			"Reassess home BP after 4 weeks of timing change before escalating dose",
			"ABPM post-change to confirm circadian pattern normalisation",
		}
		cardData := baseData
		cardData.MedicationHypothesis = c.MedicationTimingHypothesis
		card := MaskedHTNCard{
			CardType:       "MEDICATION_TIMING",
			Urgency:        "ROUTINE",
			Title:          "Medication Timing Optimisation Opportunity",
			Rationale:      fallback,
			Actions:        actions,
			ConfidenceTier: confidenceTier,
		}
		cards = append(cards, b.applyTemplate(card, cardData))
	}

	return cards
}

// EvaluateMaskedHTNCards maintains the historical pure-function signature for
// callers that have not yet been upgraded to use the builder with template
// dependencies. It defaults to the legacy hardcoded text.
func EvaluateMaskedHTNCards(c *models.BPContextClassification) []MaskedHTNCard {
	builder := NewMaskedHTNCardBuilder(nil, nil, nil)
	return builder.Evaluate(c)
}

type maskedCardTemplateConfig struct {
	TemplateID          string
	RationaleFragmentID string
}

var maskedCardTemplates = map[string]maskedCardTemplateConfig{
	"MASKED_HTN_MORNING_SURGE_COMPOUND": {TemplateID: "dc-masked-htn-morning-surge-compound-v1", RationaleFragmentID: "masked_htn_morning_surge_clinician"},
	"SUSTAINED_HTN_MORNING_SURGE":       {TemplateID: "dc-sustained-htn-morning-surge-v1", RationaleFragmentID: "sustained_htn_morning_surge_clinician"},
	"MASKED_HYPERTENSION":               {TemplateID: "dc-masked-hypertension-v1", RationaleFragmentID: "masked_hypertension_clinician"},
	"MASKED_UNCONTROLLED":               {TemplateID: "dc-masked-uncontrolled-v1", RationaleFragmentID: "masked_uncontrolled_clinician"},
	"WHITE_COAT_HYPERTENSION":           {TemplateID: "dc-white-coat-hypertension-v1", RationaleFragmentID: "white_coat_hypertension_clinician"},
	"WHITE_COAT_UNCONTROLLED":           {TemplateID: "dc-white-coat-uncontrolled-v1", RationaleFragmentID: "white_coat_uncontrolled_clinician"},
	"SELECTION_BIAS_WARNING":            {TemplateID: "dc-selection-bias-warning-v1", RationaleFragmentID: "selection_bias_warning_clinician"},
	"MEDICATION_TIMING":                 {TemplateID: "dc-medication-timing-v1", RationaleFragmentID: "medication_timing_clinician"},
}

type maskedTemplateData struct {
	ClinicSBP            string
	ClinicDBP            string
	HomeSBP              string
	HomeDBP              string
	GapSBP               string
	WhiteCoatEffect      string
	RiskMultiplier       string
	CKDNote              string
	EngagementDetail     string
	ReadingDetail        string
	MedicationHypothesis string
}

func newMaskedTemplateData(c *models.BPContextClassification) maskedTemplateData {
	return maskedTemplateData{
		ClinicSBP:            formatMMHg(c.ClinicSBPMean),
		ClinicDBP:            formatMMHg(c.ClinicDBPMean),
		HomeSBP:              formatMMHg(c.HomeSBPMean),
		HomeDBP:              formatMMHg(c.HomeDBPMean),
		GapSBP:               formatMMHg(math.Abs(c.ClinicHomeGapSBP)),
		WhiteCoatEffect:      formatMMHg(c.WhiteCoatEffect),
		MedicationHypothesis: c.MedicationTimingHypothesis,
	}
}

func formatMMHg(value float64) string {
	if value == 0 {
		return "0"
	}
	return fmt.Sprintf("%.0f", value)
}

func (b *MaskedHTNCardBuilder) applyTemplate(card MaskedHTNCard, data maskedTemplateData) MaskedHTNCard {
	cfg, ok := maskedCardTemplates[card.CardType]
	if !ok || b.templateLoader == nil || b.fragmentLoader == nil {
		return card
	}

	tmpl, ok := b.templateLoader.Get(cfg.TemplateID)
	if ok {
		if actions := b.renderActions(tmpl, data); len(actions) > 0 {
			card.Actions = actions
		}
	} else {
		b.log.Warn("masked HTN template missing", zap.String("template_id", cfg.TemplateID))
	}

	if rationale := b.renderFragment(cfg.RationaleFragmentID, data); rationale != "" {
		card.Rationale = rationale
	}

	return card
}

func (b *MaskedHTNCardBuilder) renderFragment(fragmentID string, data maskedTemplateData) string {
	if fragmentID == "" {
		return ""
	}
	frag, ok := b.fragmentLoader.Get(fragmentID)
	if !ok {
		b.log.Warn("masked HTN fragment missing", zap.String("fragment_id", fragmentID))
		return ""
	}
	return b.executeTemplate(frag.TextEn, data)
}

func (b *MaskedHTNCardBuilder) renderActions(tmpl *models.CardTemplate, data maskedTemplateData) []string {
	var actions []string
	for _, rec := range tmpl.Recommendations {
		rendered := b.executeTemplate(rec.ActionTextEn, data)
		rendered = strings.TrimSpace(rendered)
		if rendered != "" {
			actions = append(actions, rendered)
		}
	}
	return actions
}

func (b *MaskedHTNCardBuilder) executeTemplate(raw string, data maskedTemplateData) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	tpl, err := template.New("masked_htn").Parse(raw)
	if err != nil {
		b.log.Warn("failed to parse masked HTN template text", zap.Error(err))
		return raw
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		b.log.Warn("failed to execute masked HTN template text", zap.Error(err))
		return raw
	}
	return strings.TrimSpace(buf.String())
}
