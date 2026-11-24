package kit_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arvo-health/kit"
	"github.com/gofiber/fiber/v2"
)

// These tests describe the required behavior for HTTP/1.1 Host handling on the
// Fiber server this module documents and runs. They fail while fasthttp 1.62.0
// accepts a Host count other than one and lets the handler observe the last
// value. The proxy in front of that server forwards the original bytes and,
// like a shared cache, stores a successful response under the first Host.

// Duplicate Content-Length is already rejected by fasthttp 1.62.0 during header
// parsing, before Fiber calls the application handler. This locks the
// observation used by TestHTTP11HostCountMustBeOne: a parse rejection is a
// 400 from this server and does not enter the handler or the first-host cache.
func TestDuplicateContentLengthIsRejectedBeforeHandler(t *testing.T) {
	raw := "GET / HTTP/1.1\r\nHost: public.example\r\nContent-Length: 0\r\nContent-Length: 1\r\nConnection: close\r\n\r\n"
	assertHostCount(t, raw, http.StatusBadRequest, "", false)
}

func TestHTTP11HostCountMustBeOne(t *testing.T) {
	cases := []struct {
		name       string
		method     string
		target     string
		hosts      []string
		body       string
		wantStatus int
		wantBody   string
		wantCached bool
	}{
		{
			name:       "one host is served under that host",
			method:     http.MethodGet,
			target:     "/",
			hosts:      []string{"public.example"},
			wantStatus: http.StatusOK,
			wantBody:   "served-for:public.example",
			wantCached: true,
		},
		{
			name:       "two hosts must be rejected before the handler",
			method:     http.MethodGet,
			target:     "/",
			hosts:      []string{"public.example", "trusted.example"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "two identical hosts must be rejected before the handler",
			method:     http.MethodGet,
			target:     "/",
			hosts:      []string{"public.example", "public.example"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "three hosts must be rejected before the handler",
			method:     http.MethodGet,
			target:     "/",
			hosts:      []string{"public.example", "other.example", "trusted.example"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing host must be rejected before the handler",
			method:     http.MethodGet,
			target:     "/",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "post body parsing shares the rejection",
			method:     http.MethodPost,
			target:     "/users",
			hosts:      []string{"public.example", "trusted.example"},
			body:       `{"name":"ada"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "healthcheck route shares the rejection",
			method:     http.MethodGet,
			target:     "/healthcheck/live",
			hosts:      []string{"public.example", "trusted.example"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			raw := request(tt.method, tt.target, tt.hosts, tt.body)
			assertHostCount(t, raw, tt.wantStatus, tt.wantBody, tt.wantCached)
		})
	}

	t.Run("mixed case host field names must be rejected before the handler", func(t *testing.T) {
		raw := requestWithHeaderNames(http.MethodGet, "/", []headerLine{
			{name: "host", value: "public.example"},
			{name: "HOST", value: "trusted.example"},
		}, "")
		assertHostCount(t, raw, http.StatusBadRequest, "", false)
	})
}

func assertHostCount(t *testing.T, raw string, wantStatus int, wantBody string, wantCached bool) {
	t.Helper()

	origin, entered := startOrigin(t)
	proxy := startFirstHostProxy(t, origin)

	// Confirm the origin is accepting before the case under test, then clear
	// the handler flag so only the forwarded request can set it.
	waitForOrigin(t, origin)
	entered.Store(false)

	status, body, forwarded := proxy.roundTrip(t, raw)
	if !bytes.Equal(forwarded, []byte(raw)) {
		t.Fatalf("proxy rewrote the request before the origin parsed it\nforwarded: %q\noriginal: %q", forwarded, raw)
	}

	first := firstHostValue(raw)
	cached, stored := proxy.cached(first)
	wantEntered := wantStatus == http.StatusOK

	if entered.Load() != wantEntered {
		t.Fatalf("handler entered=%v, want %v; status=%d body=%q cacheHit=%v cached=%q",
			entered.Load(), wantEntered, status, body, stored, cached)
	}
	if status != wantStatus {
		t.Fatalf("status=%d, want %d; body=%q; handler entered=%v", status, wantStatus, body, entered.Load())
	}
	if wantBody != "" && body != wantBody {
		t.Fatalf("body=%q, want %q", body, wantBody)
	}
	if strings.Contains(body, "served-for:") && wantStatus != http.StatusOK {
		t.Fatalf("origin built a host-specific response after it should have refused: %q", body)
	}
	if wantCached {
		if !stored || cached != wantBody {
			t.Fatalf("cache[%q]=%q stored=%v, want %q", first, cached, stored, wantBody)
		}
		return
	}
	if stored {
		t.Fatalf("cache stored a successful response for first host %q: %q (origin status %d, body %q)",
			first, cached, status, body)
	}
}

func startOrigin(t *testing.T) (string, *atomic.Bool) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{
		ErrorHandler:          kit.ErrorHandler(logger),
		DisableStartupMessage: true,
	})

	entered := &atomic.Bool{}
	app.Use(func(c *fiber.Ctx) error {
		entered.Store(true)
		return c.Next()
	})
	app.Use(kit.LoggerMiddleware(logger))
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("served-for:" + c.Hostname())
	})
	app.Post("/users", func(c *fiber.Ctx) error {
		var body struct {
			Name string `json:"name" validate:"required"`
		}
		if err := kit.ParseRequestBody(&body, c, kit.NewValidator()); err != nil {
			return err
		}
		return c.SendString("served-for:" + c.Hostname())
	})
	app.Group("/healthcheck", kit.HealthCheckMiddleware())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() { _ = app.Listener(ln) }()
	return ln.Addr().String(), entered
}

func waitForOrigin(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var last error
	raw := request(http.MethodGet, "/", []string{"ready.example"}, "")
	for time.Now().Before(deadline) {
		status, _, err := dialRoundTrip(addr, raw)
		if err == nil && status == http.StatusOK {
			return
		}
		last = err
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("origin did not accept a single-host request: %v", last)
}

type headerLine struct {
	name  string
	value string
}

func request(method, target string, hosts []string, body string) string {
	lines := make([]headerLine, len(hosts))
	for i, host := range hosts {
		lines[i] = headerLine{name: "Host", value: host}
	}
	return requestWithHeaderNames(method, target, lines, body)
}

func requestWithHeaderNames(method, target string, headers []headerLine, body string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s HTTP/1.1\r\n", method, target)
	for _, h := range headers {
		fmt.Fprintf(&b, "%s: %s\r\n", h.name, h.value)
	}
	if body != "" {
		fmt.Fprintf(&b, "Content-Type: application/json\r\nContent-Length: %d\r\n", len(body))
	}
	b.WriteString("Connection: close\r\n\r\n")
	b.WriteString(body)
	return b.String()
}

func firstHostValue(raw string) string {
	lines := strings.Split(raw, "\r\n")
	for _, line := range lines[1:] {
		if line == "" {
			return ""
		}
		name, value, ok := strings.Cut(line, ":")
		if ok && strings.EqualFold(strings.TrimSpace(name), "Host") {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type firstHostProxy struct {
	addr      string
	cache     atomic.Value // map[string]string
	forwarded chan []byte
}

func startFirstHostProxy(t *testing.T, origin string) *firstHostProxy {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("proxy listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	p := &firstHostProxy{
		addr:      ln.Addr().String(),
		forwarded: make(chan []byte, 1),
	}
	p.cache.Store(map[string]string{})

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go p.serve(conn, origin)
		}
	}()
	return p
}

func (p *firstHostProxy) roundTrip(t *testing.T, raw string) (int, string, []byte) {
	t.Helper()
	status, body, err := dialRoundTrip(p.addr, raw)
	if err != nil {
		t.Fatalf("proxy round trip: %v", err)
	}
	select {
	case forwarded := <-p.forwarded:
		return status, body, forwarded
	case <-time.After(time.Second):
		t.Fatal("proxy did not record the bytes it forwarded")
	}
	return 0, "", nil
}

func (p *firstHostProxy) cached(host string) (string, bool) {
	if host == "" {
		return "", false
	}
	m, _ := p.cache.Load().(map[string]string)
	body, ok := m[host]
	return body, ok
}

func (p *firstHostProxy) serve(client net.Conn, origin string) {
	defer client.Close()
	_ = client.SetDeadline(time.Now().Add(2 * time.Second))

	raw, err := readRawRequest(bufio.NewReader(client))
	if err != nil {
		return
	}
	p.forwarded <- append([]byte(nil), raw...)

	upstream, err := net.DialTimeout("tcp", origin, time.Second)
	if err != nil {
		return
	}
	defer upstream.Close()
	_ = upstream.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := upstream.Write(raw); err != nil {
		return
	}

	resp, err := http.ReadResponse(bufio.NewReader(upstream), &http.Request{Method: http.MethodGet})
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if host := firstHostValue(string(raw)); host != "" {
			prev, _ := p.cache.Load().(map[string]string)
			next := make(map[string]string, len(prev)+1)
			for k, v := range prev {
				next[k] = v
			}
			next[host] = string(body)
			p.cache.Store(next)
		}
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Connection", "close")
	_ = resp.Write(client)
}

func readRawRequest(r *bufio.Reader) ([]byte, error) {
	var buf bytes.Buffer
	for {
		line, err := r.ReadSlice('\n')
		if err != nil {
			return nil, err
		}
		buf.Write(line)
		if bytes.Equal(line, []byte("\r\n")) || bytes.Equal(line, []byte("\n")) {
			break
		}
	}
	n := contentLength(buf.Bytes())
	if n > 0 {
		body := make([]byte, n)
		if _, err := io.ReadFull(r, body); err != nil {
			return nil, err
		}
		buf.Write(body)
	}
	return buf.Bytes(), nil
}

func contentLength(header []byte) int {
	for _, line := range strings.Split(string(header), "\r\n") {
		name, value, ok := strings.Cut(line, ":")
		if ok && strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			var n int
			_, _ = fmt.Sscan(strings.TrimSpace(value), &n)
			return n
		}
	}
	return 0
}

func dialRoundTrip(addr, raw string) (int, string, error) {
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		return 0, "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.WriteString(conn, raw); err != nil {
		return 0, "", err
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: http.MethodGet})
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), err
}
