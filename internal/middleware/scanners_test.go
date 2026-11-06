package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsSuspiciousPath(t *testing.T) {
	scannerCases := []struct {
		path   string
		rawURI string
	}{
		{"/.env", "/.env"},
		{"/.env.production", "/.env.production"},
		{"/.env.local", "/.env.local"},
		{"/.env.backup", "/.env.backup"},
		{"/.env.save", "/.env.save"},
		{"/core/.env", "/core/.env"},
		{"/api/.env", "/api/.env"},
		{"/vendor/.env", "/vendor/.env"},
		{"/.git", "/.git"},
		{"/.git/config", "/.git/config"},
		{"/.git/HEAD", "/.git/HEAD"},
		{"/.svn/all-wcprops", "/.svn/all-wcprops"},
		{"/.hg/store", "/.hg/store"},
		{"/.aws/credentials", "/.aws/credentials"},
		{"/.ssh/id_rsa", "/.ssh/id_rsa"},
		{"/.DS_Store", "/.DS_Store"},
		{"/wp-config.php", "/wp-config.php"},
		{"/config.php", "/config.php"},
		{"/database.yml", "/database.yml"},
		{"/secrets.yml", "/secrets.yml"},
		{"/credentials.json", "/credentials.json"},
		{"/service-account.json", "/service-account.json"},
		{"/id_rsa", "/id_rsa"},
		{"/id_ed25519", "/id_ed25519"},
		{"/wp-admin", "/wp-admin"},
		{"/wp-admin/install.php", "/wp-admin/install.php"},
		{"/wp-login.php", "/wp-login.php"},
		{"/xmlrpc.php", "/xmlrpc.php"},
		{"/phpmyadmin", "/phpmyadmin"},
		{"/phpmyadmin/index.php", "/phpmyadmin/index.php"},
		{"/pma/", "/pma/"},
		{"/adminer.php", "/adminer.php"},
		{"/actuator/health", "/actuator/health"},
		{"/solr/admin/info/system", "/solr/admin/info/system"},
		{"/boaform/admin/formLogin", "/boaform/admin/formLogin"},
		{"/setup.cgi", "/setup.cgi"},
		{"/cgi-bin/test.cgi", "/cgi-bin/test.cgi"},
		{"/backup.sql", "/backup.sql"},
		{"/dump.sqlite", "/dump.sqlite"},
		{"/index.php", "/index.php"},
		{"/info.php", "/info.php"},
		{"/default.asp", "/default.asp"},
		{"/test.aspx", "/test.aspx"},
		{"/app.jsp", "/app.jsp"},
		{"/config.bak", "/config.bak"},
		{"/app.old", "/app.old"},
		{"/..%2f..%2fetc/passwd", "/..%2f..%2fetc/passwd"},
		{"/%252e%252e/%252e%252e/etc/passwd", "/%252e%252e/%252e%252e/etc/passwd"},
	}

	for _, tc := range scannerCases {
		if !IsSuspiciousPath(tc.path, tc.rawURI) {
			t.Errorf("expected path %q (rawURI %q) to be identified as suspicious, but it was not", tc.path, tc.rawURI)
		}
	}

	legitimateCases := []struct {
		path   string
		rawURI string
	}{
		{"/", "/"},
		{"/health", "/health"},
		{"/tools/json-formatter", "/tools/json-formatter"},
		{"/tools/base64", "/tools/base64"},
		{"/tools/uuid", "/tools/uuid"},
		{"/tools/qr-code", "/tools/qr-code"},
		{"/tools/slugify", "/tools/slugify"},
		{"/tools/image-converter", "/tools/image-converter"},
		{"/tools/video-downloader", "/tools/video-downloader"},
		{"/tools/pdf-to-images", "/tools/pdf-to-images"},
		{"/tools/pdf-joiner", "/tools/pdf-joiner"},
		{"/tools/pdf-unlocker", "/tools/pdf-unlocker"},
		{"/api/tools/json-format", "/api/tools/json-format"},
		{"/api/tools/base64/encode", "/api/tools/base64/encode"},
		{"/api/tools/base64/decode", "/api/tools/base64/decode"},
		{"/api/tools/uuid/generate", "/api/tools/uuid/generate"},
		{"/api/tools/qr/generate", "/api/tools/qr/generate"},
		{"/api/tools/slugify", "/api/tools/slugify"},
		{"/api/tools/image/convert", "/api/tools/image/convert"},
		{"/api/tools/video/info", "/api/tools/video/info"},
		{"/api/tools/video/download", "/api/tools/video/download"},
		{"/api/tools/pdf/to-images", "/api/tools/pdf/to-images"},
		{"/api/tools/pdf/join", "/api/tools/pdf/join"},
		{"/api/tools/pdf/unlock", "/api/tools/pdf/unlock"},
		{"/static/css/styles.css", "/static/css/styles.css"},
		{"/static/js/main.js", "/static/js/main.js"},
		{"/static/images/logo.png", "/static/images/logo.png"},
		{"/favicon.ico", "/favicon.ico"},
		{"/robots.txt", "/robots.txt"},
		{"/.well-known/security.txt", "/.well-known/security.txt"},
		// Benign paths containing 'envelope' or '.environment-info'
		{"/tools/slugify", "/tools/slugify?name=foo.envelope"},
		{"/environment-info", "/environment-info"},
		{"/.environment-info", "/.environment-info"},
		// Legitimate requests with query parameters that contain consecutive dots or probe names
		{"/tools/slugify", "/tools/slugify?input=a..b"},
		{"/tools/slugify", "/tools/slugify?range=1..10"},
		{"/tools/json-formatter", "/tools/json-formatter?file=.env"},
		{"/tools/base64", "/tools/base64?data=test.php"},
		{"/tools/base64", "/tools/base64?q=admin.bak"},
	}

	for _, tc := range legitimateCases {
		if IsSuspiciousPath(tc.path, tc.rawURI) {
			t.Errorf("expected legitimate path %q (rawURI %q) to be allowed, but it was flagged as suspicious", tc.path, tc.rawURI)
		}
	}
}

func TestBlockBotScannersMiddleware(t *testing.T) {
	downstreamCalled := false
	handler := BlockBotScanners(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downstreamCalled = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))

	// Test 1: Bot scan request should be stopped at middleware with 404
	req := httptest.NewRequest(http.MethodGet, "/.env", nil)
	rec := httptest.NewRecorder()

	downstreamCalled = false
	handler.ServeHTTP(rec, req)

	if downstreamCalled {
		t.Error("expected downstream handler NOT to be called for scanner request")
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}

	// Test 2: Legitimate request should proceed to downstream
	req = httptest.NewRequest(http.MethodGet, "/tools/json-formatter?input=a..b", nil)
	rec = httptest.NewRecorder()

	downstreamCalled = false
	handler.ServeHTTP(rec, req)

	if !downstreamCalled {
		t.Error("expected downstream handler to be called for legitimate request")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}
