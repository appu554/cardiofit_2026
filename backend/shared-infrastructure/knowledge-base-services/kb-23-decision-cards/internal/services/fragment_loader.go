package services

import (
	"sync"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// FragmentLoader manages SummaryFragments extracted from CardTemplate YAML
// definitions. It provides lookup by fragment_id and template_id, and
// implements the V-03 locale fallback chain for patient-facing text:
// text_local -> text_hi -> text_en.
type FragmentLoader struct {
	log        *zap.Logger
	mu         sync.RWMutex
	fragments  map[string]*models.SummaryFragment   // keyed by fragment_id
	byTemplate map[string][]*models.SummaryFragment  // keyed by template_id
}

// NewFragmentLoader creates a FragmentLoader instance.
func NewFragmentLoader(log *zap.Logger) *FragmentLoader {
	return &FragmentLoader{
		log:        log,
		fragments:  make(map[string]*models.SummaryFragment),
		byTemplate: make(map[string][]*models.SummaryFragment),
	}
}

// LoadFromTemplates extracts SummaryFragment records from loaded CardTemplates,
// replacing any previously loaded fragments atomically.
func (f *FragmentLoader) LoadFromTemplates(templates []*models.CardTemplate) {
	f.mu.Lock()
	defer f.mu.Unlock()

	newFragments := make(map[string]*models.SummaryFragment)
	newByTemplate := make(map[string][]*models.SummaryFragment)

	for _, tmpl := range templates {
		for _, tf := range tmpl.Fragments {
			frag := &models.SummaryFragment{
				FragmentID:            tf.FragmentID,
				TemplateID:            tmpl.TemplateID,
				FragmentType:          tf.FragmentType,
				TextEn:               tf.TextEn,
				TextHi:               tf.TextHi,
				Version:              tf.Version,
				ReadingLevelValidated: tf.ReadingLevelValidated,
			}

			if tf.TextLocal != "" {
				local := tf.TextLocal
				frag.TextLocal = &local
			}
			if tf.LocaleCode != "" {
				locale := tf.LocaleCode
				frag.LocaleCode = &locale
			}
			if tf.PatientAdvocateReviewedBy != "" {
				reviewer := tf.PatientAdvocateReviewedBy
				frag.PatientAdvocateReviewedBy = &reviewer
			}
			if tf.GuidelineRef != "" {
				ref := tf.GuidelineRef
				frag.GuidelineRef = &ref
			}

			newFragments[tf.FragmentID] = frag
			newByTemplate[tmpl.TemplateID] = append(newByTemplate[tmpl.TemplateID], frag)
		}
	}

	f.fragments = newFragments
	f.byTemplate = newByTemplate
	f.log.Info("fragments loaded", zap.Int("count", len(f.fragments)))
}

// GetPatientText returns the best available patient-facing text for a fragment
// using the V-03 locale fallback chain: text_local -> text_hi -> text_en.
// Returns an empty string if the fragment is not found.
func (f *FragmentLoader) GetPatientText(fragmentID string) string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	frag, ok := f.fragments[fragmentID]
	if !ok {
		return ""
	}

	if frag.TextLocal != nil && *frag.TextLocal != "" {
		return *frag.TextLocal
	}
	if frag.TextHi != "" {
		return frag.TextHi
	}
	return frag.TextEn
}

// GetByTemplate returns all fragments belonging to a template.
func (f *FragmentLoader) GetByTemplate(templateID string) []*models.SummaryFragment {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.byTemplate[templateID]
}

// Get returns a fragment by its fragment_id.
func (f *FragmentLoader) Get(fragmentID string) (*models.SummaryFragment, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	frag, ok := f.fragments[fragmentID]
	return frag, ok
}

// Count returns the number of loaded fragments.
func (f *FragmentLoader) Count() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.fragments)
}
