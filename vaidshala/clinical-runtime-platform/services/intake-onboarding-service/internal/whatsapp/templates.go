package whatsapp

const (
	LangEnglish = "en"
	LangHindi   = "hi"
	LangMarathi = "mr"
	LangTamil   = "ta"
	LangTelugu  = "te"
	LangKannada = "kn"
	LangBengali = "bn"
)

type TemplateID string

const (
	TplWelcome          TemplateID = "cardiofit_welcome_v1"
	TplOTPVerify        TemplateID = "cardiofit_otp_v1"
	TplIntakeStart      TemplateID = "cardiofit_intake_start_v1"
	TplSlotQuestion     TemplateID = "cardiofit_slot_question_v1"
	TplReminder24h      TemplateID = "cardiofit_reminder_24h_v1"
	TplReminder48h      TemplateID = "cardiofit_reminder_48h_v1"
	TplReminder72h      TemplateID = "cardiofit_reminder_72h_v1"
	TplIntakeComplete   TemplateID = "cardiofit_intake_complete_v1"
	TplCheckinStart     TemplateID = "cardiofit_checkin_start_v1"
	TplHardStopEscalate TemplateID = "cardiofit_hard_stop_v1"
)

type InteractiveButton struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type SlotQuestionTemplate struct {
	QuestionText map[string]string
	Buttons      []InteractiveButton
}

func StandardYesNo(lang string) []InteractiveButton {
	switch lang {
	case LangHindi:
		return []InteractiveButton{
			{ID: "yes", Title: "हाँ"},
			{ID: "no", Title: "नहीं"},
		}
	case LangMarathi:
		return []InteractiveButton{
			{ID: "yes", Title: "होय"},
			{ID: "no", Title: "नाही"},
		}
	default:
		return []InteractiveButton{
			{ID: "yes", Title: "Yes"},
			{ID: "no", Title: "No"},
		}
	}
}

func StandardYesNoUnsure(lang string) []InteractiveButton {
	btns := StandardYesNo(lang)
	switch lang {
	case LangHindi:
		btns = append(btns, InteractiveButton{ID: "unsure", Title: "पता नहीं"})
	case LangMarathi:
		btns = append(btns, InteractiveButton{ID: "unsure", Title: "माहित नाही"})
	default:
		btns = append(btns, InteractiveButton{ID: "unsure", Title: "Not sure"})
	}
	return btns
}
