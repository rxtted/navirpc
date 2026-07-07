package art

import "testing"

func TestCAA(t *testing.T) {
	if url, ok := (CAA{}).Resolve(Meta{RGID: "abc"}, nil); !ok || url != "https://coverartarchive.org/release-group/abc/front" {
		t.Fatalf("caa: ok=%v url=%q", ok, url)
	}
	if _, ok := (CAA{}).Resolve(Meta{}, nil); ok {
		t.Fatal("caa should miss with no rgid")
	}
}

func TestTemplate(t *testing.T) {
	tp := Template{Pattern: "https://art/{albumid}.jpg"}
	if url, ok := tp.Resolve(Meta{AlbumID: "x"}, nil); !ok || url != "https://art/x.jpg" {
		t.Fatalf("template: ok=%v url=%q", ok, url)
	}
	if _, ok := tp.Resolve(Meta{}, nil); ok {
		t.Fatal("template should miss when a used placeholder has no value")
	}
}

type fakeGetter struct {
	body   []byte
	status int
}

func (f *fakeGetter) Get(string) ([]byte, int, error) { return f.body, f.status, nil }

func TestITunes(t *testing.T) {
	g := &fakeGetter{status: 200, body: []byte(`{"results":[{"artworkUrl100":"https://cdn/a/100x100bb.jpg","artistName":"Saosin","collectionName":"In Search of Solid Ground"}]}`)}
	url, ok := (iTunes{}).Resolve(Meta{Artist: "Saosin", Album: "In Search of Solid Ground"}, g)
	if !ok || url != "https://cdn/a/600x600bb.jpg" {
		t.Fatalf("itunes should upscale the artwork url: ok=%v url=%q", ok, url)
	}
	multi := &fakeGetter{status: 200, body: []byte(`{"results":[{"artworkUrl100":"https://cdn/wrong/100x100bb.jpg","artistName":"Other","collectionName":"Other"},{"artworkUrl100":"https://cdn/right/100x100bb.jpg","artistName":"Saosin","collectionName":"In Search of Solid Ground"}]}`)}
	if url, ok := (iTunes{}).Resolve(Meta{Artist: "Saosin", Album: "In Search of Solid Ground"}, multi); !ok || url != "https://cdn/right/600x600bb.jpg" {
		t.Fatalf("itunes should skip a mismatch for a matching result: ok=%v url=%q", ok, url)
	}
	if _, ok := (iTunes{}).Resolve(Meta{Artist: "x"}, g); ok {
		t.Fatal("itunes should miss with no album")
	}
	none := &fakeGetter{status: 200, body: []byte(`{"results":[{"artworkUrl100":"https://cdn/w/100x100bb.jpg","artistName":"Nope","collectionName":"Nope"}]}`)}
	if _, ok := (iTunes{}).Resolve(Meta{Artist: "Saosin", Album: "In Search of Solid Ground"}, none); ok {
		t.Fatal("itunes should miss when nothing matches")
	}
}

type countProvider struct {
	url   string
	calls int
}

func (c *countProvider) Resolve(Meta, Getter) (string, bool) {
	c.calls++
	if c.url == "" {
		return "", false
	}
	return c.url, true
}

func TestChain_FirstHitWins(t *testing.T) {
	first := &countProvider{url: "a"}
	second := &countProvider{url: "b"}
	url, ok := Chain([]Provider{first, second}, Meta{AlbumID: "id"}, nil)
	if !ok || url != "a" || second.calls != 0 {
		t.Fatalf("first hit should win and short-circuit: url=%q secondCalls=%d", url, second.calls)
	}
}

func TestKey(t *testing.T) {
	if k := Key(Meta{AlbumID: "al", RGID: "rg", Artist: "x", Album: "y"}); k != "al" {
		t.Fatalf("albumid should win, got %q", k)
	}
	if k := Key(Meta{RGID: "rg", Artist: "x", Album: "y"}); k != "rg" {
		t.Fatalf("rgid should beat the name fallback, got %q", k)
	}
	if k := Key(Meta{Artist: "x", Album: "y"}); k != "x - y" {
		t.Fatalf("name fallback should join artist and album, got %q", k)
	}
	if k := Key(Meta{}); k != "" {
		t.Fatalf("no usable identity should give no key, got %q", k)
	}
}
