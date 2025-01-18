package keythrottle

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
)

var currentReqId uint64 = 0

type ctxKeyRequestIDInt int

const RequestIdIntKey ctxKeyRequestIDInt = 0

func RequestIdInteger(next http.Handler) http.Handler {
	fh := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		myId := atomic.AddUint64(&currentReqId, 1)
		ctx = context.WithValue(ctx, RequestIdIntKey, myId)
		next.ServeHTTP(w, r.WithContext(ctx))

	}
	return http.HandlerFunc(fh)
}

func GetReqIdInteger(ctx context.Context) (uint64, error) {
	if ctx == nil {
		return 0, errors.New("ctx is nil")
	}
	if reqId, ok := ctx.Value(RequestIdIntKey).(uint64); ok {
		return reqId, nil
	}
	return 0, errors.New("not found")
}
