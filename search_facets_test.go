// Copyright 2012 Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package elastic

import (
	_ "encoding/json"
	_ "net/http"
	"testing"
	"time"
)

func TestSearchFacets(t *testing.T) {
	client := setupTestClientAndCreateIndex(t)

	tweet1 := tweet{
		User:     "olivere",
		Retweets: 108,
		Message:  "Welcome to Golang and ElasticSearch.",
		Created:  time.Date(2012, 12, 12, 17, 38, 34, 0, time.UTC),
	}
	tweet2 := tweet{
		User:     "olivere",
		Retweets: 0,
		Message:  "Another unrelated topic.",
		Created:  time.Date(2012, 10, 10, 8, 12, 03, 0, time.UTC),
	}
	tweet3 := tweet{
		User:     "sandrae",
		Retweets: 12,
		Message:  "Cycling is fun.",
		Created:  time.Date(2011, 11, 11, 10, 58, 12, 0, time.UTC),
	}

	// Add all documents
	_, err := client.Index().Index(testIndexName).Type("tweet").Id("1").BodyJson(&tweet1).Do()
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Index().Index(testIndexName).Type("tweet").Id("2").BodyJson(&tweet2).Do()
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Index().Index(testIndexName).Type("tweet").Id("3").BodyJson(&tweet3).Do()
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Flush().Index(testIndexName).Do()
	if err != nil {
		t.Fatal(err)
	}

	// Match all should return all documents
	all := NewMatchAllQuery()

	// Terms Facet by user name
	userFacet := NewTermsFacet("user").Size(10).Order("count")

	// Range Facet by retweets
	retweetsFacet := NewRangeFacet("retweets").Lt(10).Between(10, 100).Gt(100)

	// Histogram Facet by retweets
	retweetsHistoFacet := NewHistogramFacet("retweets").Interval(100)

	// Histogram Facet with time interval by retweets
	retweetsTimeHistoFacet := NewHistogramFacet("retweets").TimeInterval("1m")

	// Date Histogram Facet by creation date
	dateHisto := NewDateHistogramFacet("created").Interval("year")

	// Date Histogram Facet with Key and Value field by creation date
	dateHistoWithKeyValue := NewDateHistogramFacet("createdWithKeyValue").
		Interval("year").
		KeyField("created").
		ValueField("retweets")

	// Query Facet
	queryFacet := NewQueryFacet(NewTermQuery("user", "olivere")).Order("term").Global(true)

	// Run query
	searchResult, err := client.Search().Index(testIndexName).
		Query(&all).
		Facet("user", userFacet).
		Facet("retweets", retweetsFacet).
		Facet("retweetsHistogram", retweetsHistoFacet).
		Facet("retweetsTimeHisto", retweetsTimeHistoFacet).
		Facet("dateHisto", dateHisto).
		Facet("createdWithKeyValue", dateHistoWithKeyValue).
		Facet("queryFacet", queryFacet).
		//Pretty(true).Debug(true).
		Do()
	if err != nil {
		t.Fatal(err)
	}
	if searchResult.Hits == nil {
		t.Errorf("expected SearchResult.Hits != nil; got nil")
	}
	if searchResult.Hits.TotalHits != 3 {
		t.Errorf("expected SearchResult.Hits.TotalHits = %d; got %d", 3, searchResult.Hits.TotalHits)
	}
	if len(searchResult.Hits.Hits) != 3 {
		t.Errorf("expected len(SearchResult.Hits.Hits) = %d; got %d", 3, len(searchResult.Hits.Hits))
	}
	if searchResult.Facets == nil {
		t.Errorf("expected SearchResult.Facets != nil; got nil")
	}

	// Search for non-existent facet field should return (nil, false)
	facet, found := searchResult.Facets["no-such-field"]
	if found {
		t.Errorf("expected SearchResult.Facets.For(...) = %v; got %v", false, found)
	}
	if facet != nil {
		t.Errorf("expected SearchResult.Facets.For(...) = nil; got %v", facet)
	}

	// Search for existent facet should return (facet, true)
	facet, found = searchResult.Facets["user"]
	if !found {
		t.Errorf("expected searchResult.Facets[\"user\"] = %v; got %v", true, found)
	}
	if facet == nil {
		t.Errorf("expected searchResult.Facets[\"user\"] != nil; got nil")
	}

	// Check facet details
	if facet.Type != "terms" {
		t.Errorf("expected searchResult.Facets[\"user\"].Type = %v; got %v", "terms", facet.Type)
	}
	if facet.Total != 3 {
		t.Errorf("expected searchResult.Facets[\"user\"].Total = %v; got %v", 3, facet.Total)
	}
	if len(facet.Terms) != 2 {
		t.Errorf("expected len(searchResult.Facets[\"user\"].Terms) = %v; got %v", 2, len(facet.Terms))
	}

	// Search for range facet should return (facet, true)
	facet, found = searchResult.Facets["retweets"]
	if !found {
		t.Errorf("expected searchResult.Facets[\"retweets\"] = %v; got %v", true, found)
	}
	if facet == nil {
		t.Errorf("expected searchResult.Facets[\"retweets\"] != nil; got nil")
	}

	// Check facet details
	if facet.Type != "range" {
		t.Errorf("expected searchResult.Facets[\"retweets\"].Type = %v; got %v", "range", facet.Type)
	}
	if len(facet.Ranges) != 3 {
		t.Errorf("expected len(searchResult.Facets[\"retweets\"].Ranges) = %v; got %v", 3, len(facet.Ranges))
	}

	if facet.Ranges[0].Count != 1 {
		t.Errorf("expected searchResult.Facets[\"retweets\"][0].Count = %v; got %v", 1, facet.Ranges[0].Count)
	}
	if facet.Ranges[0].TotalCount != 1 {
		t.Errorf("expected searchResult.Facets[\"retweets\"][0].TotalCount = %v; got %v", 1, facet.Ranges[0].TotalCount)
	}
	if facet.Ranges[0].From != nil {
		t.Errorf("expected searchResult.Facets[\"retweets\"][0].From = %v; got %v", nil, facet.Ranges[0].From)
	}
	if to := facet.Ranges[0].To; to == nil || (*to) != 10.0 {
		t.Errorf("expected searchResult.Facets[\"retweets\"][0].To = %v; got %v", 10.0, to)
	}

	if facet.Ranges[1].Count != 1 {
		t.Errorf("expected searchResult.Facets[\"retweets\"][1].Count = %v; got %v", 1, facet.Ranges[1].Count)
	}
	if facet.Ranges[1].TotalCount != 1 {
		t.Errorf("expected searchResult.Facets[\"retweets\"][1].TotalCount = %v; got %v", 1, facet.Ranges[1].TotalCount)
	}
	if from := facet.Ranges[1].From; from == nil || (*from) != 10.0 {
		t.Errorf("expected searchResult.Facets[\"retweets\"][1].From = %v; got %v", 10.0, from)
	}
	if to := facet.Ranges[1].To; to == nil || (*to) != 100.0 {
		t.Errorf("expected searchResult.Facets[\"retweets\"][1].To = %v; got %v", 100.0, facet.Ranges[1].To)
	}

	if facet.Ranges[2].Count != 1 {
		t.Errorf("expected searchResult.Facets[\"retweets\"][2].Count = %v; got %v", 1, facet.Ranges[2].Count)
	}
	if facet.Ranges[2].TotalCount != 1 {
		t.Errorf("expected searchResult.Facets[\"retweets\"][2].TotalCount = %v; got %v", 1, facet.Ranges[2].TotalCount)
	}
	if from := facet.Ranges[2].From; from == nil || (*from) != 100.0 {
		t.Errorf("expected searchResult.Facets[\"retweets\"][2].From = %v; got %v", 100.0, facet.Ranges[2].From)
	}
	if facet.Ranges[2].To != nil {
		t.Errorf("expected searchResult.Facets[\"retweets\"][2].To = %v; got %v", nil, facet.Ranges[2].To)
	}

	// Search for histogram facet should return (facet, true)
	facet, found = searchResult.Facets["retweetsHistogram"]
	if !found {
		t.Errorf("expected searchResult.Facets[\"retweetsHistogram\"] = %v; got %v", true, found)
	}
	if facet == nil {
		t.Errorf("expected searchResult.Facets[\"retweetsHistogram\"] != nil; got nil")
	}

	// Check facet details
	if facet.Type != "histogram" {
		t.Errorf("expected searchResult.Facets[\"retweetsHistogram\"].Type = %v; got %v", "histogram", facet.Type)
	}
	if len(facet.Entries) != 2 {
		t.Errorf("expected len(searchResult.Facets[\"retweetsHistogram\"].Entries) = %v; got %v", 3, len(facet.Entries))
	}
	if facet.Entries[0].Key.(float64) != 0 {
		t.Errorf("expected searchResult.Facets[\"retweetsHistogram\"].Entries[0].Key = %v; got %v", 0, facet.Entries[0].Key)
	}
	if facet.Entries[0].Count != 2 {
		t.Errorf("expected searchResult.Facets[\"retweetsHistogram\"].Entries[0].Count = %v; got %v", 2, facet.Entries[0].Count)
	}
	if facet.Entries[1].Key.(float64) != 100 {
		t.Errorf("expected searchResult.Facets[\"retweetsHistogram\"].Entries[1].Key = %v; got %v", 100, facet.Entries[1].Key)
	}
	if facet.Entries[1].Count != 1 {
		t.Errorf("expected searchResult.Facets[\"retweetsHistogram\"].Entries[1].Count = %v; got %v", 1, facet.Entries[1].Count)
	}

	// Search for histogram facet with time interval should return (facet, true)
	facet, found = searchResult.Facets["retweetsTimeHisto"]
	if !found {
		t.Errorf("expected searchResult.Facets[\"retweetsTimeHisto\"] = %v; got %v", true, found)
	}
	if facet == nil {
		t.Errorf("expected searchResult.Facets[\"retweetsTimeHisto\"] != nil; got nil")
	}

	// Search for date histogram facet
	facet, found = searchResult.Facets["dateHisto"]
	if !found {
		t.Errorf("expected searchResult.Facets[\"dateHisto\"] = %v; got %v", true, found)
	}
	if facet == nil {
		t.Errorf("expected searchResult.Facets[\"dateHisto\"] != nil; got nil")
	}
	if facet.Entries[0].Time != 1293840000000 {
		t.Errorf("expected searchResult.Facets[\"dateHisto\"].Entries[0].Time = %v; got %v", 1293840000000, facet.Entries[0].Time)
	}
	if facet.Entries[0].Count != 1 {
		t.Errorf("expected searchResult.Facets[\"dateHisto\"].Entries[0].Count = %v; got %v", 1, facet.Entries[0].Count)
	}
	if facet.Entries[1].Time != 1325376000000 {
		t.Errorf("expected searchResult.Facets[\"dateHisto\"].Entries[1].Time = %v; got %v", 1325376000000, facet.Entries[0].Time)
	}
	if facet.Entries[1].Count != 2 {
		t.Errorf("expected searchResult.Facets[\"dateHisto\"].Entries[1].Count = %v; got %v", 2, facet.Entries[1].Count)
	}

	// Search for date histogram with key/value fields facet
	facet, found = searchResult.Facets["createdWithKeyValue"]
	if !found {
		t.Errorf("expected searchResult.Facets[\"createdWithKeyValue\"] = %v; got %v", true, found)
	}
	if facet == nil {
		t.Errorf("expected searchResult.Facets[\"createdWithKeyValue\"] != nil; got nil")
	}
	if len(facet.Entries) != 2 {
		t.Errorf("expected len(searchResult.Facets[\"createdWithKeyValue\"].Entries) = %v; got %v", 2, len(facet.Entries))
	}
	if facet.Entries[0].Time != 1293840000000 {
		t.Errorf("expected searchResult.Facets[\"createdWithKeyValue\"].Entries[0].Time = %v; got %v", 1293840000000, facet.Entries[0].Time)
	}
	if facet.Entries[0].Count != 1 {
		t.Errorf("expected searchResult.Facets[\"createdWithKeyValue\"].Entries[0].Count = %v; got %v", 1, facet.Entries[0].Count)
	}
	if facet.Entries[0].Min.(float64) != 12.0 {
		t.Errorf("expected searchResult.Facets[\"createdWithKeyValue\"].Entries[0].Min = %v; got %v", 12.0, facet.Entries[0].Min)
	}
	if facet.Entries[0].Max.(float64) != 12.0 {
		t.Errorf("expected searchResult.Facets[\"createdWithKeyValue\"].Entries[0].Max = %v; got %v", 12.0, facet.Entries[0].Max)
	}
	if facet.Entries[0].Total != 12.0 {
		t.Errorf("expected searchResult.Facets[\"createdWithKeyValue\"].Entries[0].Total = %v; got %v", 12.0, facet.Entries[0].Total)
	}
	if facet.Entries[0].TotalCount != 1 {
		t.Errorf("expected searchResult.Facets[\"createdWithKeyValue\"].Entries[0].TotalCount = %v; got %v", 1, facet.Entries[0].TotalCount)
	}
	if facet.Entries[0].Mean != 12.0 {
		t.Errorf("expected searchResult.Facets[\"createdWithKeyValue\"].Entries[0].Mean = %v; got %v", 12.0, facet.Entries[0].Mean)
	}
	if facet.Entries[1].Time != 1325376000000 {
		t.Errorf("expected searchResult.Facets[\"createdWithKeyValue\"].Entries[1].Time = %v; got %v", 1325376000000, facet.Entries[1].Time)
	}
	if facet.Entries[1].Count != 2 {
		t.Errorf("expected searchResult.Facets[\"createdWithKeyValue\"].Entries[1].Count = %v; got %v", 2, facet.Entries[1].Count)
	}
	if facet.Entries[1].Min.(float64) != 0.0 {
		t.Errorf("expected searchResult.Facets[\"createdWithKeyValue\"].Entries[1].Min = %v; got %v", 0.0, facet.Entries[1].Min)
	}
	if facet.Entries[1].Max.(float64) != 108.0 {
		t.Errorf("expected searchResult.Facets[\"createdWithKeyValue\"].Entries[1].Max = %v; got %v", 108.0, facet.Entries[1].Max)
	}
	if facet.Entries[1].Total != 108.0 {
		t.Errorf("expected searchResult.Facets[\"createdWithKeyValue\"].Entries[1].Total = %v; got %v", 108.0, facet.Entries[1].Total)
	}
	if facet.Entries[1].TotalCount != 2 {
		t.Errorf("expected searchResult.Facets[\"createdWithKeyValue\"].Entries[1].TotalCount = %v; got %v", 2, facet.Entries[1].TotalCount)
	}
	if facet.Entries[1].Mean != 54.0 {
		t.Errorf("expected searchResult.Facets[\"createdWithKeyValue\"].Entries[1].Mean = %v; got %v", 54.0, facet.Entries[1].Mean)
	}

}
