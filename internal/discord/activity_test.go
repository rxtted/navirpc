package discord

import (
	"encoding/json"
	"strings"
	"testing"

	"atrophy/navirpc/internal/presence"
)

func TestToWire_OmitsEmptySections(t *testing.T) {
	b, _ := json.Marshal(toWire(presence.Activity{Type: 2, Name: "Saosin", Platform: "desktop"}, "app1"))
	s := string(b)
	if strings.Contains(s, "timestamps") || strings.Contains(s, "assets") || strings.Contains(s, "buttons") {
		t.Fatalf("empty sections omitted: %s", s)
	}
	if !strings.Contains(s, `"application_id":"app1"`) {
		t.Fatalf("client id rides as application_id: %s", s)
	}
}

func TestToWire_FiltersHalfEmptyButtons(t *testing.T) {
	wa := toWire(presence.Activity{Buttons: []presence.Button{{Label: "View", URL: "https://x"}, {Label: "", URL: "https://y"}, {Label: "z", URL: ""}}}, "")
	if len(wa.Buttons) != 1 || wa.Buttons[0].Label != "View" {
		t.Fatalf("half-empty buttons dropped: %+v", wa.Buttons)
	}
}

func TestToWire_CarriesTheAssets(t *testing.T) {
	wa := toWire(presence.Activity{
		LargeImage: "https://cover", LargeText: "the album",
		SmallImage: "https://icon", SmallText: "navidrome",
	}, "")
	if wa.Assets == nil || wa.Assets.LargeImage != "https://cover" || wa.Assets.SmallText != "navidrome" {
		t.Fatalf("assets populated: %+v", wa.Assets)
	}
}

func TestToWire_TimestampsWhenEitherSet(t *testing.T) {
	wa := toWire(presence.Activity{Start: 100}, "")
	if wa.Timestamps == nil || wa.Timestamps.Start != 100 {
		t.Fatalf("start alone still timestamps: %+v", wa.Timestamps)
	}
}
