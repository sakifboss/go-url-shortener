package security

import (
	"net/http"
)

// Middleware applies baseline HTTP security protections to every request.
//
// It provides:
//   - Request body size limiting
//   - MIME sniffing protection
//   - Clickjacking protection
//   - Referrer policy
//   - Permissions policy
//
// HSTS is intentionally not enabled because the current local server
// uses plain HTTP. HSTS should be enabled only after HTTPS is configured.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Limit request bodies to 1 MiB.
		// This prevents unnecessarily large requests from reaching handlers.
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		// Prevent browsers from MIME-sniffing responses.
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent the API from being embedded inside frames.
		w.Header().Set("X-Frame-Options", "DENY")

		// Reduce unnecessary referrer information.
		w.Header().Set(
			"Referrer-Policy",
			"strict-origin-when-cross-origin",
		)

		// Disable browser capabilities that GoShort does not need.
		w.Header().Set(
			"Permissions-Policy",
			"camera=(), microphone=(), geolocation=()",
		)

		next.ServeHTTP(w, r)
	})
}
