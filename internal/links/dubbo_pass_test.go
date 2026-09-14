package links

import (
	"path/filepath"
	"testing"
)

func TestDubboPassContractAndMethodMatch(t *testing.T) {
	root := fixtureRoot(t)
	writeFixture(t, root, fixtureGraph{Repo: "client", Entities: []map[string]any{
		dubboFixtureEntity("consumer-contract", "SCOPE.Service", "rpc_client", "consumer", "com.example.OrderFacade", "orders", "1.0.0", "", "client.java"),
		dubboFixtureMethod("consumer-method", "consumer", "com.example.OrderFacade", "orders", "1.0.0", "submit", "1", "client.java"),
	}})
	writeFixture(t, root, fixtureGraph{Repo: "server", Entities: []map[string]any{
		dubboFixtureEntity("provider-contract", "SCOPE.Service", "rpc_service", "provider", "com.example.OrderFacade", "orders", "1.0.0", "", "server.java"),
		dubboFixtureMethod("provider-method", "provider", "com.example.OrderFacade", "orders", "1.0.0", "submit", "1", "server.java"),
	}})
	home := filepath.Join(root, "home")
	result, err := RunAllPasses("dubbo-test", root, home)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := readDoc(filepath.Join(home, "groups", "dubbo-test-links.json"))
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, link := range doc.Links {
		if link.Method == MethodDubbo {
			found[link.Source+"->"+link.Target] = true
		}
	}
	if !found["client::consumer-contract->server::provider-contract"] || !found["client::consumer-method->server::provider-method"] {
		t.Fatalf("missing Dubbo contract/method links: %+v; results=%+v", found, result.Results)
	}
}

func TestDubboPassLinksUniqueUnresolvedConfigAsInferred(t *testing.T) {
	root := fixtureRoot(t)
	consumer := dubboFixtureEntity("consumer", "SCOPE.Service", "rpc_client", "consumer", "com.example.OrderFacade", "orders", "1.0.0", "", "client.java")
	consumer["properties"].(map[string]any)["group_resolved"] = "false"
	writeFixture(t, root, fixtureGraph{Repo: "client", Entities: []map[string]any{consumer}})
	writeFixture(t, root, fixtureGraph{Repo: "server", Entities: []map[string]any{
		dubboFixtureEntity("provider", "SCOPE.Service", "rpc_service", "provider", "com.example.OrderFacade", "orders", "1.0.0", "", "server.java"),
	}})
	home := filepath.Join(root, "home")
	if _, err := RunAllPasses("dubbo-unresolved", root, home); err != nil {
		t.Fatal(err)
	}
	doc, err := readDoc(filepath.Join(home, "groups", "dubbo-unresolved-links.json"))
	if err != nil {
		t.Fatal(err)
	}
	link := onlyDubboLink(t, doc)
	if link.Properties[EdgeConfidenceKey] != ConfidenceInferred {
		t.Fatalf("confidence marker = %q, want %q", link.Properties[EdgeConfidenceKey], ConfidenceInferred)
	}
	if link.Properties["match"] != "interface_compatible_metadata" {
		t.Fatalf("match = %q", link.Properties["match"])
	}
	if link.Confidence >= 1.0 {
		t.Fatalf("inferred confidence = %v, want below 1.0", link.Confidence)
	}
}

func TestDubboPassLinksMissingResolutionMarkerAsInferred(t *testing.T) {
	root := fixtureRoot(t)
	consumer := dubboFixtureEntity("consumer", "SCOPE.Service", "rpc_client", "consumer", "com.example.OrderFacade", "orders", "1.0.0", "", "client.java")
	delete(consumer["properties"].(map[string]any), "group_resolved")
	writeFixture(t, root, fixtureGraph{Repo: "client", Entities: []map[string]any{consumer}})
	writeFixture(t, root, fixtureGraph{Repo: "server", Entities: []map[string]any{
		dubboFixtureEntity("provider", "SCOPE.Service", "rpc_service", "provider", "com.example.OrderFacade", "orders", "1.0.0", "", "server.java"),
	}})
	home := filepath.Join(root, "home")
	if _, err := RunAllPasses("dubbo-missing-resolution", root, home); err != nil {
		t.Fatal(err)
	}
	doc, err := readDoc(filepath.Join(home, "groups", "dubbo-missing-resolution-links.json"))
	if err != nil {
		t.Fatal(err)
	}
	link := onlyDubboLink(t, doc)
	if link.Properties[EdgeConfidenceKey] != ConfidenceInferred {
		t.Fatalf("confidence marker = %q, want %q", link.Properties[EdgeConfidenceKey], ConfidenceInferred)
	}
}

func TestDubboPassSkipsConflictingResolvedMetadata(t *testing.T) {
	root := fixtureRoot(t)
	writeFixture(t, root, fixtureGraph{Repo: "client", Entities: []map[string]any{
		dubboFixtureEntity("consumer", "SCOPE.Service", "rpc_client", "consumer", "com.example.OrderFacade", "orders-a", "1.0.0", "", "client.java"),
	}})
	writeFixture(t, root, fixtureGraph{Repo: "server", Entities: []map[string]any{
		dubboFixtureEntity("provider", "SCOPE.Service", "rpc_service", "provider", "com.example.OrderFacade", "orders-b", "1.0.0", "", "server.java"),
	}})
	home := filepath.Join(root, "home")
	if _, err := RunAllPasses("dubbo-conflict", root, home); err != nil {
		t.Fatal(err)
	}
	assertNoDubboLinks(t, filepath.Join(home, "groups", "dubbo-conflict-links.json"))
}

func TestDubboPassSkipsUnqualifiedInterface(t *testing.T) {
	root := fixtureRoot(t)
	writeFixture(t, root, fixtureGraph{Repo: "client", Entities: []map[string]any{
		dubboFixtureEntity("consumer", "SCOPE.Service", "rpc_client", "consumer", "OrderFacade", "", "", "", "client.java"),
	}})
	writeFixture(t, root, fixtureGraph{Repo: "server", Entities: []map[string]any{
		dubboFixtureEntity("provider", "SCOPE.Service", "rpc_service", "provider", "OrderFacade", "", "", "", "server.java"),
	}})
	home := filepath.Join(root, "home")
	if _, err := RunAllPasses("dubbo-unqualified", root, home); err != nil {
		t.Fatal(err)
	}
	assertNoDubboLinks(t, filepath.Join(home, "groups", "dubbo-unqualified-links.json"))
}

func TestDubboPassResolvesWildcardCandidateFromKnownInterface(t *testing.T) {
	root := fixtureRoot(t)
	writeFixture(t, root, fixtureGraph{Repo: "client", Entities: []map[string]any{
		dubboFixtureEntity("consumer", "SCOPE.Service", "rpc_client", "consumer", "com.example.orders.OrderFacade", "", "", "", "client.java"),
	}})
	provider := dubboFixtureEntity("provider", "SCOPE.Service", "rpc_service", "provider", "OrderFacade", "", "", "", "server.java")
	provider["properties"].(map[string]any)["interface_fqn_resolved"] = "false"
	provider["properties"].(map[string]any)["interface_candidates"] = "com.example.config.OrderFacade,com.example.orders.OrderFacade,com.example.risk.OrderFacade"
	writeFixture(t, root, fixtureGraph{Repo: "server", Entities: []map[string]any{
		provider,
		{"id": "interface", "name": "OrderFacade", "qualified_name": "com.example.orders.OrderFacade", "kind": "SCOPE.Component", "source_file": "OrderFacade.java"},
	}})
	home := filepath.Join(root, "home")
	if _, err := RunAllPasses("dubbo-wildcard", root, home); err != nil {
		t.Fatal(err)
	}
	doc, err := readDoc(filepath.Join(home, "groups", "dubbo-wildcard-links.json"))
	if err != nil {
		t.Fatal(err)
	}
	onlyDubboLink(t, doc)
}

func TestDubboPassSkipsAmbiguousWildcardCandidates(t *testing.T) {
	root := fixtureRoot(t)
	writeFixture(t, root, fixtureGraph{Repo: "client", Entities: []map[string]any{
		dubboFixtureEntity("consumer", "SCOPE.Service", "rpc_client", "consumer", "com.example.orders.OrderFacade", "", "", "", "client.java"),
	}})
	provider := dubboFixtureEntity("provider", "SCOPE.Service", "rpc_service", "provider", "OrderFacade", "", "", "", "server.java")
	provider["properties"].(map[string]any)["interface_fqn_resolved"] = "false"
	provider["properties"].(map[string]any)["interface_candidates"] = "com.example.orders.OrderFacade,com.example.risk.OrderFacade"
	writeFixture(t, root, fixtureGraph{Repo: "server", Entities: []map[string]any{
		provider,
		{"id": "orders-interface", "name": "OrderFacade", "qualified_name": "com.example.orders.OrderFacade", "kind": "SCOPE.Component", "source_file": "orders/OrderFacade.java"},
		{"id": "risk-interface", "name": "OrderFacade", "qualified_name": "com.example.risk.OrderFacade", "kind": "SCOPE.Component", "source_file": "risk/OrderFacade.java"},
	}})
	home := filepath.Join(root, "home")
	if _, err := RunAllPasses("dubbo-wildcard-ambiguous", root, home); err != nil {
		t.Fatal(err)
	}
	assertNoDubboLinks(t, filepath.Join(home, "groups", "dubbo-wildcard-ambiguous-links.json"))
}

func TestDubboPassSkipsAmbiguousProviders(t *testing.T) {
	root := fixtureRoot(t)
	writeFixture(t, root, fixtureGraph{Repo: "client", Entities: []map[string]any{
		dubboFixtureEntity("consumer", "SCOPE.Service", "rpc_client", "consumer", "com.example.OrderFacade", "orders", "1.0.0", "", "client.java"),
	}})
	for _, repo := range []string{"server-a", "server-b"} {
		writeFixture(t, root, fixtureGraph{Repo: repo, Entities: []map[string]any{
			dubboFixtureEntity(repo, "SCOPE.Service", "rpc_service", "provider", "com.example.OrderFacade", "orders", "1.0.0", "", repo+".java"),
		}})
	}
	home := filepath.Join(root, "home")
	result, err := RunAllPasses("dubbo-ambiguous", root, home)
	if err != nil {
		t.Fatal(err)
	}
	assertNoDubboLinks(t, filepath.Join(home, "groups", "dubbo-ambiguous-links.json"))
	for _, pass := range result.Results {
		if pass.Pass == MethodDubbo && (pass.Candidates != 2 || pass.Skipped != 2) {
			t.Fatalf("ambiguous telemetry = %+v, want 2 candidates and 2 skipped", pass)
		}
	}
}

func assertNoDubboLinks(t *testing.T, path string) {
	t.Helper()
	doc, err := readDoc(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, link := range doc.Links {
		if link.Method == MethodDubbo {
			t.Fatalf("unexpected Dubbo link: %+v", link)
		}
	}
}

func onlyDubboLink(t *testing.T, doc *Document) Link {
	t.Helper()
	var links []Link
	for _, link := range doc.Links {
		if link.Method == MethodDubbo {
			links = append(links, link)
		}
	}
	if len(links) != 1 {
		t.Fatalf("Dubbo links = %d, want 1: %+v", len(links), links)
	}
	return links[0]
}

func dubboFixtureEntity(id, kind, subtype, role, iface, group, version, protocol, source string) map[string]any {
	return map[string]any{"id": id, "name": "dubbo:" + role + ":" + iface, "kind": kind, "subtype": subtype, "source_file": source,
		"properties": map[string]any{"rpc_framework": "dubbo", "rpc_role": role, "interface_fqn": iface, "group": group, "group_resolved": "true", "version": version, "version_resolved": "true", "protocol": protocol}}
}

func dubboFixtureMethod(id, role, iface, group, version, method, arity, source string) map[string]any {
	entity := dubboFixtureEntity(id, "SCOPE.Operation", "rpc_"+role+"_method", role, iface, group, version, "", source)
	entity["properties"].(map[string]any)["rpc_method"] = method
	entity["properties"].(map[string]any)["rpc_arity"] = arity
	return entity
}
