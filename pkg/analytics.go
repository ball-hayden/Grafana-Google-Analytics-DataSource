package main

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/patrickmn/go-cache"
)

// GoogleAnalyticsDataSource handler for google sheets
type GoogleAnalytics struct {
	Cache *cache.Cache
}

func (ga *GoogleAnalytics) Query(client *GoogleClient, query backend.DataQuery) (*data.Frames, error) {
	queryModel, err := GetQueryModel(query)
	if err != nil {
		log.DefaultLogger.Error(err.Error())
		return nil, fmt.Errorf("failed to read query: %w", err)
	}

	if len(queryModel.PropertyID) < 1 {
		return nil, fmt.Errorf("Required PropertyID")
	}

	report, err := client.getReport(*queryModel)
	if err != nil {
		log.DefaultLogger.Error("Query failed", "error", err)
		return nil, err
	}

	return transformReportResponseToDataFrames(report, queryModel.RefID, queryModel.Timezone)
}

func (ga *GoogleAnalytics) GetProperties(ctx context.Context, config *DatasourceSettings) (map[string]string, error) {
	client, err := NewGoogleClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google API client: %w", err)
	}

	cacheKey := fmt.Sprintf("analytics:properties")
	if item, _, found := ga.Cache.GetWithExpiration(cacheKey); found {
		return item.(map[string]string), nil
	}

	properties, err := client.getPropertiesList()
	if err != nil {
		return nil, err
	}

	propertyNames := map[string]string{}
	for _, i := range properties {
		propertyNames[i.Name] = i.DisplayName
	}

	ga.Cache.Set(cacheKey, propertyNames, 60*time.Second)
	return propertyNames, nil
}

func (ga *GoogleAnalytics) GetProfileTimezone(ctx context.Context, config *DatasourceSettings, propertyName string) (string, error) {
	client, err := NewGoogleClient(ctx, config)
	if err != nil {
		return "", fmt.Errorf("failed to create Google API client: %w", err)
	}

	cacheKey := fmt.Sprintf("analytics:property:%s:timezone", propertyName)
	if item, _, found := ga.Cache.GetWithExpiration(cacheKey); found {
		return item.(string), nil
	}

	properties, err := client.getPropertiesList()
	if err != nil {
		return "", err
	}

	var timezone string
	for _, property := range properties {
		if property.Name == propertyName {
			timezone = property.TimeZone
			break
		}
	}

	ga.Cache.Set(cacheKey, timezone, 60*time.Second)
	return timezone, nil
}
