package dashboard

import (
	"encoding/json"
	"testing"
)

func TestCrossRepoLinkUnmarshalPreservesDubboMetadata(t *testing.T) {
	raw := []byte(`{
		"source":"aps::consumer",
		"target":"des::provider",
		"relation":"calls",
		"method":"dubbo",
		"channel":"dubbo",
		"identifier":"dubbo:com.example.DecisionService|risk|1.0.0|tri",
		"properties":{"confidence":"resolved","match":"interface_group_version_protocol"}
	}`)

	var link CrossRepoLink
	if err := json.Unmarshal(raw, &link); err != nil {
		t.Fatal(err)
	}

	if link.Identifier != "dubbo:com.example.DecisionService|risk|1.0.0|tri" {
		t.Fatalf("Identifier = %q", link.Identifier)
	}
	if got := link.Properties["match"]; got != "interface_group_version_protocol" {
		t.Fatalf("Properties[match] = %q", got)
	}
}
