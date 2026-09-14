package java

import (
	"context"
	"testing"

	extreg "github.com/cajasmota/grafel/internal/extractor"
)

func TestDubboAnnotationContracts(t *testing.T) {
	source := `
package example;

import com.example.api.OrderFacade;
import org.apache.dubbo.config.annotation.DubboReference;
import org.apache.dubbo.config.annotation.DubboService;

public class OrderClient {
    @DubboReference(group = "orders", version = "1.2.0", protocol = "dubbo")
    private OrderFacade orderFacade;

    public void submit(Order order) {
        orderFacade.submit(order);
    }
}

@DubboService(group = "orders", version = "1.2.0")
@RequiredArgsConstructor
@Slf4j
class OrderFacadeImpl implements OrderFacade {
    public void submit(Order order) {}
}
`
	result := ExtractDubbo(PatternContext{Source: source, Language: "java", Framework: "dubbo", FilePath: "OrderClient.java"})
	consumer := findDubboEntity(result, "consumer")
	provider := findDubboEntity(result, "provider")
	assertDubboProperty(t, consumer, "interface_fqn", "com.example.api.OrderFacade")
	assertDubboProperty(t, consumer, "group", "orders")
	assertDubboProperty(t, consumer, "version", "1.2.0")
	assertDubboProperty(t, consumer, "protocol", "dubbo")
	assertDubboProperty(t, consumer, "source_style", "annotation")
	assertDubboProperty(t, provider, "interface_fqn", "com.example.api.OrderFacade")
	assertDubboProperty(t, provider, "implementation", "OrderFacadeImpl")
	if countDubboMethods(result, "consumer") != 1 || countDubboMethods(result, "provider") != 1 {
		t.Fatalf("expected matched consumer/provider method entities: %+v", result.Entities)
	}
}

func countDubboMethods(result PatternResult, role string) int {
	count := 0
	for _, entity := range result.Entities {
		if entity.Properties["rpc_role"] == role && entity.Properties["rpc_method"] != nil {
			count++
		}
	}
	return count
}

func TestDubboLegacyAnnotationsRequireDubboImports(t *testing.T) {
	dubboSource := `
import com.example.api.CustomerFacade;
import com.alibaba.dubbo.config.annotation.Reference;
import com.alibaba.dubbo.config.annotation.Service;

class Client {
    @Reference(group = "customer")
    private CustomerFacade customerFacade;
}

@Service(group = "customer")
class CustomerFacadeImpl implements CustomerFacade {}
`
	result := ExtractDubbo(PatternContext{Source: dubboSource, Language: "java", Framework: "dubbo", FilePath: "Legacy.java"})
	if len(result.Entities) != 2 {
		t.Fatalf("expected two legacy Dubbo contracts, got %d", len(result.Entities))
	}

	springSource := `
import org.springframework.stereotype.Service;

@Service
class LocalService {}
`
	result = ExtractDubbo(PatternContext{Source: springSource, Language: "java", Framework: "dubbo", FilePath: "LocalService.java"})
	if len(result.Entities) != 0 {
		t.Fatalf("Spring @Service must not be extracted as Dubbo, got %+v", result.Entities)
	}
}

func TestDubboReferenceAndServiceBeans(t *testing.T) {
	source := `
import com.example.api.RiskFacade;
import org.apache.dubbo.config.spring.ReferenceBean;
import org.apache.dubbo.config.spring.ServiceBean;

class DubboConfig {
    // ReferenceBean<UnusedFacade> unusedFacade() {
    //     return new ReferenceBean<>();
    // }

    ReferenceBean<RiskFacade> riskFacadeReference() {
        ReferenceBean<RiskFacade> reference = new ReferenceBean<>();
        reference.setInterface(RiskFacade.class);
        reference.setGroup("risk");
        reference.setVersion("2.0.0");
        return reference;
    }

    ServiceBean<RiskFacade> riskFacadeService() {
        ServiceBean<RiskFacade> service = new ServiceBean<>();
        service.setInterface(RiskFacade.class);
        service.setGroup(group);
        service.setRef(riskFacadeImpl);
        return service;
    }
}
`
	result := ExtractDubbo(PatternContext{Source: source, Language: "java", Framework: "dubbo", FilePath: "DubboConfig.java"})
	if len(result.Entities) != 2 {
		t.Fatalf("expected two active bean contracts, got %d", len(result.Entities))
	}
	consumer := findDubboEntity(result, "consumer")
	provider := findDubboEntity(result, "provider")
	assertDubboProperty(t, consumer, "interface_fqn", "com.example.api.RiskFacade")
	assertDubboProperty(t, consumer, "group", "risk")
	assertDubboProperty(t, consumer, "group_resolved", true)
	assertDubboProperty(t, consumer, "source_style", "java_config")
	assertDubboProperty(t, provider, "group", "group")
	assertDubboProperty(t, provider, "group_resolved", false)
	assertDubboProperty(t, provider, "implementation", "riskFacadeImpl")
}

func TestDubboServiceResolvesSamePackageInterface(t *testing.T) {
	source := `
package com.example.risk;

import org.apache.dubbo.config.annotation.DubboService;

@DubboService
class RiskFacadeImpl implements RiskFacade {}
`
	result := ExtractDubbo(PatternContext{Source: source, Language: "java", Framework: "dubbo", FilePath: "RiskFacadeImpl.java"})
	provider := findDubboEntity(result, "provider")
	assertDubboProperty(t, provider, "interface_fqn", "com.example.risk.RiskFacade")
}

func TestDubboBeanPreservesWildcardImportCandidates(t *testing.T) {
	source := `
package com.example.config;

import com.example.orders.*;
import com.example.risk.*;
import org.apache.dubbo.config.spring.ServiceBean;

class DubboConfig {
    ServiceBean<OrderFacade> orderFacadeService(OrderFacade orderFacade) {
        return new ServiceBean<>();
    }
}
`
	result := ExtractDubbo(PatternContext{Source: source, Language: "java", Framework: "dubbo", FilePath: "DubboConfig.java"})
	provider := findDubboEntity(result, "provider")
	assertDubboProperty(t, provider, "interface_fqn", "OrderFacade")
	assertDubboProperty(t, provider, "interface_fqn_resolved", false)
	assertDubboProperty(t, provider, "interface_candidates", "com.example.config.OrderFacade,com.example.orders.OrderFacade,com.example.risk.OrderFacade")
}

func TestDubboRegisteredDispatcher(t *testing.T) {
	source := `
import com.example.api.PaymentFacade;
import org.apache.dubbo.config.annotation.DubboReference;

class PaymentClient {
    @DubboReference(version = "1.0.0")
    private PaymentFacade paymentFacade;
}
`
	extractor, ok := extreg.Get("custom_java_patterns")
	if !ok {
		t.Fatal("custom_java_patterns not registered")
	}
	entities, err := extractor.Extract(context.Background(), extreg.FileInput{
		Path: "PaymentClient.java", Language: "java", Content: []byte(source),
	})
	if err != nil {
		t.Fatalf("extract registered Dubbo pattern: %v", err)
	}
	for _, entity := range entities {
		if entity.Properties["rpc_framework"] == "dubbo" {
			if entity.Properties["interface_fqn"] != "com.example.api.PaymentFacade" {
				t.Fatalf("interface_fqn = %q", entity.Properties["interface_fqn"])
			}
			return
		}
	}
	t.Fatalf("registered dispatcher did not emit Dubbo entity: %+v", entities)
}

func findDubboEntity(result PatternResult, role string) *SecondaryEntity {
	for index := range result.Entities {
		if result.Entities[index].Properties["rpc_role"] == role {
			return &result.Entities[index]
		}
	}
	return nil
}

func assertDubboProperty(t *testing.T, entity *SecondaryEntity, key string, want any) {
	t.Helper()
	if entity == nil {
		t.Fatalf("missing Dubbo entity while checking %s", key)
	}
	if got := entity.Properties[key]; got != want {
		t.Fatalf("property %s = %#v, want %#v", key, got, want)
	}
}
