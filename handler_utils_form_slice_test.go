package kit_test

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/arvo-health/kit"
	"github.com/gofiber/fiber/v2"
)

// maxAcceptedFormSliceIndex is the highest slice index a form or multipart
// body may name. A later Fiber release rejects anything above this before
// allocating the slice. Refusing those content types entirely also satisfies
// the tests below.
const maxAcceptedFormSliceIndex = 1000

// maxAllocBytes is the most heap ParseRequestBody may use while refusing an
// oversized index. It is far below the slice that index would allocate.
const maxAllocBytes = 256 * 1024

type namedChild struct {
	Name string `json:"name"`
}

type namedParent struct {
	Items []namedChild `json:"items"`
}

type formChild struct {
	Name string `form:"name"`
}

type formParent struct {
	Items []formChild `form:"items"`
}

type treeNode struct {
	Name     string     `json:"name"`
	Children []treeNode `json:"children"`
}

type treeParent struct {
	Items []treeNode `json:"items"`
}

type pointerParent struct {
	Items []*namedChild `json:"items"`
}

type indirectParent struct {
	Items *[]namedChild `json:"items"`
}

type blobChild struct {
	Blob [1024]byte `json:"blob"`
}

type blobParent struct {
	Items []blobChild `json:"items"`
}

func TestParseRequestBodyRejectsUnboundedFormSliceIndex(t *testing.T) {
	v := kit.NewValidator()
	over := maxAcceptedFormSliceIndex + 1

	t.Run("urlencoded field name just over the limit", func(t *testing.T) {
		var out namedParent
		err, n, alloc := parseForm(t, v, "application/x-www-form-urlencoded",
			[]byte(fmt.Sprintf("Items.%d.Name=x", over)), &out, func() int { return len(out.Items) })
		assertNoOversizedSlice(t, err, n, alloc)
	})

	t.Run("urlencoded field name allocates from the index", func(t *testing.T) {
		const index = 100_000
		var out namedParent
		err, n, alloc := parseForm(t, v, "application/x-www-form-urlencoded",
			[]byte(fmt.Sprintf("Items.%d.Name=x", index)), &out, func() int { return len(out.Items) })
		assertNoOversizedSlice(t, err, n, alloc)
	})

	t.Run("urlencoded charset parameter", func(t *testing.T) {
		const index = 100_000
		var out namedParent
		err, n, alloc := parseForm(t, v, "application/x-www-form-urlencoded; charset=UTF-8",
			[]byte(fmt.Sprintf("Items.%d.Name=x", index)), &out, func() int { return len(out.Items) })
		assertNoOversizedSlice(t, err, n, alloc)
	})

	t.Run("urlencoded bracket notation", func(t *testing.T) {
		const index = 100_000
		var out namedParent
		body := []byte(fmt.Sprintf("Items[%d][Name]=x", index))
		err, n, alloc := parseForm(t, v, "application/x-www-form-urlencoded", body, &out, func() int { return len(out.Items) })
		assertNoOversizedSlice(t, err, n, alloc)
	})

	t.Run("urlencoded form tag", func(t *testing.T) {
		const index = 100_000
		var out formParent
		err, n, alloc := parseForm(t, v, "application/x-www-form-urlencoded",
			[]byte(fmt.Sprintf("items.%d.name=x", index)), &out, func() int { return len(out.Items) })
		assertNoOversizedSlice(t, err, n, alloc)
	})

	t.Run("multipart field name", func(t *testing.T) {
		const index = 100_000
		var out namedParent
		ctype, body := multipartField(t, fmt.Sprintf("Items.%d.Name", index), "x")
		err, n, alloc := parseForm(t, v, ctype, body, &out, func() int { return len(out.Items) })
		assertNoOversizedSlice(t, err, n, alloc)
	})

	t.Run("multipart bracket notation", func(t *testing.T) {
		const index = 100_000
		var out namedParent
		ctype, body := multipartField(t, fmt.Sprintf("Items[%d][Name]", index), "x")
		err, n, alloc := parseForm(t, v, ctype, body, &out, func() int { return len(out.Items) })
		assertNoOversizedSlice(t, err, n, alloc)
	})

	t.Run("nested slice index", func(t *testing.T) {
		const index = 100_000
		var out treeParent
		err, n, alloc := parseForm(t, v, "application/x-www-form-urlencoded",
			[]byte(fmt.Sprintf("Items.0.Children.%d.Name=x", index)), &out, func() int {
				if len(out.Items) == 0 {
					return 0
				}
				return len(out.Items[0].Children)
			})
		assertNoOversizedSlice(t, err, n, alloc)
	})

	t.Run("slice of pointers", func(t *testing.T) {
		const index = 200_000
		var out pointerParent
		err, n, alloc := parseForm(t, v, "application/x-www-form-urlencoded",
			[]byte(fmt.Sprintf("Items.%d.Name=x", index)), &out, func() int { return len(out.Items) })
		assertNoOversizedSlice(t, err, n, alloc)
	})

	t.Run("pointer to slice", func(t *testing.T) {
		const index = 100_000
		var out indirectParent
		err, n, alloc := parseForm(t, v, "application/x-www-form-urlencoded",
			[]byte(fmt.Sprintf("Items.%d.Name=x", index)), &out, func() int {
				if out.Items == nil {
					return 0
				}
				return len(*out.Items)
			})
		assertNoOversizedSlice(t, err, n, alloc)
	})

	// The element field cannot be decoded, but the slice is still allocated
	// from the index before that error is returned.
	t.Run("allocation before element conversion error", func(t *testing.T) {
		const index = 2_000
		var out blobParent
		err, n, alloc := parseForm(t, v, "application/x-www-form-urlencoded",
			[]byte(fmt.Sprintf("Items.%d.Blob=x", index)), &out, func() int { return len(out.Items) })
		assertNoOversizedSlice(t, err, n, alloc)
	})
}

func assertNoOversizedSlice(t *testing.T, err error, length int, alloc uint64) {
	t.Helper()
	if err == nil {
		t.Fatalf("ParseRequestBody accepted an oversized form slice index (length %d, allocated %d bytes)", length, alloc)
	}
	if length != 0 {
		t.Fatalf("slice length %d was allocated before the request was refused (err=%v, alloc=%d)", length, err, alloc)
	}
	if alloc > maxAllocBytes {
		t.Fatalf("allocated %d bytes before refusing the request (err=%v)", alloc, err)
	}
}

func parseForm(t *testing.T, v kit.Validator, contentType string, body []byte, dst any, length func() int) (error, int, uint64) {
	t.Helper()
	if len(body) > 1024 {
		t.Fatalf("request body is %d bytes; the case must stay a small request", len(body))
	}

	app := fiber.New()
	var (
		called bool
		err    error
		n      int
		alloc  uint64
	)
	app.Post("/", func(c *fiber.Ctx) error {
		called = true
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		err = kit.ParseRequestBody(dst, c, v)
		runtime.ReadMemStats(&after)
		alloc = after.TotalAlloc - before.TotalAlloc
		n = length()
		return err
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set(fiber.HeaderContentType, contentType)
	resp, reqErr := app.Test(req, 10_000)
	if reqErr != nil {
		t.Fatalf("request was not served: %v", reqErr)
	}
	if resp.StatusCode == http.StatusRequestEntityTooLarge {
		t.Fatal("the body limit rejected the small request before parsing")
	}
	if !called {
		t.Fatal("handler was not called")
	}
	return err, n, alloc
}

func multipartField(t *testing.T, key, value string) (string, []byte) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField(key, value); err != nil {
		t.Fatalf("write multipart field: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart body: %v", err)
	}
	return w.FormDataContentType(), buf.Bytes()
}
