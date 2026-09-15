package dashboard

import (
	"net/url"
	"testing"
)

func TestParseDubboContract(t *testing.T) {
	tests := []struct {
		identifier string
		want       dubboContract
		ok         bool
	}{
		{
			identifier: "dubbo:com.example.DecisionService|risk|1.0.0|tri",
			want:       dubboContract{Interface: "com.example.DecisionService", Group: "risk", Version: "1.0.0", Protocol: "tri"},
			ok:         true,
		},
		{
			identifier: "dubbo:com.example.DecisionService|risk|1.0.0|tri|decide|2",
			want:       dubboContract{Interface: "com.example.DecisionService", Group: "risk", Version: "1.0.0", Protocol: "tri", Method: "decide", Arity: 2, HasArity: true},
			ok:         true,
		},
		{identifier: "dubbo:DecisionService||||", ok: false},
		{identifier: "http:com.example.DecisionService", ok: false},
	}

	for _, test := range tests {
		got, ok := parseDubboContract(test.identifier)
		if ok != test.ok {
			t.Fatalf("parseDubboContract(%q) ok = %v, want %v", test.identifier, ok, test.ok)
		}
		if ok && got != test.want {
			t.Fatalf("parseDubboContract(%q) = %#v, want %#v", test.identifier, got, test.want)
		}
	}
}

func TestParseDubboQueryClampsAndValidates(t *testing.T) {
	query, err := parseDubboQuery(url.Values{"page": {"2"}, "page_size": {"500"}, "consumer_repo": {"aps"}})
	if err != nil {
		t.Fatal(err)
	}
	if query.Page != 2 || query.PageSize != 100 || query.ConsumerRepo != "aps" {
		t.Fatalf("query = %#v", query)
	}
	if _, err := parseDubboQuery(url.Values{"page": {"zero"}}); err == nil {
		t.Fatal("expected invalid page error")
	}
}

func TestBuildDubboReportFiltersGroupsAndPaginates(t *testing.T) {
	group := &DashGroup{
		Name: "platform",
		Links: []CrossRepoLink{
			{Source: "aps::c1", Target: "des::p1", Kind: "calls", Method: "dubbo", Channel: "dubbo", Confidence: 1, Identifier: "dubbo:com.example.AService|risk|1.0.0|tri", Properties: map[string]string{"match": "interface_group_version_protocol"}},
			{Source: "ffs::c2", Target: "des::p1", Kind: "calls", Method: "dubbo", Channel: "dubbo", Confidence: 1, Identifier: "dubbo:com.example.AService|risk|1.0.0|tri"},
			{Source: "aps::cc3", Target: "pfs::p2", Kind: "calls", Method: "dubbo", Channel: "dubbo", Confidence: 1, Identifier: "dubbo:com.example.BService|||"},
			{Source: "aps::http", Target: "des::http", Kind: "calls", Method: "http", Identifier: "GET /decide"},
		},
	}

	report := buildDubboReport(group, dubboQuery{Page: 1, PageSize: 1, ConsumerRepo: "aps"}, nil)
	if report.Summary.Services != 2 || report.Summary.Links != 3 || report.Summary.Consumers != 3 || report.Summary.Providers != 2 {
		t.Fatalf("summary = %#v", report.Summary)
	}
	if report.TotalServices != 2 || report.TotalPages != 2 || len(report.Services) != 1 {
		t.Fatalf("pagination = total %d pages %d services %d", report.TotalServices, report.TotalPages, len(report.Services))
	}
	service := report.Services[0]
	if service.Interface != "com.example.AService" || service.LinkCount != 1 {
		t.Fatalf("service = %#v", service)
	}
	if len(service.Consumers) != 1 || service.Consumers[0].Repo != "aps" {
		t.Fatalf("consumers = %#v", service.Consumers)
	}
	if len(report.Facets.ConsumerRepos) != 2 || len(report.Facets.ProviderRepos) != 2 {
		t.Fatalf("facets = %#v", report.Facets)
	}
}
