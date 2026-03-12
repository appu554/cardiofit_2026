package models

// SummaryFragment stores pre-authored, clinically reviewed text blocks
// used to compose patient-facing and clinician-facing card summaries.
type SummaryFragment struct {
	FragmentID                string       `gorm:"primaryKey" json:"fragment_id"`
	TemplateID                string       `gorm:"index;not null" json:"template_id"`
	FragmentType              FragmentType `gorm:"type:varchar(30);not null" json:"fragment_type"`
	TextEn                    string       `gorm:"type:text;not null" json:"text_en"`
	TextHi                    string       `gorm:"type:text;not null" json:"text_hi"`
	TextLocal                 *string      `gorm:"type:text" json:"text_local,omitempty"`
	LocaleCode                *string      `gorm:"type:varchar(10)" json:"locale_code,omitempty"`
	PatientAdvocateReviewedBy *string      `json:"patient_advocate_reviewed_by,omitempty"`
	ReadingLevelValidated     bool         `gorm:"default:false" json:"reading_level_validated"`
	GuidelineRef              *string      `json:"guideline_ref,omitempty"`
	Version                   string       `json:"version"`
}

// TableName sets the PostgreSQL table name.
func (SummaryFragment) TableName() string { return "summary_fragments" }
