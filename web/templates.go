package web

import (
	"fmt"
	"net/http"
)

// HTML presentation templates for developer notices and error views.
const (
	unbuiltNoticeHTML = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Scoreboard Frontend Not Built</title>
</head>
<body style="font-family:sans-serif;background:#070b14;color:#e2e8f0;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;padding:1rem;box-sizing:border-box">
	<div style="background:#0f172a;padding:2rem;border-radius:12px;border:1px solid #1e293b;max-width:480px;width:100%">
		<h1 style="font-size:1.25rem;margin-top:0">⚠️ Scoreboard Frontend Not Built</h1>
		<p>Please build Vite production assets first by running:</p>
		<pre style="background:#1e293b;padding:1rem;border-radius:6px;color:#38bdf8;overflow-x:auto;white-space:pre-wrap"><code>pnpm --prefix web build</code></pre>
	</div>
</body>
</html>`

	exportNotFoundHTML = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Class Export Not Found</title>
</head>
<body style="font-family:sans-serif;background:#070b14;color:#e2e8f0;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;padding:1rem;box-sizing:border-box">
	<div style="background:#0f172a;padding:2rem;border-radius:12px;border:1px solid #1e293b;max-width:480px;width:100%;text-align:center">
		<h1 style="font-size:1.25rem;margin-top:0;color:#f43f5e">404 - Export File Missing</h1>
		<p style="color:#94a3b8">The requested class scoreboard export could not be found or has not been generated yet.</p>
	</div>
</body>
</html>`

	serverErrorHTML = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Internal Server Error</title>
</head>
<body style="font-family:sans-serif;background:#070b14;color:#e2e8f0;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;padding:1rem;box-sizing:border-box">
	<div style="background:#0f172a;padding:2rem;border-radius:12px;border:1px solid #1e293b;max-width:480px;width:100%;text-align:center">
		<h1 style="font-size:1.25rem;margin-top:0;color:#f59e0b">500 - Server Error</h1>
		<p style="color:#94a3b8">An internal server error occurred while processing the requested export asset.</p>
	</div>
</body>
</html>`
)

// writeHTMLResponse writes standard HTML error/notice headers and status codes.
func writeHTMLResponse(w http.ResponseWriter, status int, htmlContent string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprint(w, htmlContent)
}
