package web_test

import (
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"testing"

	webassets "github.com/Florencea/dns-bench/web"
)

func TestBootstrapOnlyZeroJSAndAccessibility(t *testing.T) {
	t.Parallel()

	cssBytes, err := fs.ReadFile(webassets.Assets, "static/bootstrap.min.css")
	if err != nil {
		t.Fatalf("read embedded bootstrap.min.css: %v", err)
	}
	allowedClasses := parseCSSClasses(string(cssBytes))
	if len(allowedClasses) < 500 {
		t.Fatalf("expected hundreds of Bootstrap classes parsed, got %d", len(allowedClasses))
	}

	for _, required := range []string{"container-fluid", "btn-primary", "form-control", "is-invalid", "nav-pills", "font-monospace"} {
		if !allowedClasses[required] {
			t.Fatalf("CSS parser missed expected Bootstrap class %q", required)
		}
	}

	// 1. Static template inspection across all files in web/templates/*.html.
	entries, err := fs.ReadDir(webassets.Assets, "templates")
	if err != nil {
		t.Fatalf("read templates dir: %v", err)
	}

	forbiddenRawPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script\b`),
		regexp.MustCompile(`(?i)<style\b`),
		regexp.MustCompile(`(?i)<iframe\b`),
		regexp.MustCompile(`(?i)\sstyle\s*=`),
		regexp.MustCompile(`(?i)\son[a-z]+\s*=`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)\bdata-bs-(toggle|dismiss|target|ride|slide)\b`),
	}

	for _, entry := range entries {
		raw, readErr := fs.ReadFile(webassets.Assets, "templates/"+entry.Name())
		if readErr != nil {
			t.Fatalf("read template %s: %v", entry.Name(), readErr)
		}
		text := string(raw)
		for _, re := range forbiddenRawPatterns {
			if loc := re.FindString(text); loc != "" {
				t.Fatalf("template %s contains forbidden token %q", entry.Name(), loc)
			}
		}
	}

	// 2. Render every page state through the live HTTP handler and verify classes + a11y.
	ts, cleanup := newMockWebServer(t, nil)
	defer cleanup()

	cases := []struct {
		path       string
		wantStatus int
		expectErr  bool
	}{
		{path: "/", wantStatus: http.StatusOK},
		{path: "/?mode=doh", wantStatus: http.StatusOK},
		{path: "/?mode=dns&host=example.com", wantStatus: http.StatusOK},
		{path: "/?mode=doh&host=example.com", wantStatus: http.StatusOK},
		{path: "/?mode=dns&host=bad%20domain!", wantStatus: http.StatusBadRequest, expectErr: true},
	}

	for _, tc := range cases {
		resp, err := ts.Client().Get(ts.URL + tc.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tc.path, err)
		}
		body := readResponseBody(t, resp)
		if resp.StatusCode != tc.wantStatus {
			t.Fatalf("GET %s status = %d, want %d", tc.path, resp.StatusCode, tc.wantStatus)
		}
		assertRenderedHTMLCompliant(t, tc.path, body, allowedClasses, tc.expectErr)
	}
}

var (
	tagRe      = regexp.MustCompile(`<([a-zA-Z0-9]+)(\s+[^>]*)?>`)
	attrRe     = regexp.MustCompile(`([a-zA-Z0-9_-]+)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+)))?`)
	classSplit = regexp.MustCompile(`\s+`)
)

func assertRenderedHTMLCompliant(t *testing.T, route, htmlText string, allowed map[string]bool, expectErr bool) {
	t.Helper()

	for _, required := range []string{
		`href="/static/bootstrap.min.css"`,
		`<html lang=`,
		`<header`,
		`<nav`,
		`<main`,
		`aria-current="page"`,
	} {
		if !strings.Contains(htmlText, required) {
			t.Errorf("%s missing required markup %q", route, required)
		}
	}

	if expectErr {
		if !strings.Contains(htmlText, `aria-invalid="true"`) || !strings.Contains(htmlText, `role="alert"`) {
			t.Errorf("%s error state missing aria-invalid or role=alert", route)
		}
	}

	labelTargets := make(map[string]bool)
	type formControl struct {
		tag     string
		id      string
		typ     string
		ariaLbl string
	}
	var controls []formControl

	matches := tagRe.FindAllStringSubmatch(htmlText, -1)
	for _, m := range matches {
		tagName := strings.ToLower(m[1])
		attrsRaw := m[2]

		if tagName == "script" || tagName == "style" || tagName == "iframe" {
			t.Errorf("%s rendered forbidden <%s> tag", route, tagName)
		}

		attrs := parseTagAttrs(attrsRaw)
		for k, v := range attrs {
			kl := strings.ToLower(k)
			if kl == "style" || strings.HasPrefix(kl, "on") || strings.HasPrefix(kl, "data-bs-") {
				t.Errorf("%s rendered forbidden attribute %q on <%s>", route, kl, tagName)
			}
			if (kl == "href" || kl == "action") && strings.HasPrefix(strings.ToLower(strings.TrimSpace(v)), "javascript:") {
				t.Errorf("%s rendered javascript: URI in %s", route, kl)
			}
			if kl == "rel" && strings.EqualFold(v, "stylesheet") {
				href := attrs["href"]
				if href != "/static/bootstrap.min.css" {
					t.Errorf("%s links external/custom stylesheet %q", route, href)
				}
			}
		}

		if classAttr, ok := attrs["class"]; ok {
			for _, cls := range classSplit.Split(strings.TrimSpace(classAttr), -1) {
				if cls == "" {
					continue
				}
				if !allowed[cls] {
					t.Errorf("%s uses non-Bootstrap class %q on <%s>", route, cls, tagName)
				}
			}
		}

		if tagName == "label" {
			if forID := attrs["for"]; forID != "" {
				labelTargets[forID] = true
			}
		}
		if tagName == "input" || tagName == "select" || tagName == "textarea" {
			controls = append(controls, formControl{
				tag:     tagName,
				id:      attrs["id"],
				typ:     strings.ToLower(attrs["type"]),
				ariaLbl: attrs["aria-label"],
			})
		}
	}

	for _, c := range controls {
		if c.tag == "input" && (c.typ == "hidden" || c.typ == "submit") {
			continue
		}
		if c.ariaLbl != "" {
			continue
		}
		if c.id == "" || !labelTargets[c.id] {
			t.Errorf("%s renders <%s id=%q> without matching <label for=%q> or aria-label", route, c.tag, c.id, c.id)
		}
	}
}

func parseTagAttrs(raw string) map[string]string {
	out := make(map[string]string)
	for _, m := range attrRe.FindAllStringSubmatch(raw, -1) {
		key := m[1]
		val := m[2]
		if val == "" {
			val = m[3]
		}
		if val == "" {
			val = m[4]
		}
		out[strings.ToLower(key)] = val
	}
	return out
}

func parseCSSClasses(cssText string) map[string]bool {
	classes := make(map[string]bool)
	cleaned := stripCSSComments(cssText)

	const (
		ctxRoot = iota
		ctxAtGroup
		ctxSkipBlock
	)

	stack := []int{ctxRoot}
	var sel strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	for i := 0; i < len(cleaned); i++ {
		ch := cleaned[i]
		top := stack[len(stack)-1]

		if escaped {
			if top != ctxSkipBlock {
				sel.WriteByte(ch)
			}
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			if top != ctxSkipBlock {
				sel.WriteByte(ch)
			}
			continue
		}
		if inSingle {
			if ch == '\'' {
				inSingle = false
			}
			if top != ctxSkipBlock {
				sel.WriteByte(ch)
			}
			continue
		}
		if inDouble {
			if ch == '"' {
				inDouble = false
			}
			if top != ctxSkipBlock {
				sel.WriteByte(ch)
			}
			continue
		}
		if ch == '\'' {
			inSingle = true
			if top != ctxSkipBlock {
				sel.WriteByte(ch)
			}
			continue
		}
		if ch == '"' {
			inDouble = true
			if top != ctxSkipBlock {
				sel.WriteByte(ch)
			}
			continue
		}

		switch ch {
		case '{':
			if top == ctxSkipBlock {
				stack = append(stack, ctxSkipBlock)
				continue
			}
			header := strings.TrimSpace(sel.String())
			sel.Reset()
			if strings.HasPrefix(header, "@") {
				lower := strings.ToLower(header)
				if strings.HasPrefix(lower, "@media") ||
					strings.HasPrefix(lower, "@supports") ||
					strings.HasPrefix(lower, "@container") ||
					strings.HasPrefix(lower, "@layer") {
					stack = append(stack, ctxAtGroup)
				} else {
					stack = append(stack, ctxSkipBlock)
				}
			} else {
				extractClassTokens(header, classes)
				stack = append(stack, ctxSkipBlock)
			}
		case '}':
			sel.Reset()
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		case ';':
			if top != ctxSkipBlock {
				sel.Reset()
			}
		default:
			if top != ctxSkipBlock {
				sel.WriteByte(ch)
			}
		}
	}

	return classes
}

func stripCSSComments(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

var cssClassTokenRe = regexp.MustCompile(`\.((?:[A-Za-z_-]|\\.)(?:[A-Za-z0-9_-]|\\.)*)`)

func extractClassTokens(selectorList string, out map[string]bool) {
	for _, m := range cssClassTokenRe.FindAllStringSubmatch(selectorList, -1) {
		raw := strings.ReplaceAll(m[1], `\`, "")
		if raw != "" {
			out[raw] = true
		}
	}
}
