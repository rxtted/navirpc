package presence

import (
	"encoding/json"
	"testing"
)

func TestLook_BlankTemplateStaysBlank(t *testing.T) {
	var l Look
	if err := json.Unmarshal([]byte(`{"details": ""}`), &l); err != nil {
		t.Fatal(err)
	}
	p := l.Prefs()
	if p.Details != "" || p.Header != "{artist}" {
		t.Fatalf("an explicit blank hides the line, absent keys still default: %+v", p)
	}
}
