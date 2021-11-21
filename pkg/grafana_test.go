package main

import (
	"testing"

	analyticsdata "google.golang.org/api/analyticsdata/v1beta"
)

func TestTransformingReortToFrames(t *testing.T) {
	response := analyticsdata.RunReportResponse{
		DimensionHeaders: []*analyticsdata.DimensionHeader{
			{Name: "appVersion"},
			{Name: "operatingSystem"},
		},
		MetricHeaders: []*analyticsdata.MetricHeader{
			{Name: "active1DayUsers", Type: "TYPE_INTEGER"},
		},
		Rows: []*analyticsdata.Row{
			{
				DimensionValues: []*analyticsdata.DimensionValue{
					{Value: "14.0.0"},
					{Value: "iOS"},
				},
				MetricValues: []*analyticsdata.MetricValue{
					{Value: "265"},
				},
			},
			{
				DimensionValues: []*analyticsdata.DimensionValue{
					{Value: "14.0.0"},
					{Value: "Android"},
				},
				MetricValues: []*analyticsdata.MetricValue{
					{Value: "29"},
				},
			},
			{
				DimensionValues: []*analyticsdata.DimensionValue{
					{Value: "13.0.0"},
					{Value: "iOS"},
				},
				MetricValues: []*analyticsdata.MetricValue{
					{Value: "12"},
				},
			},
			{
				DimensionValues: []*analyticsdata.DimensionValue{
					{Value: "13.0.0"},
					{Value: "Android"},
				},
				MetricValues: []*analyticsdata.MetricValue{
					{Value: "5"},
				},
			},
		},
	}

	frames, err := transformReportResponseToDataFrames(&response, "refId", "timezone")

	if err != nil {
		t.Fatalf("Error transforming report response to data frames: %v", err)
	}

	if len(*frames) != 1 {
		t.Fatalf("Expected 1 frame, got %v", len(*frames))
	}

	frame := (*frames)[0]

	if frame.Name != "active1DayUsers" {
		t.Fatalf("Expected frame name to be active1DayUsers, got %v", frame.Name)
	}

	if len(frame.Fields) != 3 {
		t.Fatalf("Expected 2 fields, got %v", len(frame.Fields))
	}

	if frame.Fields[0].Name != "value" {
		t.Fatalf("Expected field name to be value, got %v", frame.Fields[2].Name)
	}

	if frame.Fields[1].Name != "appVersion" {
		t.Fatalf("Expected field name to be appVersion, got %v", frame.Fields[0].Name)
	}

	if frame.Fields[2].Name != "operatingSystem" {
		t.Fatalf("Expected field name to be operatingSystem, got %v", frame.Fields[1].Name)
	}

	if *frame.At(0, 0).(*float64) != 265 || frame.At(1, 0).(string) != "14.0.0" || frame.At(2, 0).(string) != "iOS" {
		t.Fatalf("Expected row to be 265, 14.0.0, iOS, got %v, %v, %v", frame.At(0, 0), frame.At(1, 0), frame.At(2, 0))
	}

	if *frame.At(0, 1).(*float64) != 29 || frame.At(1, 1).(string) != "14.0.0" || frame.At(2, 1).(string) != "Android" {
		t.Fatalf("Expected row to be 29, 14.0.0, Android, got %v, %v, %v", frame.At(0, 1), frame.At(1, 1), frame.At(2, 1))
	}

	if *frame.At(0, 2).(*float64) != 12 || frame.At(1, 2).(string) != "13.0.0" || frame.At(2, 2).(string) != "iOS" {
		t.Fatalf("Expected row to be 12, 13.0.0, iOS, got %v, %v, %v", frame.At(0, 2), frame.At(1, 2), frame.At(2, 2))
	}

	if *frame.At(0, 3).(*float64) != 5 || frame.At(1, 3).(string) != "13.0.0" || frame.At(2, 3).(string) != "Android" {
		t.Fatalf("Expected row to be 5, 13.0.0, Android, got %v, %v, %v", frame.At(0, 3), frame.At(1, 3), frame.At(2, 3))
	}
}
