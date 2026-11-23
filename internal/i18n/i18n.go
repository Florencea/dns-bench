package i18n

import (
	"net/http"
	"strings"
)

const (
	LocaleZhTW       = "zh-TW"
	LocaleEnUS       = "en-US"
	LocaleCookieName = "dns_bench_locale"
)

// Messages defines the compile-time checked translation catalog.
type Messages struct {
	LangCode string
	AppTitle string

	UI struct {
		ChangeLanguage     string
		QueryMode          string
		QueryFormLabel     string
		DNSMode            string
		DoHMode            string
		DomainLabel        string
		DomainPlaceholder  string
		DomainHelp         string
		DomainInvalid      string
		SubmitBtn          string
		ResetBtn           string
		QuerySummaryPrefix string
		ServerInfoLabel    string
		DocumentLink       string
		DNSLabel           string
		DoHLabel           string
		ReadyHint          string
		NoResult           string
	}

	Errors struct {
		CSRFRejected           string
		CrossSiteQueryRejected string
		HostRejected           string
		InvalidRequest         string
	}
}

// ResolveLocale determines the preferred locale from cookie, Accept-Language, or default.
func ResolveLocale(r *http.Request, defaultLocale string) string {
	if r != nil {
		if c, err := r.Cookie(LocaleCookieName); err == nil {
			if normalized, ok := ParseLocale(c.Value); ok {
				return normalized
			}
		}
		if accept := strings.ToLower(strings.TrimSpace(r.Header.Get("Accept-Language"))); accept != "" {
			primary := strings.Split(accept, ",")[0]
			if strings.HasPrefix(strings.TrimSpace(primary), "zh") {
				return LocaleZhTW
			}
			if strings.HasPrefix(strings.TrimSpace(primary), "en") {
				return LocaleEnUS
			}
		}
	}
	if normalized, ok := ParseLocale(defaultLocale); ok {
		return normalized
	}
	return LocaleZhTW
}

// ParseLocale validates and normalizes a supported locale code.
func ParseLocale(raw string) (string, bool) {
	switch strings.TrimSpace(raw) {
	case LocaleZhTW:
		return LocaleZhTW, true
	case LocaleEnUS:
		return LocaleEnUS, true
	default:
		return "", false
	}
}

// Catalog returns the compile-time checked Messages dictionary for locale.
func Catalog(locale string) Messages {
	if locale == LocaleEnUS {
		return enUS
	}
	return zhTW
}

var zhTW = func() Messages {
	var m Messages
	m.LangCode = LocaleZhTW
	m.AppTitle = "DNS Bench"

	m.UI.ChangeLanguage = "English"
	m.UI.QueryMode = "查詢模式"
	m.UI.QueryFormLabel = "網域查詢表單"
	m.UI.DNSMode = "DNS"
	m.UI.DoHMode = "DNS over HTTPS"
	m.UI.DomainLabel = "網域名稱"
	m.UI.DomainPlaceholder = "例如：example.com"
	m.UI.DomainHelp = "輸入主機名稱以同時量測 IPv4 (A) 與 IPv6 (AAAA) 解析延遲"
	m.UI.DomainInvalid = "請輸入有效的網域名稱"
	m.UI.SubmitBtn = "查詢"
	m.UI.ResetBtn = "清除"
	m.UI.QuerySummaryPrefix = "查詢結果："
	m.UI.ServerInfoLabel = "解析器資訊"
	m.UI.DocumentLink = "說明文件"
	m.UI.DNSLabel = "DNS"
	m.UI.DoHLabel = "DNS over HTTPS"
	m.UI.ReadyHint = "尚未查詢"
	m.UI.NoResult = "無回應或逾時"

	m.Errors.CSRFRejected = "安全驗證失敗：拒絕跨站請求"
	m.Errors.CrossSiteQueryRejected = "安全驗證失敗：拒絕跨站觸發本機 DNS 查詢"
	m.Errors.HostRejected = "拒絕非本機回環的主機標頭"
	m.Errors.InvalidRequest = "請求格式或參數無效"
	return m
}()

var enUS = func() Messages {
	var m Messages
	m.LangCode = LocaleEnUS
	m.AppTitle = "DNS Bench"

	m.UI.ChangeLanguage = "繁體中文"
	m.UI.QueryMode = "Query Mode"
	m.UI.QueryFormLabel = "Domain Query Form"
	m.UI.DNSMode = "DNS"
	m.UI.DoHMode = "DNS over HTTPS"
	m.UI.DomainLabel = "Domain"
	m.UI.DomainPlaceholder = "e.g. example.com"
	m.UI.DomainHelp = "Enter a hostname to measure IPv4 (A) and IPv6 (AAAA) lookup latency"
	m.UI.DomainInvalid = "Please enter a valid domain name"
	m.UI.SubmitBtn = "Submit"
	m.UI.ResetBtn = "Reset"
	m.UI.QuerySummaryPrefix = "Query results for"
	m.UI.ServerInfoLabel = "Resolver details"
	m.UI.DocumentLink = "Document"
	m.UI.DNSLabel = "DNS"
	m.UI.DoHLabel = "DNS over HTTPS"
	m.UI.ReadyHint = "Ready"
	m.UI.NoResult = "No answer or timed out"

	m.Errors.CSRFRejected = "Security check failed: cross-origin request rejected"
	m.Errors.CrossSiteQueryRejected = "Security check failed: cross-site local DNS query rejected"
	m.Errors.HostRejected = "Non-loopback Host header rejected"
	m.Errors.InvalidRequest = "Invalid request parameters"
	return m
}()
