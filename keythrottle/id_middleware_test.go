package keythrottle

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIdInteger(t *testing.T) {
	tests := map[string]struct {
		startReqId        uint64
		request           func() *http.Request
		expectedRequestId uint64
		expectError       bool
	}{
		"expected response": {
			startReqId: 100,
			request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				req.Header.Add("X-Request-Id", "req-123456")

				return req
			},
			expectedRequestId: 101,
			expectError:       false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			currentReqId = test.startReqId
			w := httptest.NewRecorder()
			r := chi.NewRouter()

			r.Use(RequestIdInteger)
			gotError := false
			var resultRequestId uint64 = 0
			r.Get("/", func(writer http.ResponseWriter, r *http.Request) {
				var err error
				resultRequestId, err = GetReqIdInteger(r.Context())
				if err != nil {
					gotError = true
				}
				w.Write([]byte("testing"))
			})
			r.ServeHTTP(w, test.request())

			if test.expectError {
				if !gotError {
					t.Errorf("got no error")
				}
			}
			if !test.expectError {
				if gotError {
					t.Errorf("got an error")
				}
				if resultRequestId != test.expectedRequestId {
					t.Errorf("got %d, want %d", resultRequestId, test.expectedRequestId)
				}
			}
		})

	}
}
