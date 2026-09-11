package handler

import "github.com/rio9466/easy-admin/server/internal/domain/analytics"

// publicPageViewRequest is the raw public page-view report body (§4.10).
type publicPageViewRequest struct {
	Path     string `json:"path"`
	Referrer string `json:"referrer"`
	Locale   string `json:"locale"`
}

func (r publicPageViewRequest) toInput() analytics.PageViewInput {
	return analytics.PageViewInput{Path: r.Path, Referrer: r.Referrer, Locale: r.Locale}
}

// analyticsSourceData is one aggregated traffic source (§6.3).
type analyticsSourceData struct {
	Source string `json:"source"`
	Count  int64  `json:"count"`
}

// analyticsOverviewData is the administrator overview payload (§6.3).
type analyticsOverviewData struct {
	Range       string                `json:"range"`
	PV          int64                 `json:"pv"`
	SourceCount int64                 `json:"source_count"`
	Sources     []analyticsSourceData `json:"sources"`
}

func toAnalyticsOverviewData(view *analytics.Overview) analyticsOverviewData {
	sources := make([]analyticsSourceData, 0, len(view.Sources))
	for _, source := range view.Sources {
		sources = append(sources, analyticsSourceData{Source: source.Source, Count: source.Count})
	}
	return analyticsOverviewData{
		Range:       view.Range,
		PV:          view.PV,
		SourceCount: view.SourceCount,
		Sources:     sources,
	}
}
