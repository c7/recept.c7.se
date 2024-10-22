package main

import (
	"net/http/httptest"
	"testing"
)

func TestGetPort(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		getenv := func(string) string { return "1234" }

		if got, want := getPort(getenv), "1234"; got != want {
			t.Fatalf("getPort(getenv) = %q, want %q", got, want)
		}
	})

	t.Run("Default", func(t *testing.T) {
		getenv := func(string) string { return "" }

		if got, want := getPort(getenv), defaultPort; got != want {
			t.Fatalf("getPort(getenv) = %q, want %q", got, want)
		}
	})
}

func TestParseApp(t *testing.T) {
	app, err := parseApp()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := len(app.recipes), 6; got < want {
		t.Fatalf("len(app.recipes) = %d, want >=%d", got, want)
	}

	t.Run("Det fantastiska brödet", func(t *testing.T) {
		var dfb Recipe

		for _, r := range app.recipes {
			if r.Path == "Det-fantastiska-brödet.md" {
				dfb = r
			}
		}

		titel, ok := dfb.Meta["Titel"].(string)
		if !ok {
			t.Fatalf("expected Titel string")
		}

		if got, want := titel, "Det fantastiska brödet"; got != want {
			t.Fatalf("titel = %q, want %q", got, want)
		}
	})

	t.Run("HTTP", func(t *testing.T) {
		for _, tt := range []struct {
			path string
			code int
		}{
			{"/", 200},
			{"/favicon.ico", 200},
			{"/Det-fantastiska-brödet.md", 200},
			{"/not-found", 404},
		} {
			t.Run(tt.path, func(t *testing.T) {
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", tt.path, nil)

				app.ServeHTTP(w, r)

				if got, want := w.Code, tt.code; got != want {
					t.Fatalf("w.Code = %d, want %d", got, want)
				}
			})
		}
	})
}
