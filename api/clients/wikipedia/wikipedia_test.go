package wikipedia

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeServer routes both opensearch (action=...) and REST summary (/page/summary/...)
// on a single httptest server.
func fakeServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for path, h := range handlers {
		mux.HandleFunc(path, h)
	}
	return httptest.NewServer(mux)
}

func TestSearch_HappyPath(t *testing.T) {
	big := strings.Repeat("e", 10000)
	srv := fakeServer(t, map[string]http.HandlerFunc{
		"/w/api.php": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("action") != "opensearch" {
				http.Error(w, "unexpected action", 400)
				return
			}
			if r.URL.Query().Get("search") != "voyager 1" {
				http.Error(w, "unexpected search term", 400)
				return
			}
			w.Write([]byte(`["voyager 1",["Voyager 1"],["probe"],["https://en.wikipedia.org/wiki/Voyager_1"]]`))
		},
		"/api/rest_v1/page/summary/Voyager%201": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"type":"standard","title":"Voyager 1","description":"Space probe","extract":"` + big + `","content_urls":{"desktop":{"page":"https://en.wikipedia.org/wiki/Voyager_1"}}}`))
		},
	})
	defer srv.Close()

	c := New(WithBaseURLs(srv.URL+"/w/api.php", srv.URL+"/api/rest_v1"), WithCharBudget(500))
	r, err := c.Search(context.Background(), "voyager 1")
	if err != nil {
		t.Fatal(err)
	}
	if r.Title != "Voyager 1" {
		t.Errorf("title: %s", r.Title)
	}
	if n := len([]rune(r.Extract)); n > 500 {
		t.Errorf("extract not truncated to 500 runes: got %d", n)
	}
	if !strings.HasSuffix(r.Extract, "…[truncated]") {
		t.Errorf("truncation suffix missing")
	}
	if r.URL != "https://en.wikipedia.org/wiki/Voyager_1" {
		t.Errorf("url: %s", r.URL)
	}
}

func TestSearch_OpensearchNoResults(t *testing.T) {
	srv := fakeServer(t, map[string]http.HandlerFunc{
		"/w/api.php": func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`["xyz",[],[],[]]`))
		},
	})
	defer srv.Close()
	c := New(WithBaseURLs(srv.URL+"/w/api.php", srv.URL+"/api/rest_v1"))
	_, err := c.Search(context.Background(), "xyz")
	if err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestSearch_Summary404(t *testing.T) {
	srv := fakeServer(t, map[string]http.HandlerFunc{
		"/w/api.php": func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`["x",["X"],[""],["https://en.wikipedia.org/wiki/X"]]`))
		},
		"/api/rest_v1/page/summary/X": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		},
	})
	defer srv.Close()
	c := New(WithBaseURLs(srv.URL+"/w/api.php", srv.URL+"/api/rest_v1"))
	_, err := c.Search(context.Background(), "x")
	if err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestSearch_DisambiguationPopulatesCandidates(t *testing.T) {
	srv := fakeServer(t, map[string]http.HandlerFunc{
		"/w/api.php": func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			switch q.Get("action") {
			case "opensearch":
				w.Write([]byte(`["mercury",["Mercury"],[""],[""]]`))
			case "query":
				w.Write([]byte(`{"query":{"pages":{"1":{"links":[{"title":"Mercury (planet)"},{"title":"Mercury (element)"},{"title":"Freddie Mercury"}]}}}}`))
			default:
				http.Error(w, "bad action", 400)
			}
		},
		"/api/rest_v1/page/summary/Mercury": func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"type":"disambiguation","title":"Mercury","extract":"","content_urls":{"desktop":{"page":"https://en.wikipedia.org/wiki/Mercury"}}}`))
		},
	})
	defer srv.Close()
	c := New(WithBaseURLs(srv.URL+"/w/api.php", srv.URL+"/api/rest_v1"))
	r, err := c.Search(context.Background(), "mercury")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Candidates) == 0 {
		t.Fatal("expected disambiguation candidates")
	}
	found := false
	for _, c := range r.Candidates {
		if c == "Mercury (planet)" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Mercury (planet)' in candidates, got %v", r.Candidates)
	}
	if !strings.Contains(r.Extract, "disambiguation") {
		t.Errorf("extract should describe disambiguation, got %q", r.Extract)
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	c := New()
	if _, err := c.Search(context.Background(), "   "); err == nil {
		t.Fatal("expected error")
	}
}

func TestSearch_SummaryNon2xxError(t *testing.T) {
	srv := fakeServer(t, map[string]http.HandlerFunc{
		"/w/api.php": func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`["x",["X"],[""],[""]]`))
		},
		"/api/rest_v1/page/summary/X": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			w.Write([]byte("internal error"))
		},
	})
	defer srv.Close()
	c := New(WithBaseURLs(srv.URL+"/w/api.php", srv.URL+"/api/rest_v1"))
	_, err := c.Search(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*APIError); !ok {
		t.Errorf("want APIError, got %T", err)
	}
}

func TestSearch_UserAgentSent(t *testing.T) {
	var sawUA string
	srv := fakeServer(t, map[string]http.HandlerFunc{
		"/w/api.php": func(w http.ResponseWriter, r *http.Request) {
			sawUA = r.Header.Get("User-Agent")
			w.Write([]byte(`["x",[],[],[]]`))
		},
	})
	defer srv.Close()
	c := New(WithBaseURLs(srv.URL+"/w/api.php", srv.URL+"/api/rest_v1"), WithUserAgent("custom/1.0"))
	c.Search(context.Background(), "x")
	if sawUA != "custom/1.0" {
		t.Errorf("UA: %q", sawUA)
	}
}

// ─── Integration (real Wikipedia) ────────────────────────────────────────────

func TestSearch_Integration_Voyager(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	c := New()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	r, err := c.Search(ctx, "Voyager 1")
	if err != nil {
		t.Fatalf("live search: %v", err)
	}
	if !strings.Contains(strings.ToLower(r.Title), "voyager") {
		t.Errorf("unexpected title: %s", r.Title)
	}
	if !strings.Contains(strings.ToLower(r.Extract), "probe") && !strings.Contains(strings.ToLower(r.Extract), "spacecraft") {
		t.Errorf("extract should mention probe/spacecraft: %q", r.Extract)
	}
	if n := len([]rune(r.Extract)); n > 5000 {
		t.Errorf("extract %d runes exceeds 5000", n)
	}
}

func TestSearch_Integration_Disambig(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	c := New()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	r, err := c.Search(ctx, "Mercury")
	if err != nil {
		t.Fatalf("live search: %v", err)
	}
	// "Mercury" on en.wiki is a disambig page today.
	if len(r.Candidates) == 0 {
		t.Logf("note: Mercury may have resolved to a concrete page: %+v", r)
	}
}
