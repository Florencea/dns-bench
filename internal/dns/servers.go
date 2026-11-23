package dns

// Mode selects the query transport protocol.
type Mode string

const (
	ModeDNS Mode = "dns"
	ModeDoH Mode = "doh"
)

// RecordType selects IPv4 (A) or IPv6 (AAAA) lookup.
type RecordType string

const (
	TypeA    RecordType = "A"
	TypeAAAA RecordType = "AAAA"
)

// Server describes a preset DNS or DoH resolver.
type Server struct {
	Title       string
	Description string
	Document    string
	DoHEndpoint string
	IP          string
}

var defaultServers = []Server{
	{
		Title:       "Google Public DNS",
		Description: "A free, global DNS resolution service that you can use as an alternative to your current DNS provider.",
		Document:    "https://developers.google.com/speed/public-dns",
		DoHEndpoint: "https://dns.google/resolve",
		IP:          "8.8.8.8",
	},
	{
		Title:       "Cloudflare 1.1.1.1",
		Description: "A public DNS resolver operated by Cloudflare that offers a fast and private way to browse the Internet.",
		Document:    "https://1.1.1.1/dns/",
		DoHEndpoint: "https://cloudflare-dns.com/dns-query",
		IP:          "1.1.1.1",
	},
	{
		Title:       "Quad 101",
		Description: "A TWNIC's experimental Public DNS project. TWNIC (Taiwan Network Information Center) is a ccTLD (country-code Top Level Domain) registry operator. TWNIC runs one of the world's fastest DNS infrastructure",
		Document:    "https://101.101.101.101/index_en.html",
		DoHEndpoint: "https://dns.twnic.tw/dns-query",
		IP:          "101.101.101.101",
	},
	{
		Title:       "NextDNS",
		Description: "A service that allows you to protect your online browsing from unwanted content, malware, and privacy threats.",
		Document:    "https://nextdns.io/",
		DoHEndpoint: "https://dns.nextdns.io/dns-query",
		IP:          "45.90.28.170",
	},
	{
		Title:       "OpenDNS",
		Description: "An American company providing Domain Name System (DNS) resolution services—with features such as phishing protection, optional content filtering, and DNS lookup in its DNS servers—and a cloud computing security product suite, Umbrella, designed to protect enterprise customers from malware, botnets, phishing, and targeted online attacks.",
		Document:    "https://www.opendns.com/",
		DoHEndpoint: "-",
		IP:          "208.67.222.222",
	},
	{
		Title:       "Quad9",
		Description: "A free service that replaces your default ISP or enterprise Domain Name Server (DNS) configuration.",
		Document:    "https://www.quad9.net/",
		DoHEndpoint: "-",
		IP:          "9.9.9.9",
	},
	{
		Title:       "Adguard",
		Description: "A reliable way to block ads on the Internet without installing a mandatory application.",
		Document:    "https://adguard-dns.io/en/public-dns.html",
		DoHEndpoint: "-",
		IP:          "94.140.14.14",
	},
	{
		Title:       "Hinet 1",
		Description: "A public DNS resolver operated by ISP Chunghwa Telecom (dns.hinet.net)",
		Document:    "-",
		DoHEndpoint: "-",
		IP:          "168.95.1.1",
	},
	{
		Title:       "Hinet 2",
		Description: "A public DNS resolver operated by ISP Chunghwa Telecom (hntp1.hinet.net)",
		Document:    "-",
		DoHEndpoint: "-",
		IP:          "168.95.192.1",
	},
	{
		Title:       "AliDNS",
		Description: "A public DNS resolver operated by Alibaba Cloud (AliDNS)",
		Document:    "https://www.alidns.com/",
		DoHEndpoint: "-",
		IP:          "223.5.5.5",
	},
	{
		Title:       "BaiduDNS",
		Description: "A public DNS resolver operated by Baidu (BaiduDNS)",
		Document:    "https://dudns.baidu.com/",
		DoHEndpoint: "-",
		IP:          "180.76.76.76",
	},
	{
		Title:       "System Default",
		Description: "DNS provided by your operating system",
		Document:    "-",
		DoHEndpoint: "-",
		IP:          "-",
	},
}

// DefaultServers returns a fresh copy of the built-in DNS/DoH server list.
func DefaultServers() []Server {
	out := make([]Server, len(defaultServers))
	copy(out, defaultServers)
	return out
}

// FilterServers returns servers eligible for the selected query mode.
func FilterServers(servers []Server, mode Mode) []Server {
	if mode != ModeDoH {
		out := make([]Server, len(servers))
		copy(out, servers)
		return out
	}
	out := make([]Server, 0, len(servers))
	for _, s := range servers {
		if s.DoHEndpoint != "-" && s.DoHEndpoint != "" {
			out = append(out, s)
		}
	}
	return out
}
