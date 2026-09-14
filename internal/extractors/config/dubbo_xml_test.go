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
