package kit_test

import (
	"bufio"
	"bytes"
	"compress/gzip"
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

// connectionReadBuffer is the read buffer of the Fiber server these tests
// start. Fiber's default (DefaultReadBufferSize) and fasthttp's default
// (defaultReadBufferSize) are both 4096, and the tests set the same value
// explicitly so the epilogue is placed on the first byte the parser does not
// consume.
const connectionReadBuffer = 4096

// streamPrefetch is the most fasthttp reads of a streamed body before the
// handler runs when multipart pre-parsing is disabled (readBodyWithStreaming
// caps the prefetch at 8 KiB). Bytes after that stay on the connection until
// something reads them.
const streamPrefetch = 8 * 1024

const (
	smuggledBody = "smuggled-request-executed"
	nextBody     = "next-owner"
	boundary     = "kitbound"
)

// TestMultipartEpilogueIsNotASecondRequest ensures a keep-alive multipart
// request cannot carry a second HTTP request after the closing boundary.
// fasthttp 1.62.0, which this module requires, stops at that boundary while
// Content-Length still covers the tail, and the server then parses the tail
// as the next request on the connection.
func TestMultipartEpilogueIsNotASecondRequest(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		assertEpilogueNotExecuted(t, serverOptions{}, inlineBody)
	})
	t.Run("stream", func(t *testing.T) {
		assertEpilogueNotExecuted(t, serverOptions{stream: true}, inlineBody)
	})
	t.Run("expect-continue", func(t *testing.T) {
		assertEpilogueNotExecuted(t, serverOptions{}, afterContinue)
	})
	t.Run("expect-continue-stream", func(t *testing.T) {
		assertEpilogueNotExecuted(t, serverOptions{stream: true}, afterContinue)
	})
	// Pre-parsing is off, so the tail is left by the handler parse that
	// kit.ParseRequestBody performs on a streamed body, not by the server's
	// pre-parser. Logger middleware is omitted here because its call to
	// c.Body() copies the rest of that stream and would hide the unread tail.
	t.Run("stream-without-preparse", func(t *testing.T) {
		assertEpilogueNotExecuted(t, serverOptions{stream: true, disablePreParse: true, skipLogger: true}, streamPrefetchBody)
	})
	// Content-Encoding skips the server pre-parser. With a streamed body,
	// kit.ParseRequestBody reaches MultipartForm, which gunzips and stops at
	// the closing boundary. Logger middleware is omitted for the same reason
	// as stream-without-preparse.
	t.Run("gzip-stream", func(t *testing.T) {
		// A parser that rejects the trailing bytes still has to keep them off
		// the next request. The outer call may therefore be a 400.
		assertEpilogueNotExecuted(t, serverOptions{stream: true, skipLogger: true}, gzipStreamBody)
	})
}

type serverOptions struct {
	stream          bool
	disablePreParse bool
	skipLogger      bool
}

type bodyPlacement int

const (
	// inlineBody sends the body in the same write as the headers. The first
	// connection read is connectionReadBuffer bytes, and that read includes
	// the headers, so the unread tail begins at body offset
	// connectionReadBuffer-len(header).
	inlineBody bodyPlacement = iota
	// afterContinue sends the body only once the server has asked for it.
	// The connection buffer is empty, so the first body read consumes
	// connectionReadBuffer body bytes.
	afterContinue
	// streamPrefetchBody is the unread offset when the server prefetches a
	// streamed body and the handler's multipart parser stops at the boundary.
	streamPrefetchBody
	// gzipStreamBody is that same prefetch, with the form carried in a gzip
	// member. Pre-parsing does not run when Content-Encoding is set.
	gzipStreamBody
)

func assertEpilogueNotExecuted(t *testing.T, opts serverOptions, placement bodyPlacement) {
	t.Helper()
	srv := startKitServer(t, opts)

	secretBefore := atomic.LoadInt32(&srv.secretHits)
	raw, continueAt := buildSmuggle(t, placement)
	conn := dial(t, srv.addr)
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))

	reader := bufio.NewReader(conn)
	if placement == afterContinue {
		if _, err := io.WriteString(conn, raw[:continueAt]); err != nil {
			t.Fatalf("write headers: %v", err)
		}
		resp := readResponse(t, reader)
		if resp.StatusCode != http.StatusContinue {
			t.Fatalf("status = %d, want 100 Continue before the body is sent", resp.StatusCode)
		}
		raw = raw[continueAt:]
	}
	if _, err := io.WriteString(conn, raw); err != nil {
		t.Fatalf("write request: %v", err)
	}

	outer := readResponse(t, reader)
	next, nextErr := readResponseOptional(reader)
	if atomic.LoadInt32(&srv.secretHits) != secretBefore || outer.Body == smuggledBody || (nextErr == nil && next.Body == smuggledBody) {
		t.Fatalf("hidden request was executed as its own call: secret hits %d, outer %d %q, next %d %q (%v)",
			atomic.LoadInt32(&srv.secretHits)-secretBefore, outer.StatusCode, outer.Body, next.StatusCode, next.Body, nextErr)
	}
	if placement == gzipStreamBody && outer.StatusCode != http.StatusOK {
		// Rejecting the body is safe when the hidden request was not executed
		// and the following request was not answered with the hidden response.
		return
	}
	if outer.StatusCode != http.StatusOK || outer.Body != "outer:Ada" {
		t.Fatalf("outer response = %d %q, want 200 outer:Ada", outer.StatusCode, outer.Body)
	}
	if nextErr != nil || next.StatusCode != http.StatusOK || next.Body != nextBody {
		t.Fatalf("following request response = %d %q (%v), want 200 %s", next.StatusCode, next.Body, nextErr, nextBody)
	}
}

type recordedResponse struct {
	StatusCode int
	Body       string
}

func readResponseOptional(reader *bufio.Reader) (recordedResponse, error) {
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		return recordedResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return recordedResponse{StatusCode: resp.StatusCode}, err
	}
	return recordedResponse{StatusCode: resp.StatusCode, Body: string(body)}, nil
}

func readResponse(t *testing.T, reader *bufio.Reader) recordedResponse {
	t.Helper()
	resp, err := readResponseOptional(reader)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return resp
}

// buildSmuggle returns the bytes to write and, for afterContinue, the index
// at which the body begins. The hidden request is the first unread body byte.
// A real follow-up request is appended outside Content-Length so a fix that
// drains the declared body still answers that request.
func paddedForm(t *testing.T, prefix string, size int) string {
	t.Helper()
	if len(prefix) > size {
		t.Fatalf("prefix length %d exceeds %d", len(prefix), size)
	}
	return prefix + strings.Repeat("a", size-len(prefix))
}

func buildSmuggle(t *testing.T, placement bodyPlacement) (raw string, continueAt int) {
	t.Helper()
	form := "--" + boundary + "\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nAda\r\n--" + boundary + "--\r\n"
	hidden := "GET /secret HTTP/1.1\r\nHost: localhost\r\n\r\n"
	followUp := "GET /next HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"

	switch placement {
	case afterContinue:
		body := form + strings.Repeat("a", connectionReadBuffer-len(form)) + hidden
		header := fmt.Sprintf("POST /upload HTTP/1.1\r\nHost: localhost\r\nContent-Type: multipart/form-data; boundary=%s\r\nContent-Length: %d\r\nExpect: 100-continue\r\nConnection: keep-alive\r\n\r\n", boundary, len(body))
		return header + body + followUp, len(header)
	case streamPrefetchBody:
		body := paddedForm(t, form, streamPrefetch) + hidden
		header := fmt.Sprintf("POST /upload HTTP/1.1\r\nHost: localhost\r\nContent-Type: multipart/form-data; boundary=%s\r\nContent-Length: %d\r\nConnection: keep-alive\r\n\r\n", boundary, len(body))
		return header + body + followUp, 0
	case gzipStreamBody:
		var gz bytes.Buffer
		zw := gzip.NewWriter(&gz)
		if _, err := zw.Write([]byte(form)); err != nil {
			t.Fatalf("gzip form: %v", err)
		}
		if err := zw.Close(); err != nil {
			t.Fatalf("close gzip: %v", err)
		}
		body := paddedForm(t, gz.String(), streamPrefetch) + hidden
		header := fmt.Sprintf("POST /upload HTTP/1.1\r\nHost: localhost\r\nContent-Type: multipart/form-data; boundary=%s\r\nContent-Encoding: gzip\r\nContent-Length: %d\r\nConnection: keep-alive\r\n\r\n", boundary, len(body))
		return header + body + followUp, 0
	case inlineBody:
		var header, body string
		cl := connectionReadBuffer
		for i := 0; i < 6; i++ {
			header = fmt.Sprintf("POST /upload HTTP/1.1\r\nHost: localhost\r\nContent-Type: multipart/form-data; boundary=%s\r\nContent-Length: %d\r\nConnection: keep-alive\r\n\r\n", boundary, cl)
			consumed := connectionReadBuffer - len(header)
			if consumed <= len(form) {
				t.Fatalf("first read (%d) does not leave room for the form after %d header bytes", connectionReadBuffer, len(header))
			}
			body = form + strings.Repeat("a", consumed-len(form)) + hidden
			if len(body) == cl {
				break
			}
			cl = len(body)
		}
		if len(header)+len(body)-len(hidden) != connectionReadBuffer {
			t.Fatalf("hidden request is not aligned to the first unread byte: header %d body-without-hidden %d", len(header), len(body)-len(hidden))
		}
		return header + body + followUp, 0
	default:
		t.Fatalf("unknown placement %d", placement)
		return "", 0
	}
}

type kitServer struct {
	addr       string
	secretHits int32
}

func startKitServer(t *testing.T, opts serverOptions) *kitServer {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := &kitServer{}
	app := fiber.New(fiber.Config{
		DisableStartupMessage:        true,
		ErrorHandler:                 kit.ErrorHandler(logger),
		ReadBufferSize:               connectionReadBuffer,
		StreamRequestBody:            opts.stream,
		DisablePreParseMultipartForm: opts.disablePreParse,
	})
	if !opts.skipLogger {
		app.Use(kit.LoggerMiddleware(logger))
	}
	validator := kit.NewValidator()
	app.Post("/upload", func(c *fiber.Ctx) error {
		var body struct {
			Name string `form:"name" json:"name" validate:"required"`
		}
		if err := kit.ParseRequestBody(&body, c, validator); err != nil {
			return err
		}
		return c.SendString("outer:" + body.Name)
	})
	app.Get("/secret", func(c *fiber.Ctx) error {
		atomic.AddInt32(&srv.secretHits, 1)
		return c.SendString(smuggledBody)
	})
	app.Get("/next", func(c *fiber.Ctx) error {
		return c.SendString(nextBody)
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		_ = app.Shutdown()
		_ = ln.Close()
	})
	go func() { _ = app.Listener(ln) }()
	srv.addr = ln.Addr().String()
	return srv
}

func dial(t *testing.T, addr string) net.Conn {
	t.Helper()
	var last error
	for i := 0; i < 50; i++ {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			return conn
		}
		last = err
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("dial %s: %v", addr, last)
	return nil
}
