package bufio

import (
	"github.com/alexandre-normand/glukit/app/apimodel"
	"github.com/alexandre-normand/glukit/app/container"
	"github.com/alexandre-normand/glukit/app/glukitio"
)

type BufferedGlucoseReadBatchWriter struct {
	head      *container.ImmutableList
	size      int
	flushSize int
	wr        glukitio.GlucoseReadBatchWriter
}

// NewGlucoseReadWriterSize returns a new Writer whose Buffer has the specified size.
func NewGlucoseReadWriterSize(wr glukitio.GlucoseReadBatchWriter, flushSize int) *BufferedGlucoseReadBatchWriter {
	return newGlucoseReadWriterSize(wr, nil, 0, flushSize)
}

func newGlucoseReadWriterSize(wr glukitio.GlucoseReadBatchWriter, head *container.ImmutableList, size int, flushSize int) *BufferedGlucoseReadBatchWriter {
	// Is it already a Writer?
	b, ok := wr.(*BufferedGlucoseReadBatchWriter)
	if ok && b.flushSize >= flushSize {
		return b
	}

	w := new(BufferedGlucoseReadBatchWriter)
	w.size = size
	w.flushSize = flushSize
	w.wr = wr
	w.head = head

	return w
}

// WriteGlucose writes a single apimodel.DayOfGlucoseReads
func (b *BufferedGlucoseReadBatchWriter) WriteGlucoseReadBatch(p []apimodel.GlucoseRead) (glukitio.GlucoseReadBatchWriter, error) {
	return b.WriteGlucoseReadBatches([]apimodel.DayOfGlucoseReads{apimodel.NewDayOfGlucoseReads(p)})
}

// WriteGlucoseReadBatches writes the contents of p into the Buffer.
// It returns the number of batches written.
// If nn < len(p), it also returns an error explaining
// why the write is short.
func (b *BufferedGlucoseReadBatchWriter) WriteGlucoseReadBatches(p []apimodel.DayOfGlucoseReads) (glukitio.GlucoseReadBatchWriter, error) {
	w := b
	for i := range p {
		batch := p[i]
		if w.size >= w.flushSize {
			fw, err := w.Flush()
			if err != nil {
				return fw, err
			}
			w = fw.(*BufferedGlucoseReadBatchWriter)
		}

		w = newGlucoseReadWriterSize(w.wr, container.NewImmutableList(w.head, batch), w.size+1, w.flushSize)
	}

	return w, nil
}

// Flush writes any Buffered data to the underlying glukitio.Writer.
func (b *BufferedGlucoseReadBatchWriter) Flush() (w glukitio.GlucoseReadBatchWriter, err error) {
	if b.size == 0 {
		return newGlucoseReadWriterSize(b.wr, nil, 0, b.flushSize), nil
	}
	r, size := b.head.ReverseList()
	batch := ListToArrayOfGlucoseReadBatch(r, size)

	if len(batch) > 0 {
		innerWriter, err := b.wr.WriteGlucoseReadBatches(batch)
		if err != nil {
			return nil, err
		}

		return newGlucoseReadWriterSize(innerWriter, nil, 0, b.flushSize), nil
	}

	return newGlucoseReadWriterSize(b.wr, nil, 0, b.flushSize), nil
}

func ListToArrayOfGlucoseReadBatch(head *container.ImmutableList, size int) []apimodel.DayOfGlucoseReads {
	r := make([]apimodel.DayOfGlucoseReads, size)
	cursor := head
	for i := 0; i < size; i++ {
		r[i] = cursor.Value().(apimodel.DayOfGlucoseReads)
		cursor = cursor.Next()
	}

	return r
}
