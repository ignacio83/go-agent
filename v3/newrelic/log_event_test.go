package newrelic

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestWriteJSON(t *testing.T) {
	event := logEvent{
		severity:  "INFO",
		message:   "test message",
		timestamp: 123456,
	}
	actual, err := event.MarshalJSON()
	if err != nil {
		t.Error(err)
	}

	expect := `{"level":"INFO","message":"test message","timestamp":123456}`
	actualString := string(actual)
	if expect != actualString {
		t.Errorf("Log json did not build correctly: expecting %s, got %s", expect, actualString)
	}
}

func TestToLogEvent(t *testing.T) {
	type testcase struct {
		name          string
		data          LogData
		expectEvent   logEvent
		expectErr     error
		skipTimestamp bool
	}

	testcases := []testcase{
		{
			name: "context nil",
			data: LogData{
				Timestamp: 123456,
				Severity:  "info",
				Message:   "test 123",
			},
			expectEvent: logEvent{
				timestamp: 123456,
				severity:  "info",
				message:   "test 123",
			},
		},
		{
			name: "severity empty",
			data: LogData{
				Timestamp: 123456,
				Message:   "test 123",
			},
			expectEvent: logEvent{
				timestamp: 123456,
				severity:  "UNKNOWN",
				message:   "test 123",
			},
		},
		{
			name: "no timestamp",
			data: LogData{
				Severity: "info",
				Message:  "test 123",
			},
			expectEvent: logEvent{
				severity: "info",
				message:  "test 123",
			},
			skipTimestamp: true,
		},
		{
			name: "message too large",
			data: LogData{
				Timestamp: 123456,
				Severity:  "info",
				Message:   randomString(32769),
			},
			expectErr: errLogMessageTooLarge,
		},
	}

	for _, testcase := range testcases {
		actualEvent, err := testcase.data.toLogEvent()

		if testcase.expectErr != err {
			t.Error(fmt.Errorf("%s: expected error %v, got %v", testcase.name, testcase.expectErr, err))
		}

		if testcase.expectErr == nil {
			expect := testcase.expectEvent
			if expect.message != actualEvent.message {
				t.Error(fmt.Errorf("%s: expected message %s, got %s", testcase.name, expect.message, actualEvent.message))
			}
			if expect.severity != actualEvent.severity {
				t.Error(fmt.Errorf("%s: expected severity %s, got %s", testcase.name, expect.severity, actualEvent.severity))
			}
			if actualEvent.timestamp == 0 {
				t.Errorf("timestamp was not set on test %s", testcase.name)
			}
			if expect.timestamp != actualEvent.timestamp && !testcase.skipTimestamp {
				t.Error(fmt.Errorf("%s: expected timestamp %d, got %d", testcase.name, expect.timestamp, actualEvent.timestamp))
			}
		}
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func TestWriteJSONWithTrace(t *testing.T) {
	event := logEvent{
		severity:  "INFO",
		message:   "test message",
		timestamp: 123456,
		traceID:   "123Ad234",
		spanID:    "adf3441",
	}
	actual, err := event.MarshalJSON()
	if err != nil {
		t.Error(err)
	}

	expect := `{"level":"INFO","message":"test message","span.id":"adf3441","trace.id":"123Ad234","timestamp":123456}`
	actualString := string(actual)
	if expect != actualString {
		t.Errorf("Log json did not build correctly: expecting %s, got %s", expect, actualString)
	}
}

func BenchmarkToLogEvent(b *testing.B) {
	data := LogData{
		Timestamp: 123456,
		Severity:  "INFO",
		Message:   "test message",
	}

	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		data.toLogEvent()
	}

}

func recordLogBenchmarkHelper(b *testing.B, data *LogData, h *harvest) {
	event, _ := data.toLogEvent()
	event.MergeIntoHarvest(h)
}

func BenchmarkRecordLog(b *testing.B) {
	harvest := newHarvest(time.Now(), testHarvestCfgr)
	data := LogData{
		Timestamp: 123456,
		Severity:  "INFO",
		Message:   "test message",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		recordLogBenchmarkHelper(b, &data, harvest)
	}
}

func BenchmarkWriteJSON(b *testing.B) {
	data := LogData{
		Timestamp: 123456,
		Severity:  "INFO",
		Message:   "This is a log message that represents an estimate for how long the average log message is. The average log payload is 700 bytese.",
	}

	event, err := data.toLogEvent()
	if err != nil {
		b.Fail()
	}

	buf := bytes.NewBuffer(make([]byte, 0, averageLogSizeEstimate))

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		event.WriteJSON(buf)
	}
}