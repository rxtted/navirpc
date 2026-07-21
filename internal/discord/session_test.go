package discord

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"atrophy/navirpc/internal/presence"
)

type fakeDoer struct {
	resp Response
	err  error
	got  Request
}

func (f *fakeDoer) Do(r Request) (Response, error) {
	f.got = r
	return f.resp, f.err
}

func creds() presence.Creds { return presence.Creds{Access: "at", ClientID: "app1"} }

func TestPublish_CreateOmitsSessionToken(t *testing.T) {
	f := &fakeDoer{resp: Response{StatusCode: 200, Body: []byte(`{"token":"sess1"}`)}}
	got, err := Publisher{D: f}.Publish("noah", presence.Desired{}, "", creds())
	if err != nil || got != "sess1" || strings.Contains(string(f.got.Body), `"token"`) {
		t.Fatalf("create adopts, sends no token: got=%q err=%v body=%s", got, err, f.got.Body)
	}
}

func TestPublish_UpdateCarriesSessionToken(t *testing.T) {
	f := &fakeDoer{resp: Response{StatusCode: 200, Body: []byte(`{"token":"sess2"}`)}}
	got, _ := Publisher{D: f}.Publish("noah", presence.Desired{}, "sess1", creds())
	var sent map[string]any
	if err := json.Unmarshal(f.got.Body, &sent); err != nil {
		t.Fatalf("request body unparseable: %v", err)
	}
	if sent["token"] != "sess1" || got != "sess2" {
		t.Fatalf("update carries held, adopts returned: sent=%v got=%q", sent["token"], got)
	}
}

func TestPublish_UpdateKeepsTokenWhenBodyOmitsIt(t *testing.T) {
	f := &fakeDoer{resp: Response{StatusCode: 200, Body: []byte(`{}`)}}
	got, err := Publisher{D: f}.Publish("noah", presence.Desired{}, "sess1", creds())
	if err != nil || got != "sess1" {
		t.Fatalf("tokenless body keeps held: got=%q err=%v", got, err)
	}
}

func TestPublish_RateLimitCarriesRetryAfter(t *testing.T) {
	f := &fakeDoer{resp: Response{StatusCode: 429, Headers: map[string]string{"retry-after": "3.5"}}}
	_, err := Publisher{D: f}.Publish("noah", presence.Desired{}, "", creds())
	var rl interface{ RetryAfterMs() int64 }
	if !errors.As(err, &rl) || rl.RetryAfterMs() != 3500 {
		t.Fatalf("429 surfaces the window: %v", err)
	}
}

func TestPublish_NonSuccessErrors(t *testing.T) {
	f := &fakeDoer{resp: Response{StatusCode: 500}}
	_, err := Publisher{D: f}.Publish("noah", presence.Desired{}, "", creds())
	if err == nil {
		t.Fatal("500 errors")
	}
}

func TestPublish_NoAccessTokenErrors(t *testing.T) {
	f := &fakeDoer{}
	_, err := Publisher{D: f}.Publish("noah", presence.Desired{}, "", presence.Creds{ClientID: "app1"})
	if err == nil || f.got.URL != "" {
		t.Fatalf("no bearer, no request: err=%v url=%q", err, f.got.URL)
	}
}

func TestClear_NoSessionIsNoop(t *testing.T) {
	f := &fakeDoer{}
	err := Publisher{D: f}.Clear("noah", "", creds())
	if err != nil || f.got.URL != "" {
		t.Fatalf("no session, no call: err=%v url=%q", err, f.got.URL)
	}
}
