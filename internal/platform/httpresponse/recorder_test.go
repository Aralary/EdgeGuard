package httpresponse

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecorderCapturesStatusAndResponseSize(t *testing.T) {
	response := httptest.NewRecorder()
	recorder := NewRecorder(response)

	recorder.WriteHeader(http.StatusCreated)
	if _, err := recorder.Write([]byte("created")); err != nil {
		t.Fatalf("write response: %v", err)
	}

	if recorder.StatusCode() != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", recorder.StatusCode(), http.StatusCreated)
	}
	if recorder.BytesWritten() != len("created") {
		t.Fatalf("bytes written = %d, want %d", recorder.BytesWritten(), len("created"))
	}
	if response.Code != http.StatusCreated {
		t.Fatalf("underlying status code = %d, want %d", response.Code, http.StatusCreated)
	}
}

func TestRecorderUsesOKWhenBodyIsWrittenWithoutExplicitStatus(t *testing.T) {
	response := httptest.NewRecorder()
	recorder := NewRecorder(response)

	if _, err := recorder.Write([]byte("ok")); err != nil {
		t.Fatalf("write response: %v", err)
	}

	if recorder.StatusCode() != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.StatusCode(), http.StatusOK)
	}
}

func TestRecorderIgnoresRepeatedWriteHeader(t *testing.T) {
	response := httptest.NewRecorder()
	recorder := NewRecorder(response)

	recorder.WriteHeader(http.StatusAccepted)
	recorder.WriteHeader(http.StatusInternalServerError)

	if recorder.StatusCode() != http.StatusAccepted {
		t.Fatalf("status code = %d, want %d", recorder.StatusCode(), http.StatusAccepted)
	}
}
