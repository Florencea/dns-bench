package i18n

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestCatalogCompleteness(t *testing.T) {
	t.Parallel()

	for _, locale := range []string{LocaleZhTW, LocaleEnUS} {
		msg := Catalog(locale)
		if msg.LangCode != locale {
			t.Fatalf("Catalog(%q).LangCode = %q", locale, msg.LangCode)
		}
		assertNoEmptyStrings(t, locale, reflect.ValueOf(msg), "")
	}
}

func assertNoEmptyStrings(t *testing.T, locale string, v reflect.Value, prefix string) {
	t.Helper()
	switch v.Kind() {
	case reflect.Struct:
		typ := v.Type()
		for i := range v.NumField() {
			name := typ.Field(i).Name
			if prefix != "" {
				name = prefix + "." + name
			}
			assertNoEmptyStrings(t, locale, v.Field(i), name)
		}
	case reflect.String:
		if v.String() == "" {
			t.Errorf("locale %s has empty translation at %s", locale, prefix)
		}
	}
}

func TestResolveLocalePriority(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if got := ResolveLocale(req, LocaleZhTW); got != LocaleEnUS {
		t.Fatalf("expected Accept-Language en-US, got %q", got)
	}

	req.AddCookie(&http.Cookie{Name: LocaleCookieName, Value: LocaleZhTW})
	if got := ResolveLocale(req, LocaleEnUS); got != LocaleZhTW {
		t.Fatalf("expected cookie zh-TW to override Accept-Language, got %q", got)
	}

	fallbackReq := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := ResolveLocale(fallbackReq, LocaleEnUS); got != LocaleEnUS {
		t.Fatalf("expected default en-US, got %q", got)
	}
}
