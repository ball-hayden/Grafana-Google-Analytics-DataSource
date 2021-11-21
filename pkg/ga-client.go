package main

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/analytics/v3"
	"google.golang.org/api/option"

	admin "google.golang.org/api/analyticsadmin/v1alpha"
	data "google.golang.org/api/analyticsdata/v1beta"
)

type GoogleClient struct {
	data  *data.Service
	admin *admin.Service
}

func NewGoogleClient(ctx context.Context, auth *DatasourceSettings) (*GoogleClient, error) {
	dataService, dataError := createDataService(ctx, auth)
	if dataError != nil {
		return nil, dataError
	}

	adminService, adminError := createAdminService(ctx, auth)
	if adminError != nil {
		return nil, adminError
	}

	return &GoogleClient{dataService, adminService}, nil
}

func createDataService(ctx context.Context, auth *DatasourceSettings) (*data.Service, error) {
	jwtConfig, err := google.JWTConfigFromJSON([]byte(auth.JWT), data.AnalyticsReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("error parsing JWT file: %w", err)
	}

	client := jwtConfig.Client(ctx)
	return data.NewService(ctx, option.WithHTTPClient(client))
}

func createAdminService(ctx context.Context, auth *DatasourceSettings) (*admin.Service, error) {
	jwtConfig, err := google.JWTConfigFromJSON([]byte(auth.JWT), analytics.AnalyticsReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("error parsing JWT file: %w", err)
	}

	client := jwtConfig.Client(ctx)
	return admin.NewService(ctx, option.WithHTTPClient(client))
}

func (client *GoogleClient) getPropertiesList() ([]*admin.GoogleAnalyticsAdminV1alphaProperty, error) {
	propertiesService := admin.NewPropertiesService(client.admin)
	properties, err := propertiesService.List().Do()
	if err != nil {
		log.DefaultLogger.Error(err.Error())
		return nil, err
	}

	return properties.Properties, nil
}

func (client *GoogleClient) getReport(query QueryModel) (*data.RunReportResponse, error) {
	log.DefaultLogger.Info("getReport", "queries", query)
	Metrics := []*data.Metric{}
	Dimensions := []*data.Dimension{}

	for _, metric := range query.Metrics {
		Metrics = append(Metrics, &data.Metric{Expression: metric})
	}
	for _, dimension := range query.Dimensions {
		Dimensions = append(Dimensions, &data.Dimension{Name: dimension})
	}

	reportRequest := data.RunReportRequest{
		Property: query.PropertyID,
		DateRanges: []*data.DateRange{
			// Create the DateRange object.
			{StartDate: query.StartDate, EndDate: query.EndDate},
		},
		Metrics:    Metrics,
		Dimensions: Dimensions,
	}

	log.DefaultLogger.Info("getReport", "reportRequest", reportRequest)

	propertiesService := data.NewPropertiesService(client.data)

	req := propertiesService.RunReport(
		query.PropertyID,
		&reportRequest,
	)

	log.DefaultLogger.Info("Doing GET request from analytics reporting", "req", req)
	// Call the BatchGet method and return the response.
	report, err := req.Do()
	if err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	return report, nil
}
