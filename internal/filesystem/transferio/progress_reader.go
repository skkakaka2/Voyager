package transferio

import (
	"context"
	"io"
)

func NewProgressReader(ctx context.Context, reader io.Reader, progress func(bytesDone int64)) io.Reader {
	return &progressReader{ctx: ctx, reader: reader, progress: progress}
}

type progressReader struct {
	ctx      context.Context
	reader   io.Reader
	progress func(bytesDone int64)
	done     int64
}

func (r *progressReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(p)
	if n > 0 {
		r.done += int64(n)
		if r.progress != nil {
			r.progress(r.done)
		}
	}
	if err == nil {
		if ctxErr := r.ctx.Err(); ctxErr != nil {
			return n, ctxErr
		}
	}
	return n, err
}
