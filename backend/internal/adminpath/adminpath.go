package adminpath

import (
	"crypto/rand"
	"strings"
)

const letters = "abcdefghjkmnpqrstuvwxyz23456789"

var reserved = map[string]struct{}{
	"admin": {}, "api": {}, "assets": {}, "auth": {}, "category": {},
	"article": {}, "page": {}, "uploads": {}, "static": {}, "public": {},
	"login": {}, "sitemap": {}, "robots": {}, "favicon": {}, "index": {},
	"system": {},
}

func Normalize(raw string) (string, string) {
	raw = strings.ToLower(strings.Trim(strings.TrimSpace(raw), "/"))
	if raw == "admin" {
		return "", "后台路径不能使用 admin"
	}
	if len(raw) < 3 || len(raw) > 32 {
		return "", "后台路径需要 3 到 32 位"
	}
	for i, r := range raw {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if i == 0 {
			ok = r >= 'a' && r <= 'z'
		}
		if !ok {
			return "", "后台路径只能使用小写字母和数字，并且以字母开头"
		}
	}
	if _, blocked := reserved[raw]; blocked {
		return "", "这个后台路径已被系统占用"
	}
	return raw, ""
}

func Random() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "panelk7m2"
	}
	out := make([]byte, len(buf))
	for i, b := range buf {
		out[i] = letters[int(b)%len(letters)]
	}
	if _, msg := Normalize(string(out)); msg != "" {
		return "panelk7m2"
	}
	return string(out)
}
