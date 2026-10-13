package hostutil

import (
	"net"
	"net/http"
	"strings"
)

// Normalize 把访问地址收成纯主机名：去掉协议、路径、端口和大小写差异。
func Normalize(raw string) string {
	host := strings.TrimSpace(strings.ToLower(raw))
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	if i := strings.IndexAny(host, "/?#"); i >= 0 {
		host = host[:i]
	}
	host = strings.TrimSuffix(host, ".")
	if name, _, err := net.SplitHostPort(host); err == nil {
		host = name
	}
	return host
}

// Valid 判断规范化之后的主机名能否作为站点绑定。
func Valid(host string) bool {
	if host == "" || len(host) > 253 {
		return false
	}
	if strings.ContainsAny(host, " \t/\\?#@") {
		return false
	}
	return true
}

// FromRequest 决定这次请求属于哪个域名。
// 预览头优先，其次是查询参数，最后才是浏览器真实 Host。
func FromRequest(r *http.Request) string {
	if host := strings.TrimSpace(r.Header.Get("X-Site-Host")); host != "" {
		return Normalize(host)
	}
	if host := strings.TrimSpace(r.URL.Query().Get("host")); host != "" {
		return Normalize(host)
	}
	return Normalize(r.Host)
}
