package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

var shutdownInitiated = atomic.Bool{}
var shutdownTimer atomic.Pointer[time.Timer]
var gracefulShutdown = os.Getenv("GRACEFUL_SHUTDOWN") == "true"
var shutdownSleepDuration = 10 * time.Second
var numConnections atomic.Int32

const clientSideIdleTimeout = 15 * time.Second

func withLastModified(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))
		next.ServeHTTP(w, r)
	})
}

type connectionCloseWriter struct {
	http.ResponseWriter
	headerWritten bool
}

func (w *connectionCloseWriter) injectHeader() {
	if !w.headerWritten {
		w.headerWritten = true
		if shutdownInitiated.Load() {
			w.ResponseWriter.Header().Set("Connection", "close")
		}
	}
}

func (w *connectionCloseWriter) WriteHeader(code int) {
	w.injectHeader()
	w.ResponseWriter.WriteHeader(code)
}

func (w *connectionCloseWriter) Write(b []byte) (int, error) {
	// Write implicitly sends a 200 WriteHeader if not yet called,
	// so we inject before that happens.
	w.injectHeader()
	return w.ResponseWriter.Write(b)
}

func (w *connectionCloseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *connectionCloseWriter) ReadFrom(src io.Reader) (int64, error) {
	w.injectHeader()
	if rf, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		return rf.ReadFrom(src)
	}
	// Fallback: copy manually via Write (which won't re-inject thanks to the guard).
	return io.Copy(w.ResponseWriter, src)
}

func graceful(next http.Handler) http.Handler {
	if !gracefulShutdown {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cw := &connectionCloseWriter{
			ResponseWriter: w,
		}
		next.ServeHTTP(cw, r)
	})
}

func buildInverseDiscreteCDF(values []time.Duration, probabilities []float32) func() time.Duration {
	cdf := make([]float32, len(probabilities))
	var cumProb float32 = 0.0
	for i, p := range probabilities {
		cumProb += p
		cdf[i] = cumProb
	}
	return func() time.Duration {
		r := rand.Float32()
		for i, cp := range cdf {
			if r <= cp {
				return values[i]
			}
		}
		return values[len(values)-1]
	}
}

func sleep(w http.ResponseWriter, r *http.Request) {
	lo, hi := 50*time.Millisecond, 1*time.Second
	minD, maxD := r.URL.Query().Get("min"), r.URL.Query().Get("max")
	pdf := r.URL.Query().Get("pdf")
	if pdf != "" {
		var values []time.Duration
		var probabilities []float32
		pairs := strings.Split(pdf, ",")
		var totalProb float32 = 0.0
		for _, pair := range pairs {
			parts := strings.SplitN(pair, ":", 2)
			if len(parts) != 2 {
				_, _ = fmt.Fprintf(os.Stderr, "%v: invalid pdf pair: %v\n", time.Now().Format(time.RFC3339), pair)
				http.Error(w, "Invalid pdf parameter\n", http.StatusBadRequest)
				return
			}
			durStr, probStr := parts[0], parts[1]
			dur, err := time.ParseDuration(durStr)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "%v: failed to parse duration in pdf: %v\n", time.Now().Format(time.RFC3339), err)
				http.Error(w, "Failed to parse duration in pdf\n", http.StatusBadRequest)
				return
			}
			var prob float32
			_, err = fmt.Sscanf(probStr, "%f", &prob)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "%v: failed to parse probability in pdf: %v\n", time.Now().Format(time.RFC3339), err)
				http.Error(w, "Failed to parse probability in pdf\n", http.StatusBadRequest)
				return
			}
			values = append(values, dur)
			probabilities = append(probabilities, prob)
			totalProb += prob
		}
		if totalProb <= 0.0 {
			_, _ = fmt.Fprintf(os.Stderr, "%v: total probability in pdf must be greater than 0\n", time.Now().Format(time.RFC3339))
			http.Error(w, "Total probability in pdf must be greater than 0\n", http.StatusBadRequest)
			return
		}
		invTotalProb := float32(1.0) / totalProb
		for i := range probabilities {
			probabilities[i] *= invTotalProb
		}
		inverseCDF := buildInverseDiscreteCDF(values, probabilities)
		sleepDuration := inverseCDF()
		time.Sleep(sleepDuration)
		_, _ = fmt.Fprintf(w, "Slept for %v\n", sleepDuration)
		return
	}
	if d, err := time.ParseDuration(minD); err == nil {
		lo = d
	} else if minD != "" {
		_, _ = fmt.Fprintf(os.Stderr, "%v: NewRequest err: %v\n", time.Now().Format(time.RFC3339), err)
		http.Error(w, "Failed to parse min duration\n", http.StatusBadRequest)
		return
	}
	if d, err := time.ParseDuration(maxD); err == nil {
		hi = d
	} else if maxD != "" {
		_, _ = fmt.Fprintf(os.Stderr, "%v: NewRequest err: %v\n", time.Now().Format(time.RFC3339), err)
		http.Error(w, "Failed to parse max duration\n", http.StatusBadRequest)
		return
	}
	sleepDuration := lo + time.Duration(rand.Int63n(int64(hi-lo+1)))
	time.Sleep(sleepDuration)
	_, _ = fmt.Fprintf(w, "Slept for %v\n", sleepDuration)
}

func forwardTraceHeaders(src, dest http.Header) {
	traceHeaders := []string{
		"X-Request-ID",
		"X-B3-Traceid",
		"X-B3-Spanid",
		"X-B3-Parentspanid",
		"X-B3-Sampled",
		"X-B3-Flags",
		"X-Ot-Span-Context",
		"Traceparent",
		"Tracestate",
		"B3",
	}
	for _, h := range traceHeaders {
		if v := src.Get(h); v != "" {
			dest.Set(h, v)
		}
	}
}

func proxy(service string, w http.ResponseWriter, r *http.Request, client *http.Client) {
	req, err := http.NewRequest(r.Method, "http://"+service+"/", http.NoBody)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v: NewRequest err: %v\n", time.Now().Format(time.RFC3339), err)
		http.Error(w, "Failed to create request\n", http.StatusInternalServerError)
		return
	}
	forwardTraceHeaders(r.Header, req.Header)
	req.URL.Path = r.URL.Path[1+len(service):]
	req.URL.RawQuery = r.URL.RawQuery
	resp, err := client.Do(req)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v: request to envoy failed: %v\n", time.Now().Format(time.RFC3339), err)
		http.Error(w, "Request to envoy failed\n", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	if _, err = io.Copy(w, resp.Body); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v: failed to copy response body: %v\n", time.Now().Format(time.RFC3339), err)
		return
	}
}

func ready(w http.ResponseWriter) {
	if shutdownInitiated.Load() {
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		_, _ = w.Write([]byte("OK"))
	}
}

func status(w http.ResponseWriter, r *http.Request) {
	codeStr := r.URL.Query().Get("code")
	if codeStr == "" {
		http.Error(w, "Missing code parameter\n", http.StatusBadRequest)
		return
	}
	var code int
	_, err := fmt.Sscanf(codeStr, "%d", &code)
	if err != nil || code < 100 || code > 599 {
		http.Error(w, "Invalid code parameter\n", http.StatusBadRequest)
		return
	}
	w.WriteHeader(code)
	_, _ = fmt.Fprintf(w, "Returned status code %d\n", code)
}

func registerHandlers(mux *http.ServeMux, client *http.Client) {
	mux.Handle("/ready", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ready(w)
	}))
	mux.Handle("/sleep", graceful(withLastModified(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sleep(w, r)
	}))))
	mux.Handle("/status", graceful(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status(w, r)
	})))
	mux.Handle("/envoy/", graceful(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy("envoy", w, r, client)
	})))
	mux.Handle("/nginx/", graceful(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy("nginx", w, r, client)
	})))
	mux.Handle("/varnish/", graceful(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy("varnish", w, r, client)
	})))
	mux.Handle("/node-demo/", graceful(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy("node-demo", w, r, client)
	})))
	mux.Handle("/java-demo/", graceful(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy("java-demo", w, r, client)
	})))
	// add default 404 handler
	mux.Handle("/", graceful(http.NotFoundHandler()))
}

func shutdown(server *http.Server) {
	// sleep for shutdownSleepDuration
	_, _ = fmt.Printf("%v: sleeping for %v before starting shutdown...\n", time.Now().Format(time.RFC3339), shutdownSleepDuration)
	time.Sleep(shutdownSleepDuration)

	// initiate shutdown
	shutdownInitiated.Store(true)
	if gracefulShutdown {
		doGracefulShutdown()
	}
	_, _ = fmt.Printf("%v: shutting down server...\n", time.Now().Format(time.RFC3339))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v: server shutdown error: %v\n", time.Now().Format(time.RFC3339), err)
	}
	_, _ = fmt.Printf("%v: server exited properly\n", time.Now().Format(time.RFC3339))
}

func doGracefulShutdown() {
	_, _ = fmt.Printf("%v: initiating graceful shutdown...\n", time.Now().Format(time.RFC3339))
	// let all incoming requests know that shutdown is initiated by
	// responding with "Connection: close" such that they don't attempt
	// to reuse connections.
	gracefulChan := make(chan struct{})
	shutdownTimer = atomic.Pointer[time.Timer]{}
	shutdownTimer.Store(time.AfterFunc(clientSideIdleTimeout, func() {
		_, _ = fmt.Printf("%v: graceful shutdown timeout reached, forcing exit\n", time.Now().Format(time.RFC3339))
		close(gracefulChan)
	}))
	// Check every 500ms if there are active connections and abort the drain period if either
	// there are no active connections or the shutdown timer has fired.
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				n := numConnections.Load()
				if n == 0 {
					_, _ = fmt.Printf("%v: no active connections remaining\n", time.Now().Format(time.RFC3339))
					close(gracefulChan)
					return
				} else {
					_, _ = fmt.Printf("%v: %d active connections remaining...\n", time.Now().Format(time.RFC3339), n)
				}
			case <-gracefulChan:
				return
			}
		}
	}()
	// wait for graceful shutdown to complete
	<-gracefulChan
}

func main() {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Tune the Transport to allow more concurrent connections.
	// This is to exacerbate the problems we will demonstrate later.
	transport.MaxIdleConns = 200
	transport.MaxIdleConnsPerHost = 200
	// Lower the client-side idle timeout from 90s to 4s to be
	// compatible with all known servers, like:
	// - Node.js with 5s (or 6s) timeout
	// - Tomcat with 60s timeout
	// - Jetty with 30s timeout
	transport.IdleConnTimeout = 4 * time.Second
	client := &http.Client{
		Transport: transport,
	}
	server := &http.Server{
		Addr: ":8080",
		ConnState: func(conn net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				numConnections.Add(1)
			case http.StateClosed, http.StateHijacked:
				numConnections.Add(-1)
			default:
				// do nothing
			}
		},
	}
	mux := http.NewServeMux()
	server.Handler = mux
	registerHandlers(mux, client)

	// set up signal handling for graceful shutdown
	sigs := make(chan os.Signal, 1)
	done := make(chan struct{}, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigs
		_, _ = fmt.Printf("%v: received signal: %v\n", time.Now().Format(time.RFC3339), sig)
		done <- struct{}{}
	}()

	// start server
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			_, _ = fmt.Fprintf(os.Stderr, "%v: server error: %v\n", time.Now().Format(time.RFC3339), err)
		}
	}()

	// wait for signal to shutdown
	<-done

	shutdown(server)
}
