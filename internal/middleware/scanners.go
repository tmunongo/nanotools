package middleware

import (
	"net/http"
	"net/url"
	"strings"
)

// suspiciousExtensions are extensions commonly probed by automated scanners,
// none of which are used or served by nanotools.
var suspiciousExtensions = []string{
	".php", ".php3", ".php4", ".php5", ".php7", ".phtml",
	".asp", ".aspx", ".jsp", ".jspx", ".cgi",
	".bak", ".old", ".orig", ".save", ".swp",
	".sql", ".sqlite",
}

// suspiciousPathSubstrings are filenames and paths known to be targeted by bots
// looking for credentials, private keys, and admin exploits.
var suspiciousPathSubstrings = []string{
	"wp-config",
	"config.php",
	"configuration.php",
	"web.config",
	"database.yml",
	"secrets.yml",
	"credentials.json",
	"service-account.json",
	"client_secrets.json",
	"id_rsa",
	"id_ed25519",
	"id_dsa",
	"known_hosts",
	"authorized_keys",
	".bash_history",
	"/wp-admin",
	"/wp-includes",
	"/wp-content",
	"/wp-login",
	"/xmlrpc.php",
	"/phpmyadmin",
	"/pma/",
	"/adminer",
	"/actuator",
	"/_profiler",
	"/telescope",
	"/solr/",
	"/boaform",
	"/setup.cgi",
	"/cgi-bin/",
}

// decodeNestedPath decodes percent-encoded path strings iteratively using path semantics
// up to a bounded number of iterations to handle nested encoding safely.
func decodeNestedPath(s string) string {
	res := s
	for i := 0; i < 3; i++ {
		unescaped, err := url.PathUnescape(res)
		if err != nil || unescaped == res {
			break
		}
		res = unescaped
	}
	return strings.ToLower(res)
}

// IsSuspiciousPath checks whether a given request path or URI matches known
// bot scanning patterns (such as .env scans, credential probes, and exploit scripts).
// It inspects only request paths, ignoring query strings.
func IsSuspiciousPath(path, rawURI string) bool {
	// Strip query string from rawURI so query parameters do not trigger 404
	uriPath := rawURI
	if idx := strings.Index(uriPath, "?"); idx != -1 {
		uriPath = uriPath[:idx]
	}

	p := decodeNestedPath(strings.ToLower(path))
	u := decodeNestedPath(strings.ToLower(uriPath))

	// 1. Path traversal or null bytes in request path
	if strings.Contains(p, "..") || strings.Contains(u, "..") || strings.Contains(p, "\x00") || strings.Contains(u, "\x00") {
		return true
	}

	// 2. Sensitive path substrings (e.g. wp-config, credentials, id_rsa, admin tools)
	for _, substr := range suspiciousPathSubstrings {
		if strings.Contains(p, substr) || strings.Contains(u, substr) {
			return true
		}
	}

	// 3. Probing for dangerous / script extensions (checked only against request path p)
	for _, ext := range suspiciousExtensions {
		if strings.HasSuffix(p, ext) || strings.Contains(p, ext+"/") {
			return true
		}
	}

	// 4. Dotfiles, hidden VCS/system directories, and .env segment matches
	// Inspect both p and u segments
	for _, target := range []string{p, u} {
		clean := strings.Trim(target, "/")
		if clean == "" {
			continue
		}
		segments := strings.Split(clean, "/")
		for _, seg := range segments {
			// Match full .env segment or .env.* prefix (e.g. .env, .env.local, .env.backup)
			// Avoids matching non-secrets like foo.envelope or .environment-info
			if seg == ".env" || strings.HasPrefix(seg, ".env.") || strings.HasPrefix(seg, ".env_") || strings.HasPrefix(seg, ".env-") {
				return true
			}

			// Dotfile check: allow .well-known (RFC 8615) and benign names like .environment-info
			if strings.HasPrefix(seg, ".") && seg != ".well-known" && seg != ".environment-info" {
				return true
			}
		}
	}

	return false
}

// BlockBotScanners intercepts automated bot scans searching for .env, credentials,
// and exploit scripts before chi routes, logs, or renders templates for them.
// It immediately terminates the request with a lightweight 404 response to save
// memory, CPU, log volume, and server bandwidth.
func BlockBotScanners(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsSuspiciousPath(r.URL.Path, r.RequestURI) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Not Found\n"))
			return
		}

		next.ServeHTTP(w, r)
	})
}
