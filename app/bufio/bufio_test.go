package bufio_test

import (
	. "github.com/alexandre-normand/glukit/app/bufio"
	"github.com/alexandre-normand/glukit/app/model"
	"log"
	"testing"
)

type statsCalibrationWriter struct {
	total      int
	batchCount int
}

func (w *statsCalibrationWriter) WriteCalibrations(p []model.CalibrationRead) (n int, err error) {
	log.Printf("Got some calibrations")
	w.total += len(p)
	w.batchCount++

	return len(p), nil
}

func TestSimpleWriteOfSingleBatch(t *testing.T) {
	statsWriter := new(statsCalibrationWriter)
	w := NewWriterSize(statsWriter, 10)
	calibrations := make([]model.CalibrationRead, 10)
	for i := 0; i < 10; i++ {
		calibrations[i] = model.CalibrationRead{model.Timestamp{"", 0}, 75}
	}
	w.WriteCalibrations(calibrations)
	w.Flush()

	if statsWriter.total != 10 {
		t.Errorf("simple batch test failed: got a total of %d but expected %d", statsWriter.total, 10)
	}

	if statsWriter.batchCount != 1 {
		t.Errorf("simple batch test failed: got a total of %d but expected %d", statsWriter.batchCount, 1)
	}
}

func TestIndividualWrite(t *testing.T) {
	statsWriter := new(statsCalibrationWriter)
	w := NewWriterSize(statsWriter, 10)
	w.WriteCalibration(model.CalibrationRead{model.Timestamp{"", 0}, 75})
	w.Flush()

	if statsWriter.total != 1 {
		t.Errorf("individual write test failed: got a total of %d but expected %d", statsWriter.total, 10)
	}

	if statsWriter.batchCount != 1 {
		t.Errorf("individual write test failed: got a total of %d but expected %d", statsWriter.batchCount, 1)
	}
}

func TestSimpleWriteLargerThanOneBatch(t *testing.T) {
	statsWriter := new(statsCalibrationWriter)
	w := NewWriterSize(statsWriter, 10)
	calibrations := make([]model.CalibrationRead, 11)
	for i := 0; i < 11; i++ {
		calibrations[i] = model.CalibrationRead{model.Timestamp{"", 0}, 75}
	}
	w.WriteCalibrations(calibrations)

	if statsWriter.total != 10 {
		t.Errorf("simple write larger than one batch test failed: got a total of %d but expected %d", statsWriter.total, 10)
	}

	if statsWriter.batchCount != 1 {
		t.Errorf("simple write larger than one batch test: got a total of %d but expected %d", statsWriter.batchCount, 1)
	}

	// Flushing should cause the extra calibration to be written
	w.Flush()

	if statsWriter.total != 11 {
		t.Errorf("simple write larger than one batch test: got a total of %d but expected %d", statsWriter.total, 11)
	}

	if statsWriter.batchCount != 2 {
		t.Errorf("simple write larger than one batch test: got a total of %d but expected %d", statsWriter.batchCount, 2)
	}
}

func TestWriteTwoFullBatches(t *testing.T) {
	statsWriter := new(statsCalibrationWriter)
	w := NewWriterSize(statsWriter, 10)
	calibrations := make([]model.CalibrationRead, 20)
	for i := 0; i < 20; i++ {
		calibrations[i] = model.CalibrationRead{model.Timestamp{"", 0}, 75}
	}
	w.WriteCalibrations(calibrations)

	if statsWriter.total != 10 {
		t.Errorf("two full batch test failed: got a total of %d but expected %d", statsWriter.total, 10)
	}

	if statsWriter.batchCount != 1 {
		t.Errorf("two full batch test failed: got a total of %d but expected %d", statsWriter.batchCount, 1)
	}

	// Flushing should cause the extra batch to be written
	w.Flush()

	if statsWriter.total != 20 {
		t.Errorf("two full batch test failed: got a total of %d but expected %d", statsWriter.total, 20)
	}

	if statsWriter.batchCount != 2 {
		t.Errorf("two full batch test failed: got a total of %d but expected %d", statsWriter.batchCount, 2)
	}
}
