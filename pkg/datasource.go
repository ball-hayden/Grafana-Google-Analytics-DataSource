package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
	"github.com/patrickmn/go-cache"
)

// GoogleAnalyticsDataSource handler for google sheets
type GoogleAnalyticsDataSource struct {
	analytics *GoogleAnalytics
}

// NewDataSource creates the google analytics datasource and sets up all the routes
func NewDataSource(mux *http.ServeMux) *GoogleAnalyticsDataSource {
	cache := cache.New(300*time.Second, 5*time.Second)
	ds := &GoogleAnalyticsDataSource{
		analytics: &GoogleAnalytics{
			Cache: cache,
		},
	}

	mux.HandleFunc("/properties", ds.handleResourceProperties)
	mux.HandleFunc("/property/timezone", ds.handleResourceProfileTimezone)
	mux.HandleFunc("/dimensions", ds.handleResourceDimensions)
	mux.HandleFunc("/metrics", ds.handleResourceMetrics)
	return ds
}

// CheckHealth checks if the plugin is running properly
func (ds *GoogleAnalyticsDataSource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	var status = backend.HealthStatusOk
	var message = "Success"

	config, err := LoadSettings(req.PluginContext)

	if err != nil {
		log.DefaultLogger.Error("Fail LoadSetting", err.Error())
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Setting Configuration Read Fail",
		}, nil
	}

	client, err := NewGoogleClient(ctx, config)
	if err != nil {
		log.DefaultLogger.Error("Fail NewGoogleClient", err.Error())
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Invalid config",
		}, nil
	}

	properties, err := client.getPropertiesList()
	if err != nil {
		log.DefaultLogger.Error("Fail getAllPropertiesList", err.Error())
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Invalid config",
		}, nil
	}

	testData := QueryModel{properties[0].Name, "yesterday", "today", "a", []string{"ga:sessions"}, "ga:dateHour", []string{}, "UTC", ""}
	res, err := client.getReport(testData)

	if err != nil {
		log.DefaultLogger.Error("GET request to analyticsreporting/v4 returned error", err.Error())
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Test Request Fail",
		}, nil
	}

	if res != nil {
		log.DefaultLogger.Info("HTTPStatusCode", "status", res.HTTPStatusCode)
		log.DefaultLogger.Info("res", res)
	}

	return &backend.CheckHealthResult{
		Status:  status,
		Message: message,
	}, nil
}

// QueryData queries for data.
func (ds *GoogleAnalyticsDataSource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	res := backend.NewQueryDataResponse()
	config, err := LoadSettings(req.PluginContext)
	if err != nil {
		return nil, err
	}

	client, err := NewGoogleClient(ctx, config)
	if err != nil {
		return nil, err
	}

	for _, query := range req.Queries {
		frames, err := ds.analytics.Query(client, query)
		if err != nil {
			log.DefaultLogger.Error(err.Error())
			continue
			// return nil, err
		}
		res.Responses[query.RefID] = backend.DataResponse{Frames: *frames, Error: err}
	}

	return res, nil
}

func writeResult(rw http.ResponseWriter, path string, val interface{}, err error) {
	response := make(map[string]interface{})
	code := http.StatusOK
	if err != nil {
		response["error"] = err.Error()
		code = http.StatusBadRequest
	} else {
		response[path] = val
	}

	body, err := json.Marshal(response)
	if err != nil {
		body = []byte(err.Error())
		code = http.StatusInternalServerError
	}
	_, err = rw.Write(body)
	if err != nil {
		code = http.StatusInternalServerError
	}
	rw.WriteHeader(code)
}

func (ds *GoogleAnalyticsDataSource) handleResourceProperties(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		return
	}

	ctx := req.Context()
	config, err := LoadSettings(httpadapter.PluginConfigFromContext(ctx))
	if err != nil {
		writeResult(rw, "?", nil, err)
		return
	}

	res, err := ds.analytics.GetProperties(ctx, config)
	writeResult(rw, "properties", res, err)
}

func (ds *GoogleAnalyticsDataSource) handleResourceDimensions(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		return
	}

	res, err := ds.analytics.GetDimensions()
	writeResult(rw, "dimensions", res, err)
}

func (ds *GoogleAnalyticsDataSource) handleResourceMetrics(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		return
	}

	res, err := ds.analytics.GetMetrics()
	writeResult(rw, "metrics", res, err)
}

func (ds *GoogleAnalyticsDataSource) handleResourceProfileTimezone(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		return
	}

	ctx := req.Context()
	config, err := LoadSettings(httpadapter.PluginConfigFromContext(ctx))
	if err != nil {
		writeResult(rw, "?", nil, err)
		return
	}

	res, err := ds.analytics.GetProfileTimezone(ctx, config, req.URL.Query().Get("propertyName"))
	writeResult(rw, "timezone", res, err)
}
