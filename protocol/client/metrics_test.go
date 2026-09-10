package client

import (
	"bytes"
	"compress/zlib"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/metrics"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/retry"
)

// testRecorder is a simple metrics.Recorder implementation for testing.
type testRecorder struct {
	mu               sync.Mutex
	httpRequests     []metrics.HTTPRequestSample
	objectsFetched   []metrics.ObjectsFetchedSample
	cacheAccessCalls []metrics.CacheAccessSample
}

func (r *testRecorder) HTTPRequest(ctx context.Context, sample metrics.HTTPRequestSample) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.httpRequests = append(r.httpRequests, sample)
}

func (r *testRecorder) ObjectsFetched(ctx context.Context, sample metrics.ObjectsFetchedSample) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.objectsFetched = append(r.objectsFetched, sample)
}

func (r *testRecorder) CacheAccess(ctx context.Context, sample metrics.CacheAccessSample) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cacheAccessCalls = append(r.cacheAccessCalls, sample)
}

// testPackfileStorage is a simple storage.PackfileStorage implementation for testing.
type testPackfileStorage struct {
	getFunc func(hash.Hash) (*protocol.PackfileObject, bool)
}

func (s *testPackfileStorage) Get(key hash.Hash) (*protocol.PackfileObject, bool) {
	return s.getFunc(key)
}
func (s *testPackfileStorage) GetByType(key hash.Hash, objType protocol.ObjectType) (*protocol.PackfileObject, bool) {
	return s.getFunc(key)
}
func (s *testPackfileStorage) GetAllKeys() []hash.Hash              { return nil }
func (s *testPackfileStorage) Add(objs ...*protocol.PackfileObject) {}
func (s *testPackfileStorage) Delete(key hash.Hash)                 {}
func (s *testPackfileStorage) Len() int                             { return 0 }

func TestDo_RecordsHTTPRequestMetric(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(formatTestResponse(t, protocol.PackLine("version 2\n"))))
	}))
	t.Cleanup(server.Close)

	recorder := &testRecorder{}
	ctx := metrics.ToContext(context.Background(), recorder)

	client, err := NewRawClient(server.URL + "/repo")
	require.NoError(t, err)

	require.NoError(t, client.SmartInfo(ctx, "git-upload-pack"))

	require.Len(t, recorder.httpRequests, 1)
	sample := recorder.httpRequests[0]
	require.Equal(t, metrics.OperationSmartInfo, sample.Operation)
	require.Equal(t, http.StatusOK, sample.StatusCode)
	require.GreaterOrEqual(t, sample.Duration.Nanoseconds(), int64(0))
	require.Equal(t, 1, sample.Attempt)
}

func TestDo_RecordsHTTPRequestMetricPerRetryAttempt(t *testing.T) {
	t.Parallel()

	attemptCount := 0
	successResponse := formatTestResponse(t, protocol.PackLine("version 2\n"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(successResponse))
	}))
	t.Cleanup(server.Close)

	testRetrier := newTestRetrier(3)
	testRetrier.shouldRetryFunc = func(ctx context.Context, err error, attempt int) bool {
		return err != nil
	}

	recorder := &testRecorder{}
	ctx := metrics.ToContext(retry.ToContext(context.Background(), testRetrier), recorder)

	client, err := NewRawClient(server.URL + "/repo")
	require.NoError(t, err)

	require.NoError(t, client.SmartInfo(ctx, "git-upload-pack"))

	require.Len(t, recorder.httpRequests, 3)
	for i, sample := range recorder.httpRequests {
		require.Equal(t, metrics.OperationSmartInfo, sample.Operation)
		require.Equal(t, i+1, sample.Attempt)
		if i < 2 {
			require.Equal(t, http.StatusInternalServerError, sample.StatusCode)
		} else {
			require.Equal(t, http.StatusOK, sample.StatusCode)
		}
	}
}

func TestCheckCacheForObjects_RecordsCacheAccessMetric(t *testing.T) {
	t.Parallel()

	hit, err := hash.FromHex("0123456789abcdef0123456789abcdef01234567")
	require.NoError(t, err)
	miss, err := hash.FromHex("0123456789abcdef0123456789abcdef01234568")
	require.NoError(t, err)

	cachedObj := &protocol.PackfileObject{Hash: hit, Type: protocol.ObjectTypeBlob, Data: []byte("x")}

	storage := &testPackfileStorage{
		getFunc: func(h hash.Hash) (*protocol.PackfileObject, bool) {
			if h == hit {
				return cachedObj, true
			}
			return nil, false
		},
	}

	recorder := &testRecorder{}
	ctx := metrics.ToContext(context.Background(), recorder)

	client, err := NewRawClient("https://example.com/repo")
	require.NoError(t, err)

	objects := make(map[string]*protocol.PackfileObject)
	_, _ = client.checkCacheForObjects(ctx, FetchOptions{Want: []hash.Hash{hit, miss}}, objects, storage)

	require.Equal(t, []metrics.CacheAccessSample{{Hit: true}, {Hit: false}}, recorder.cacheAccessCalls)
}

func TestFetch_RecordsObjectsFetchedMetric(t *testing.T) {
	t.Parallel()

	blobData := []byte("abc")
	var zbuf bytes.Buffer
	zw := zlib.NewWriter(&zbuf)
	_, err := zw.Write(blobData)
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	var body bytes.Buffer
	writePkt := func(b []byte) {
		fmt.Fprintf(&body, "%04x", len(b)+4)
		body.Write(b)
	}
	writePkt([]byte("packfile\n"))

	pack := []byte("PACK" +
		"\x00\x00\x00\x02" + // version 2
		"\x00\x00\x00\x01") // 1 object
	objHeader := byte(3)<<4 | byte(len(blobData)&0xF) // blob type=3, size fits in one nibble
	pack = append(pack, objHeader)
	pack = append(pack, zbuf.Bytes()...)
	writePkt(append([]byte{1}, pack...))
	body.WriteString("0000")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write(body.Bytes()); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	recorder := &testRecorder{}
	ctx := metrics.ToContext(context.Background(), recorder)

	client, err := NewRawClient(server.URL + "/repo")
	require.NoError(t, err)

	want, err := hash.FromHex("0123456789abcdef0123456789abcdef01234567")
	require.NoError(t, err)

	objects, err := client.Fetch(ctx, FetchOptions{Want: []hash.Hash{want}, Done: true})
	require.NoError(t, err)
	require.Len(t, objects, 1)

	require.Len(t, recorder.objectsFetched, 1)
	require.Equal(t, 1, recorder.objectsFetched[0].Count)
	require.Greater(t, recorder.objectsFetched[0].Bytes, int64(0))
}

func TestFetch_RecordsObjectsFetchedOnMidStreamFailure(t *testing.T) {
	t.Parallel()

	// Same corrupt-packfile shape as TestFetch_CorruptPackfile in
	// fetch_test.go: a single object whose zlib stream is invalid,
	// causing processPackfileResponse to fail before any object is
	// successfully parsed. ObjectsFetched must still fire, reporting
	// zero objects but the bytes read before the failure.
	var body bytes.Buffer
	writePkt := func(b []byte) {
		fmt.Fprintf(&body, "%04x", len(b)+4)
		body.Write(b)
	}
	writePkt([]byte("packfile\n"))
	pack := []byte("PACK" +
		"\x00\x00\x00\x02" + // version 2
		"\x00\x00\x00\x01" + // 1 object
		"\x33" + // blob, size 3
		"\xff\xff") // invalid zlib stream
	writePkt(append([]byte{1}, pack...))
	body.WriteString("0000")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write(body.Bytes()); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	recorder := &testRecorder{}
	ctx := metrics.ToContext(context.Background(), recorder)

	client, err := NewRawClient(server.URL + "/repo")
	require.NoError(t, err)

	want, err := hash.FromHex("0123456789abcdef0123456789abcdef01234567")
	require.NoError(t, err)

	_, err = client.Fetch(ctx, FetchOptions{Want: []hash.Hash{want}, Done: true})
	require.Error(t, err)

	require.Len(t, recorder.objectsFetched, 1,
		"ObjectsFetched must fire even when the fetch fails mid-stream")
	require.Equal(t, 0, recorder.objectsFetched[0].Count,
		"the corrupt object never parsed, so no objects were collected")
	require.Greater(t, recorder.objectsFetched[0].Bytes, int64(0),
		"bytes read before the failure must still be reported")
}

func TestDo_RecordsStatusCodeWhenClientReturnsResponseAndError(t *testing.T) {
	t.Parallel()

	// http.Client.Do returns a non-nil Response alongside a non-nil
	// error specifically when CheckRedirect rejects a redirect. do()
	// must preserve that response's status code rather than reporting
	// the zero value, which is reserved for "no response was received
	// at all".
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/elsewhere", http.StatusFound)
	}))
	t.Cleanup(server.Close)

	stopRedirect := errors.New("stop redirect")
	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return stopRedirect
		},
	}

	recorder := &testRecorder{}
	ctx := metrics.ToContext(context.Background(), recorder)

	client, err := NewRawClient(server.URL+"/repo", options.WithHTTPClient(httpClient))
	require.NoError(t, err)

	err = client.SmartInfo(ctx, "git-upload-pack")
	require.Error(t, err)

	require.Len(t, recorder.httpRequests, 1)
	require.Equal(t, http.StatusFound, recorder.httpRequests[0].StatusCode)
}

// slowCloseBody is an io.ReadCloser whose Close blocks for a configured
// delay, so tests can tell whether code under test captured a timestamp
// before or after closing the body.
type slowCloseBody struct {
	io.Reader
	delay time.Duration
}

func (s *slowCloseBody) Close() error {
	time.Sleep(s.delay)
	return nil
}

// slowCloseTransport returns a fixed status code with a slow-closing body,
// bypassing the network entirely.
type slowCloseTransport struct {
	statusCode int
	closeDelay time.Duration
}

func (t *slowCloseTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: t.statusCode,
		Status:     fmt.Sprintf("%d", t.statusCode),
		Body:       &slowCloseBody{Reader: strings.NewReader(""), delay: t.closeDelay},
		Header:     make(http.Header),
		Request:    req,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}, nil
}

func TestDo_DurationExcludesBodyCloseOnRetryableStatus(t *testing.T) {
	t.Parallel()

	// A 5xx response takes the retryable "CheckServerUnavailable" branch
	// in do(), which closes res.Body before returning. Duration must be
	// snapshotted immediately after Do returns, not after that close, so
	// a slow Close (e.g. draining a large unread body) must not inflate
	// the reported Duration.
	const closeDelay = 200 * time.Millisecond
	httpClient := &http.Client{
		Transport: &slowCloseTransport{statusCode: http.StatusInternalServerError, closeDelay: closeDelay},
	}

	recorder := &testRecorder{}
	ctx := metrics.ToContext(context.Background(), recorder)

	client, err := NewRawClient("https://example.com/repo", options.WithHTTPClient(httpClient))
	require.NoError(t, err)

	err = client.SmartInfo(ctx, "git-upload-pack")
	require.Error(t, err)

	require.Len(t, recorder.httpRequests, 1)
	require.Less(t, recorder.httpRequests[0].Duration, closeDelay,
		"Duration must be captured before the slow Body.Close(), not after")
}

func TestDo_DurationIncludesBodyReadOnSuccess(t *testing.T) {
	t.Parallel()

	// A 2xx response is handed to the caller to consume, so its Duration
	// must be measured when the caller closes the body — after the body
	// (the packfile, for upload-pack) has been read — not snapshotted at
	// the response headers. A slow-closing body stands in for a slow body
	// read: if Duration were captured at headers it would be near zero,
	// but it must be at least the close delay.
	const closeDelay = 150 * time.Millisecond
	httpClient := &http.Client{
		Transport: &slowCloseTransport{statusCode: http.StatusOK, closeDelay: closeDelay},
	}

	recorder := &testRecorder{}
	ctx := metrics.ToContext(context.Background(), recorder)

	client, err := NewRawClient("https://example.com/repo", options.WithHTTPClient(httpClient))
	require.NoError(t, err)

	// SmartInfo returns 2xx here and closes the body on the way out.
	require.NoError(t, client.SmartInfo(ctx, "git-upload-pack"))

	require.Len(t, recorder.httpRequests, 1)
	require.GreaterOrEqual(t, recorder.httpRequests[0].Duration, closeDelay,
		"Duration must include reading/closing the response body, not stop at headers")
}

func TestFetch_RecordsObjectsFetchedCountOnDeltaResolutionFailure(t *testing.T) {
	t.Parallel()

	// A ref-delta object whose base is never sent in this response.
	// ReadObject succeeds (the delta itself is well-formed), but
	// resolveDeltas fails afterwards because the base can't be found —
	// this must not erase the fact that one object was successfully
	// read off the wire, even though it's absent from the returned
	// objects map.
	//
	// Delta payload format (uncompressed, before zlib): source-size
	// varint (0x01), target-size varint (0x01), then one "insert
	// literal" instruction: a cmd byte equal to the literal length
	// (0x01) followed by that many literal bytes ('A').
	deltaPayload := []byte{0x01, 0x01, 0x01, 'A'}
	var zbuf bytes.Buffer
	zw := zlib.NewWriter(&zbuf)
	_, err := zw.Write(deltaPayload)
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	missingBase, err := hash.FromHex(strings.Repeat("0", 39) + "a")
	require.NoError(t, err)

	var body bytes.Buffer
	writePkt := func(b []byte) {
		fmt.Fprintf(&body, "%04x", len(b)+4)
		body.Write(b)
	}
	writePkt([]byte("packfile\n"))

	pack := []byte("PACK" +
		"\x00\x00\x00\x02" + // version 2
		"\x00\x00\x00\x01") // 1 object
	objHeader := byte(protocol.ObjectTypeRefDelta)<<4 | byte(len(deltaPayload)&0xF) // ref-delta, size fits in one nibble
	pack = append(pack, objHeader)
	pack = append(pack, missingBase[:]...) // 20-byte raw parent hash
	pack = append(pack, zbuf.Bytes()...)
	writePkt(append([]byte{1}, pack...))
	body.WriteString("0000")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write(body.Bytes()); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	recorder := &testRecorder{}
	ctx := metrics.ToContext(context.Background(), recorder)

	client, err := NewRawClient(server.URL + "/repo")
	require.NoError(t, err)

	want, err := hash.FromHex("0123456789abcdef0123456789abcdef01234567")
	require.NoError(t, err)

	_, err = client.Fetch(ctx, FetchOptions{Want: []hash.Hash{want}, Done: true})
	require.ErrorContains(t, err, "missing base objects")

	require.Len(t, recorder.objectsFetched, 1)
	require.Equal(t, 1, recorder.objectsFetched[0].Count,
		"the delta object was successfully read off the wire even though its base was never resolved")
}
