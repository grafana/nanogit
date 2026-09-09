package client

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/metrics"
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
