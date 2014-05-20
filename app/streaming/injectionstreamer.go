package streaming

import (
	"github.com/alexandre-normand/glukit/app/container"
	"github.com/alexandre-normand/glukit/app/glukitio"
	"github.com/alexandre-normand/glukit/app/model"
	"time"
)

type InjectionStreamer struct {
	head    *container.ImmutableList
	tailVal *model.Injection
	wr      glukitio.InjectionBatchWriter
	d       time.Duration
}

// NewInjectionStreamerDuration returns a new InjectionStreamer whose buffer has the specified size.
func NewInjectionStreamerDuration(wr glukitio.InjectionBatchWriter, bufferDuration time.Duration) *InjectionStreamer {
	return newInjectionStreamerDuration(nil, nil, wr, bufferDuration)
}

func newInjectionStreamerDuration(head *container.ImmutableList, tailVal *model.Injection, wr glukitio.InjectionBatchWriter, bufferDuration time.Duration) *InjectionStreamer {
	w := new(InjectionStreamer)
	w.head = head
	w.tailVal = tailVal
	w.wr = wr
	w.d = bufferDuration

	return w
}

// WriteInjection writes a single Injection into the buffer.
func (b *InjectionStreamer) WriteInjection(c model.Injection) (s *InjectionStreamer, err error) {
	return b.WriteInjections([]model.Injection{c})
}

// WriteInjections writes the contents of p into the buffer.
// It returns the number of bytes written.
// If nn < len(p), it also returns an error explaining
// why the write is short. p must be sorted by time (oldest to most recent).
func (b *InjectionStreamer) WriteInjections(p []model.Injection) (s *InjectionStreamer, err error) {
	s = newInjectionStreamerDuration(b.head, b.tailVal, b.wr, b.d)
	if err != nil {
		return s, err
	}

	for _, c := range p {
		t := c.GetTime()

		if s.head == nil {
			s = newInjectionStreamerDuration(container.NewImmutableList(nil, c), &c, s.wr, s.d)
		} else if t.Sub(s.tailVal.GetTime()) >= s.d {
			s, err = s.Flush()
			if err != nil {
				return s, err
			}
			s = newInjectionStreamerDuration(container.NewImmutableList(nil, c), &c, s.wr, s.d)
		} else {
			s = newInjectionStreamerDuration(container.NewImmutableList(s.head, c), s.tailVal, s.wr, s.d)
		}
	}

	return s, err
}

// Flush writes any buffered data to the underlying glukitio.Writer as a batch.
func (b *InjectionStreamer) Flush() (s *InjectionStreamer, err error) {
	r, size := b.head.ReverseList()
	batch := ListToArrayOfInjectionReads(r, size)

	if len(batch) > 0 {
		innerWriter, err := b.wr.WriteInjectionBatch(batch)
		if err != nil {
			return nil, err
		} else {
			return newInjectionStreamerDuration(nil, nil, innerWriter, b.d), nil
		}
	}

	return newInjectionStreamerDuration(nil, nil, b.wr, b.d), nil
}

func ListToArrayOfInjectionReads(head *container.ImmutableList, size int) []model.Injection {
	r := make([]model.Injection, size)
	cursor := head
	for i := 0; i < size; i++ {
		r[i] = cursor.Value().(model.Injection)
		cursor = cursor.Next()
	}

	return r
}

// Close flushes the buffer and the inner writer to effectively ensure nothing is left
// unwritten
func (b *InjectionStreamer) Close() (s *InjectionStreamer, err error) {
	g, err := b.Flush()
	if err != nil {
		return g, err
	}

	innerWriter, err := g.wr.Flush()
	if err != nil {
		return newInjectionStreamerDuration(g.head, g.tailVal, innerWriter, b.d), err
	}

	return g, nil
}
