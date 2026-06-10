package sign

import (
	"bytes"
	"io"
	"net/http"
)

func Middleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if err := verifyRequest(w, r, key); err != nil {
				return
			}

			buf := &responseBuf{code: http.StatusOK}
			next.ServeHTTP(buf, r)

			for k, vals := range buf.Header() {
				w.Header()[k] = vals
			}
			if len(buf.body) > 0 {
				w.Header().Set(HeaderHashSHA256, ComputeHMAC(buf.body, key))
			}
			w.WriteHeader(buf.code)
			_, _ = w.Write(buf.body)
		})
	}
}

func verifyRequest(w http.ResponseWriter, r *http.Request, key string) error {
	if r.Body == nil {
		return nil
	}
	if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
		return nil
	}

	body, err := io.ReadAll(r.Body)
	_ = r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	if h := r.Header.Get(HeaderHashSHA256); h != "" && !EqualHMAC(h, body, key) {
		w.WriteHeader(http.StatusBadRequest)
		return bytes.ErrTooLarge
	}

	r.Body = io.NopCloser(bytes.NewReader(body))
	return nil
}

type responseBuf struct {
	code    int
	headers http.Header
	body    []byte
}

func (b *responseBuf) Header() http.Header {
	if b.headers == nil {
		b.headers = make(http.Header)
	}
	return b.headers
}

func (b *responseBuf) WriteHeader(code int) { b.code = code }

func (b *responseBuf) Write(p []byte) (int, error) {
	b.body = append(b.body, p...)
	return len(p), nil
}
