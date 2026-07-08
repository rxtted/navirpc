package presence

// the config value's json shape, authored on the connect page or hand-written,
// pointer fields tell an absent key from an explicit blank. absent falls back to
// the default card, a blank template stays blank so a line can be hidden
type Look struct {
	Type              *string  `json:"type"`
	Header            *string  `json:"header"`
	Details           *string  `json:"details"`
	State             *string  `json:"state"`
	DetailsURL        string   `json:"details_url"`
	StateURL          string   `json:"state_url"`
	StatusDisplayType *string  `json:"status_display_type"`
	LargeText         string   `json:"large_text"`
	SmallImage        string   `json:"small_image"`
	SmallText         string   `json:"small_text"`
	Buttons           []Button `json:"buttons"`
}

func (l Look) Prefs() Prefs {
	return Prefs{
		Type:              orDefault(l.Type, "listening"),
		Header:            orDefault(l.Header, "{artist}"),
		Details:           orDefault(l.Details, "{track}"),
		State:             orDefault(l.State, "{album}"),
		DetailsURL:        l.DetailsURL,
		StateURL:          l.StateURL,
		StatusDisplayType: orDefault(l.StatusDisplayType, "name"),
		LargeText:         l.LargeText,
		SmallImage:        l.SmallImage,
		SmallText:         l.SmallText,
		Buttons:           l.Buttons,
	}
}

func orDefault(v *string, def string) string {
	if v == nil {
		return def
	}
	return *v
}
