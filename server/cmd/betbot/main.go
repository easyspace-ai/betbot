package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><title>Betbot Demo</title></head>
<body>
<h1>🤖 Betbot Demo Server</h1>
<p>Version: %s | Commit: %s</p>
</body>
</html>
		`, version, commit)))
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	fmt.Printf("Betbot server v%s starting on port %s...\n", version, port)
	fmt.Printf("Go: %s, OS/Arch: %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	http.ListenAndServe(":"+port, nil)
}