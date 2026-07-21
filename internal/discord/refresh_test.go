package discord

import (
	"errors"
	"testing"

	"atrophy/navirpc/internal/auth"
)

func TestRefresh_ParsesGrant(t *testing.T) {
	f := &fakeDoer{resp: Response{StatusCode: 200, Body: []byte(`{"access_token":"at2","refresh_token":"rt2","expires_in":604800}`)}}
	access, refresh, exp, err := Refresher{D: f}.Refresh("app1", "rt1")
	if err != nil || access != "at2" || refresh != "rt2" || exp != 604800 {
		t.Fatalf("grant parsed whole: access=%q refresh=%q exp=%d err=%v", access, refresh, exp, err)
	}
}

func TestRefresh_InvalidGrantOn400(t *testing.T) {
	f := &fakeDoer{resp: Response{StatusCode: 400, Body: []byte(`{"error":"invalid_grant"}`)}}
	_, _, _, err := Refresher{D: f}.Refresh("app1", "rt1")
	if !errors.Is(err, auth.ErrInvalidGrant) {
		t.Fatalf("400 is the dead-token sentinel: %v", err)
	}
}

func TestRefresh_ServerErrorIsTransient(t *testing.T) {
	f := &fakeDoer{resp: Response{StatusCode: 502}}
	_, _, _, err := Refresher{D: f}.Refresh("app1", "rt1")
	if err == nil || errors.Is(err, auth.ErrInvalidGrant) {
		t.Fatalf("5xx is not a dead token: %v", err)
	}
}

func TestRefresh_MalformedBodyErrors(t *testing.T) {
	f := &fakeDoer{resp: Response{StatusCode: 200, Body: []byte(`{"access_token":`)}}
	access, _, _, err := Refresher{D: f}.Refresh("app1", "rt1")
	if err == nil || access != "" {
		t.Fatalf("unparseable 200 is an error: access=%q err=%v", access, err)
	}
}

func TestRefresh_TransportErrorPropagates(t *testing.T) {
	f := &fakeDoer{err: errors.New("dial timeout")}
	if _, _, _, err := (Refresher{D: f}.Refresh("app1", "rt1")); err == nil {
		t.Fatal("transport failure surfaces")
	}
}
