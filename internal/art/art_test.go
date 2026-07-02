package art

import "testing"

func TestCAA(t *testing.T) {
	if url, ok := (CAA{}).Resolve(Meta{RGID: "abc"}); !ok || url != "https://coverartarchive.org/release-group/abc/front" {
		t.Fatalf("caa: ok=%v url=%q", ok, url)
	}
	if _, ok := (CAA{}).Resolve(Meta{}); ok {
		t.Fatal("caa should miss with no rgid")
	}
}

func TestTemplate(t *testing.T) {
	tp := Template{Pattern: "https://art/{albumid}.jpg"}
	if url, ok := tp.Resolve(Meta{AlbumID: "x"}); !ok || url != "https://art/x.jpg" {
		t.Fatalf("template: ok=%v url=%q", ok, url)
	}
	if _, ok := tp.Resolve(Meta{}); ok {
		t.Fatal("template should miss when a used placeholder has no value")
	}
}

type countProvider struct {
	url   string
	calls int
}

func (c *countProvider) Resolve(Meta) (string, bool) {
	c.calls++
	if c.url == "" {
		return "", false
	}
	return c.url, true
}

type mapCache map[string]string

func (m mapCache) Get(k string) (string, bool) { v, ok := m[k]; return v, ok }
func (m mapCache) Set(k, v string)             { m[k] = v }

func TestChain_FirstHitWins(t *testing.T) {
	first := &countProvider{url: "a"}
	second := &countProvider{url: "b"}
	url, ok := Chain([]Provider{first, second}, mapCache{}, Meta{AlbumID: "id"})
	if !ok || url != "a" || second.calls != 0 {
		t.Fatalf("first hit should win and short-circuit: url=%q secondCalls=%d", url, second.calls)
	}
}

func TestChain_CachesHit(t *testing.T) {
	p := &countProvider{url: "a"}
	cache := mapCache{}
	m := Meta{AlbumID: "id"}
	Chain([]Provider{p}, cache, m)
	Chain([]Provider{p}, cache, m)
	if p.calls != 1 {
		t.Fatalf("second call should hit cache, provider calls=%d", p.calls)
	}
}

func TestChain_CachesMiss(t *testing.T) {
	p := &countProvider{url: ""} // always misses
	cache := mapCache{}
	m := Meta{AlbumID: "id"}
	Chain([]Provider{p}, cache, m)
	if _, ok := Chain([]Provider{p}, cache, m); ok {
		t.Fatal("cached miss should stay a miss")
	}
	if p.calls != 1 {
		t.Fatalf("a cached miss should not re-call the provider, calls=%d", p.calls)
	}
}
