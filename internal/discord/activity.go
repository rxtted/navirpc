package discord

import "atrophy/navirpc/internal/presence"

type wireActivity struct {
	Type              int             `json:"type"`
	Name              string          `json:"name"`
	Platform          string          `json:"platform"`
	ApplicationID     string          `json:"application_id,omitempty"`
	Details           string          `json:"details,omitempty"`
	DetailsURL        string          `json:"details_url,omitempty"`
	State             string          `json:"state,omitempty"`
	StateURL          string          `json:"state_url,omitempty"`
	StatusDisplayType int             `json:"status_display_type,omitempty"`
	Timestamps        *wireTimestamps `json:"timestamps,omitempty"`
	Assets            *wireAssets     `json:"assets,omitempty"`
	Buttons           []wireButton    `json:"buttons,omitempty"`
}

type wireTimestamps struct {
	Start int64 `json:"start,omitempty"`
	End   int64 `json:"end,omitempty"`
}

type wireAssets struct {
	LargeImage string `json:"large_image,omitempty"`
	LargeText  string `json:"large_text,omitempty"`
	SmallImage string `json:"small_image,omitempty"`
	SmallText  string `json:"small_text,omitempty"`
}

// buttons go up as objects, discord normalizes them to a label array plus metadata
type wireButton struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

func toWire(a presence.Activity, clientID string) wireActivity {
	wa := wireActivity{
		Type: a.Type, Name: a.Name, Platform: a.Platform, ApplicationID: clientID,
		Details: a.Details, DetailsURL: a.DetailsURL,
		State: a.State, StateURL: a.StateURL,
		StatusDisplayType: a.StatusDisplayType,
	}
	if a.Start != 0 || a.End != 0 {
		wa.Timestamps = &wireTimestamps{Start: a.Start, End: a.End}
	}
	if a.LargeImage != "" || a.LargeText != "" || a.SmallImage != "" || a.SmallText != "" {
		wa.Assets = &wireAssets{LargeImage: a.LargeImage, LargeText: a.LargeText, SmallImage: a.SmallImage, SmallText: a.SmallText}
	}
	for _, b := range a.Buttons {
		if b.Label != "" && b.URL != "" {
			wa.Buttons = append(wa.Buttons, wireButton{Label: b.Label, URL: b.URL})
		}
	}
	return wa
}
