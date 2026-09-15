package java

import (
	"strings"
	"testing"
)

// The Dubbo annotation regexes also accept the bare `@Service` and `@Reference`
// spellings, which Spring and OSGi use for entirely unrelated purposes. The only
// thing keeping a plain Spring bean out of the RPC graph is the
// hasDubboAnnotationImport check at the top of each annotation pass. These two
// tests observe that check at each of its two call sites independently: one
// neutralised site must fail exactly one of them.
//
// Each test carries a positive control — the same fixture with the Dubbo import
// swapped in — so it cannot pass vacuously if the fixture later stops matching
// the annotation regex at all.

const springServiceProviderSource = `
package example;

import com.example.api.Order;
import com.example.api.OrderFacade;
import org.springframework.stereotype.Service;

@Service
class OrderFacadeImpl implements OrderFacade {
    public void submit(Order order) {}
}
`

const osgiReferenceConsumerSource = `
package example;

import com.example.api.Order;
import com.example.api.OrderFacade;
import org.osgi.service.component.annotations.Component;
import org.osgi.service.component.annotations.Reference;

@Component
public class OrderConsumer {
    @Reference
    private OrderFacade orderFacade;

    public void submit(Order order) {
        orderFacade.submit(order);
    }
}
`

func TestDubboProviderAnnotationRequiresDubboImport(t *testing.T) {
	assertNoDubboAnnotationImport(t, springServiceProviderSource)
	if !strings.Contains(springServiceProviderSource, "@Service") {
		t.Fatalf("fixture no longer carries the @Service annotation under test")
	}
	if !strings.Contains(springServiceProviderSource, "implements OrderFacade") {
		t.Fatalf("fixture must implement an interface, or the provider pass has nothing to resolve")
	}
	if !strings.Contains(springServiceProviderSource, "org.springframework.stereotype.Service") {
		t.Fatalf("fixture must import Spring's @Service, not some other annotation")
	}

	// Positive control: the identical class, with Dubbo's @Service import, is a
	// provider. This proves the negative assertion below is not vacuous.
	control := strings.Replace(
		springServiceProviderSource,
		"import org.springframework.stereotype.Service;",
		"import org.apache.dubbo.config.annotation.Service;",
		1,
	)
	if got := countDubboRole(extractDubboSource(control, "OrderFacadeImpl.java"), "provider"); got == 0 {
		t.Fatalf("control: a Dubbo-imported @Service class must yield provider entities, got none")
	}

	result := extractDubboSource(springServiceProviderSource, "OrderFacadeImpl.java")
	if got := countDubboRole(result, "provider"); got != 0 {
		t.Fatalf("Spring @Service without a Dubbo import must yield no provider entities, got %d: %+v", got, result.Entities)
	}
}

func TestDubboReferenceAnnotationRequiresDubboImport(t *testing.T) {
	assertNoDubboAnnotationImport(t, osgiReferenceConsumerSource)
	if !strings.Contains(osgiReferenceConsumerSource, "@Reference") {
		t.Fatalf("fixture no longer carries the @Reference annotation under test")
	}
	if !strings.Contains(osgiReferenceConsumerSource, "org.osgi.service.component.annotations.Reference") {
		t.Fatalf("fixture must import OSGi's @Reference, not some other annotation")
	}

	// Positive control: the identical field, with Dubbo's @Reference import, is a
	// consumer. This proves the negative assertion below is not vacuous.
	control := strings.Replace(
		osgiReferenceConsumerSource,
		"import org.osgi.service.component.annotations.Reference;",
		"import org.apache.dubbo.config.annotation.Reference;",
		1,
	)
	if got := countDubboRole(extractDubboSource(control, "OrderConsumer.java"), "consumer"); got == 0 {
		t.Fatalf("control: a Dubbo-imported @Reference field must yield consumer entities, got none")
	}

	result := extractDubboSource(osgiReferenceConsumerSource, "OrderConsumer.java")
	if got := countDubboRole(result, "consumer"); got != 0 {
		t.Fatalf("OSGi @Reference without a Dubbo import must yield no consumer entities, got %d: %+v", got, result.Entities)
	}
}

func assertNoDubboAnnotationImport(t *testing.T, source string) {
	t.Helper()
	for _, prefix := range []string{"org.apache.dubbo.config.annotation", "com.alibaba.dubbo.config.annotation"} {
		if strings.Contains(source, prefix) {
			t.Fatalf("fixture must not import %s, or it no longer exercises the import gate", prefix)
		}
	}
}

func extractDubboSource(source, filePath string) PatternResult {
	return ExtractDubbo(PatternContext{Source: source, Language: "java", Framework: "dubbo", FilePath: filePath})
}

func countDubboRole(result PatternResult, role string) int {
	count := 0
	for _, entity := range result.Entities {
		if entity.Properties["rpc_role"] == role {
			count++
		}
	}
	return count
}
