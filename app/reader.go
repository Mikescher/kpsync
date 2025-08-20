package app

import (
	"io"
	"sync"
)

type teeReadCloser struct {
	r io.Reader
	c io.Closer
}

func (t *teeReadCloser) Read(p []byte) (int, error) { return t.r.Read(p) }
func (t *teeReadCloser) Close() error               { return t.c.Close() }

type progressWriter struct {
	sync.Mutex

	done  int64
	total int64
	cb    func(done, total int64)
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)

	if pw.cb != nil {
		pw.Lock()
		defer pw.Unlock()
		pw.done += int64(n)
		pw.cb(pw.done, pw.total)
	}

	return n, nil
}

func ReadAllWithProgress(r io.Reader, totalBytes int64, onProgress func(done, total int64)) ([]byte, error) {
	return io.ReadAll(NewProgressReader(r, totalBytes, onProgress))
}

func NewProgressReader(r io.Reader, totalBytes int64, onProgress func(done, total int64)) io.ReadCloser {
	pw := &progressWriter{total: totalBytes, cb: onProgress}

	return &teeReadCloser{r: io.TeeReader(r, pw), c: io.NopCloser(r)}
}
