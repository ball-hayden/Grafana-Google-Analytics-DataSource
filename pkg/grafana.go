package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"

	analyticsdata "google.golang.org/api/analyticsdata/v1beta"
)

func transformReportResponseToDataFrames(report *analyticsdata.RunReportResponse, refId string, timezone string) (*data.Frames, error) {
	frames := make(data.Frames, 0, len(report.MetricHeaders))

	fieldNames := make([]string, 0, len(report.DimensionHeaders))
	fieldTypes := make([]data.FieldType, 0, len(report.DimensionHeaders))

	for _, dimension := range report.DimensionHeaders {
		fieldTypes = append(fieldTypes, data.FieldTypeString)
		fieldNames = append(fieldNames, dimension.Name)
	}

	for _, metricHeader := range report.MetricHeaders {
		var fieldConverter = getFieldConverter(metricHeader.Type)

		frame := data.NewFrameOfFieldTypes(
			metricHeader.Name,
			int(report.RowCount),
			append([]data.FieldType{fieldConverter.OutputFieldType}, fieldTypes...)...,
		)

		frame.SetFieldNames(append([]string{"value"}, fieldNames...)...)

		frames = append(frames, frame)
	}

	for _, row := range report.Rows {
		for metrixIdx, metric := range row.MetricValues {
			metricType := report.MetricHeaders[metrixIdx].Type
			fieldConverter := getFieldConverter(metricType)

			frame := frames[metrixIdx]
			value, _ := fieldConverter.Converter(metric.Value)
			frame.Fields[0].Append(value)

			for dimensionIdx, dimension := range row.DimensionValues {
				frame.Fields[dimensionIdx+1].Append(dimension.Value)
			}
		}
	}

	return &frames, nil
}

var timeConverter = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableTime,
	Converter: func(i interface{}) (interface{}, error) {
		strTime, ok := i.(string)
		if !ok {
			return nil, fmt.Errorf("expected type string, but got %T", i)
		}
		time, err := time.Parse(time.RFC3339, strTime)
		if err != nil {
			log.DefaultLogger.Info("timeConverter", "err", err)
			return nil, err
		}
		return &time, nil
	},
}

// stringConverter handles sheets STRING column types.
var stringConverter = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableString,
	Converter: func(i interface{}) (interface{}, error) {
		value, ok := i.(string)
		if !ok {
			return nil, fmt.Errorf("expected type string, but got %T", i)
		}

		return &value, nil
	},
}

// numberConverter handles sheets STRING column types.
var numberConverter = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableFloat64,
	Converter: func(i interface{}) (interface{}, error) {
		value, ok := i.(string)
		if !ok {
			return nil, fmt.Errorf("expected type string, but got %T", i)
		}

		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, fmt.Errorf("expected type string, but got %T", value)
		}

		return &num, nil
	},
}

func getFieldConverter(headerType string) data.FieldConverter {
	switch headerType {
	case "TYPE_INTEGER", "TYPE_FLOAT", "TYPE_CURRENCY", "TYPE_PERCENT":
		return numberConverter
	case "TYPE_TIME":
		return timeConverter
	default:
		return stringConverter
	}
}
