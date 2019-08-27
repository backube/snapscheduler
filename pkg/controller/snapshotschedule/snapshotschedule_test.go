package snapshotschedule

import (
	"testing"
	"time"
)

func TestGetNextSnapTime(t *testing.T) {
	var tests = []struct {
		inCronspec string
		inTime     string
		wantTime   string
		wantErr    bool
	}{
		{"@hourly", "2013-02-01T11:04:05Z", "2013-02-01T12:00:00Z", false},
		{"invalid_spec", "2013-02-01T11:04:05Z", "unused", true},
	}

	for _, test := range tests {
		inTime, _ := time.Parse(timeFormat, test.inTime)
		gotTime, err := getNextSnapTime(test.inCronspec, inTime)
		if err != nil {
			if !test.wantErr {
				t.Errorf("unexpected error: %v", err)
			}
			continue
		}
		if test.wantErr {
			t.Errorf("expected an error, but didn't get it")
			continue
		}
		wantTime, _ := time.Parse(timeFormat, test.wantTime)
		if gotTime != wantTime {
			t.Errorf("expected: %v -- got: %v", wantTime, gotTime)
		}
	}
}
