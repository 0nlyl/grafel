<!-- DO NOT EDIT — generated from docs/coverage/registry.json by 'go run ./tools/coverage gen' -->
# `lang.java.framework.dubbo` — Apache Dubbo

Auto-generated. Back to [summary](../summary.md).

- **Language:** [java](../by-language/java.md)
- **Category:** [http_framework](../by-category/http_framework.md)
- **Subcategory:** RPC Framework
- **Capability cells:** 54

## Capabilities


### Schema

| Capability | Status | Verified at | Issue | Cites | Notes |
|------------|--------|-------------|-------|-------|-------|
| Federation extraction | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Procedure extraction | 🟢 `partial` | `2026-09-14` | backfill:dictionary-completeness | `internal/custom/java/dubbo.go`<br>`internal/custom/java/dubbo_test.go`<br>`internal/links/dubbo_pass.go`<br>`internal/links/dubbo_pass_test.go` | Extracts method-level Dubbo consumer call sites from @DubboReference/@Reference fields and provider implementation methods from @DubboService/legacy @Service classes. Canonical interface, method, and arity properties allow the Dubbo pass to link uniquely matched client operations to provider operations. Programmatic and XML contracts remain contract-level when the call site or implementation class is not statically attributable. |
| Schema extraction | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Type graph extraction | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |

### Codegen

| Capability | Status | Verified at | Issue | Cites | Notes |
|------------|--------|-------------|-------|-------|-------|
| Client codegen | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |

### Transport

| Capability | Status | Verified at | Issue | Cites | Notes |
|------------|--------|-------------|-------|-------|-------|
| Transport binding | 🟢 `partial` | `2026-09-14` | — | `internal/custom/java/dubbo.go`<br>`internal/extractors/config/dubbo_xml.go`<br>`internal/links/dubbo_pass.go`<br>`internal/links/dubbo_pass_test.go` | Extracts Apache Dubbo and legacy Alibaba Dubbo consumer/provider contracts from Java annotations, ReferenceBean<T>/ServiceBean<T>, and Spring XML <dubbo:reference>/<dubbo:service>. Wildcard Java imports retain candidate interface FQNs and resolve only when the indexed type graph identifies one candidate. The cross-repository pass prefers exact interface/group/version/protocol matches, then emits a lower-confidence inferred link only when unresolved or omitted optional metadata has no known conflict and identifies one external provider. |

### Routing

| Capability | Status | Verified at | Issue | Cites | Notes |
|------------|--------|-------------|-------|-------|-------|
| Endpoint deprecation versioning | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Endpoint pagination posture | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Endpoint response codes | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Endpoint synthesis | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Handler attribution | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Route extraction | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |

### View

| Capability | Status | Verified at | Issue | Cites | Notes |
|------------|--------|-------------|-------|-------|-------|
| View rendering | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |

### Auth

| Capability | Status | Verified at | Issue | Cites | Notes |
|------------|--------|-------------|-------|-------|-------|
| Auth coverage | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |

### Validation

| Capability | Status | Verified at | Issue | Cites | Notes |
|------------|--------|-------------|-------|-------|-------|
| DTO extraction | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Request validation | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |

### Middleware

| Capability | Status | Verified at | Issue | Cites | Notes |
|------------|--------|-------------|-------|-------|-------|
| Middleware coverage | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Rate limit stamping | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |

### Type System

| Capability | Status | Verified at | Issue | Cites | Notes |
|------------|--------|-------------|-------|-------|-------|
| Enum extraction | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Interface extraction | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Type alias extraction | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Type extraction | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |

### DI

| Capability | Status | Verified at | Issue | Cites | Notes |
|------------|--------|-------------|-------|-------|-------|
| DI binding extraction | 🟢 `partial` | `2026-09-14` | backfill:dictionary-completeness | `internal/custom/java/dubbo.go`<br>`internal/extractors/config/dubbo_xml.go`<br>`internal/extractors/config/dubbo_xml_test.go` | Recognizes Dubbo ReferenceBean<T>/ServiceBean<T> Java factory methods and Spring XML reference/service id/ref bindings. Emits normalized rpc_client/rpc_service entities; it does not yet resolve every XML bean id or programmatic factory helper to an implementation class. |
| DI injection point | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| DI scope resolution | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |

### Testing

| Capability | Status | Verified at | Issue | Cites | Notes |
|------------|--------|-------------|-------|-------|-------|
| Tests linkage | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |

### Observability

| Capability | Status | Verified at | Issue | Cites | Notes |
|------------|--------|-------------|-------|-------|-------|
| Log extraction | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Metric extraction | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Trace extraction | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |

### Substrate

| Capability | Status | Verified at | Issue | Cites | Notes |
|------------|--------|-------------|-------|-------|-------|
| Confidence overlay | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Config consumption | 🟢 `partial` | `2026-09-14` | backfill:dictionary-completeness | `internal/custom/java/dubbo.go`<br>`internal/custom/java/dubbo_test.go`<br>`internal/extractors/config/dubbo_xml.go`<br>`internal/extractors/config/dubbo_xml_test.go`<br>`internal/links/dubbo_pass.go` | Preserves Dubbo group, version, and protocol values from annotations, Java setters, and XML attributes. Literal values are marked resolved; variables and ${...} placeholders remain explicitly unresolved and are excluded from automatic cross-repository linking. |
| Constant propagation | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| DB effect | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Dead code detection | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Def use chain extraction | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Env fallback recognition | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Error flow | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Feature flag gating | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Fs effect | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| HTTP effect | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Import resolution quality | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Module cycle detection | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Mutation effect | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Pure function tagging | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Reachability analysis | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Request shape extraction | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Request sink dataflow | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Response shape extraction | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Sanitizer recognition | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Schema drift detection | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Taint sink detection | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Taint source detection | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Template pattern catalog | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |
| Vulnerability finding | 🔴 `missing` | — | backfill:dictionary-completeness | — | — |

## Provenance

This record is sourced from `docs/coverage/registry.json`. To update it, edit the JSON
(or use `go run ./tools/coverage update lang.java.framework.dubbo ...`) then regenerate:

```
go run ./tools/coverage validate
go run ./tools/coverage gen
```
