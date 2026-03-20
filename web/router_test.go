package web

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestRootRedirectsToStudyclaw(t *testing.T) {
	router := RouterInit()

	for _, path := range []string{"/", "/studyclaw", "/studyclaw/index.html"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusTemporaryRedirect {
			t.Fatalf("path %s returned %d, want %d", path, rec.Code, http.StatusTemporaryRedirect)
		}
		if location := rec.Header().Get("Location"); location != "/studyclaw/" {
			t.Fatalf("path %s redirected to %s, want /studyclaw/", path, location)
		}
	}
}

func TestStudyclawEntryAndAssetsAreServed(t *testing.T) {
	router := RouterInit()

	indexReq := httptest.NewRequest(http.MethodGet, "/studyclaw/", nil)
	indexRec := httptest.NewRecorder()
	router.ServeHTTP(indexRec, indexReq)
	if indexRec.Code != http.StatusOK {
		t.Fatalf("entry returned %d, want %d", indexRec.Code, http.StatusOK)
	}

	jsRe := regexp.MustCompile(`/studyclaw/static/js/[^"]+\.js`)
	jsPath := jsRe.FindString(indexRec.Body.String())
	if jsPath == "" {
		t.Fatal("failed to find studyclaw JS asset in entry html")
	}

	for _, path := range []string{"/studyclaw/manifest.json", jsPath} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("asset %s returned %d, want %d", path, rec.Code, http.StatusOK)
		}
	}
}
