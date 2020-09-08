package e2e

import (
	"context"
	"crypto/subtle"
	"net"
	"net/http"
	"testing"
)

func TestExampleSecrets(t *testing.T) {
	server := &http.Server{Addr: ":8100", Handler: HTTPBasicAuth(http.FileServer(http.Dir("./../..")))}
	errCh := make(chan error)

	go func() {
		ln, err := net.Listen("tcp", server.Addr)
		errCh <- err

		server.Serve(ln)
	}()

	err := <-errCh
	if err != nil {
		t.Fatalf("Expected no error for server start")
	}

	example{Name: "secrets"}.Check(t)

	server.Shutdown(context.TODO())
}

func HTTPBasicAuth(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(user), []byte("admin")) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte("password")) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
			w.WriteHeader(401)
			w.Write([]byte("Unauthorized\n"))
			return
		}
		handler.ServeHTTP(w, r)
	}
}
