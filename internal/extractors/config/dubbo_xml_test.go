package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverDubboSpringXML(t *testing.T) {
	root := t.TempDir()
	rel := filepath.Join("src", "main", "resources", "spring", "services.xml")
	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `<beans xmlns:dubbo="http://dubbo.apache.org/schema/dubbo">
  <dubbo:reference id="orderFacade" interface="com.example.OrderFacade" group="orders" version="1.0.0"/>
  <dubbo:service interface="com.example.OrderFacade" ref="orderFacadeImpl" group="${dubbo.group}" version="1.0.0"/>
</beans>`
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	entities, _, err := Discover(context.Background(), root, []string{rel})
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]map[string]string{}
	for _, entity := range entities {
		if entity.Properties["rpc_framework"] == "dubbo" {
			roles[entity.Properties["rpc_role"]] = entity.Properties
		}
	}
	if roles["consumer"]["interface_fqn"] != "com.example.OrderFacade" {
		t.Fatalf("missing XML consumer: %+v", roles)
	}
	if roles["provider"]["implementation"] != "orderFacadeImpl" {
		t.Fatalf("missing XML provider implementation: %+v", roles)
	}
	if roles["provider"]["group_resolved"] != "false" {
		t.Fatalf("placeholder group must remain unresolved: %+v", roles["provider"])
	}
}

// The `<dubbo:` file gate says the document contains Dubbo somewhere, not that
// every <service>/<reference> element in it is Dubbo's. A mixed Spring context
// also carries Gemini Blueprint and CXF elements with the same local names and
// an `interface` attribute, and those must not become Dubbo contracts — two of
// them in different repositories would otherwise be linked to each other as a
// cross-repo Dubbo call.
func TestDiscoverDubboSpringXMLIgnoresForeignNamespaces(t *testing.T) {
	content := []byte(`<beans xmlns="http://www.springframework.org/schema/beans"
       xmlns:dubbo="http://dubbo.apache.org/schema/dubbo"
       xmlns:osgi="http://www.eclipse.org/gemini/blueprint/schema/blueprint"
       xmlns:jaxws="http://cxf.apache.org/jaxws">
  <dubbo:reference id="orderFacade" interface="com.example.api.OrderFacade"/>
  <osgi:service interface="com.other.NotDubbo" ref="notDubboBean"/>
  <osgi:reference id="alsoNotDubbo" interface="com.other.AlsoNot"/>
  <jaxws:service interface="com.other.Soap" ref="soapBean"/>
</beans>`)
	entities, _ := discoverDubboXML("/repo", "spring/app.xml", configSpec{"dubbo_spring_xml", formatXML}, content, "config")
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want only the <dubbo:reference>: %+v", len(entities), entities)
	}
	if entities[0].Properties["interface_fqn"] != "com.example.api.OrderFacade" {
		t.Fatalf("wrong contract kept: %+v", entities[0].Properties)
	}
}

// A `dubbo:` prefix with no matching xmlns declaration leaves encoding/xml with
// the literal prefix as the namespace; those elements still count.
func TestDiscoverDubboSpringXMLAcceptsUndeclaredDubboPrefix(t *testing.T) {
	content := []byte(`<beans>
  <dubbo:reference id="orderFacade" interface="com.example.api.OrderFacade"/>
</beans>`)
	entities, _ := discoverDubboXML("/repo", "spring/app.xml", configSpec{"dubbo_spring_xml", formatXML}, content, "config")
	if len(entities) != 1 {
		t.Fatalf("entities = %d, want 1: %+v", len(entities), entities)
	}
}
