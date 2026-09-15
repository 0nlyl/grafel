# Dubbo Call Map Design

## Goal

Provide a dedicated, understandable, and responsive Dubbo screen for multi-repository groups without loading the full cross-repo link set or the full code graph.

## User Experience

- Add a `Dubbo` item to the group navigation at `/g/:groupId/dubbo`.
- Show summary counts for services, consumers, providers, repositories, and exact links.
- Default to a service-centric list grouped by Dubbo interface contract.
- Each service card shows `consumer repository → consumer declaration → interface → provider declaration → provider repository`.
- Provide filters for free-text interface search, consumer repository, provider repository, group, version, and protocol.
- Provide paginated list and focused call-map modes. The call map renders only the selected service and never the entire group graph.
- Show source file, line, declaration name, contract attributes, and match confidence for each endpoint.

## Backend

- Preserve `identifier` and `properties` from the persisted cross-repo link records in `CrossRepoLink`.
- Add `GET /api/v2/groups/{group}/dubbo`.
- Select only links where `method=dubbo` or `channel=dubbo`.
- Parse exact contract identifiers in the form `dubbo:<interface>|<group>|<version>|<protocol>` and method identifiers with optional method and arity suffixes.
- Group records by interface, group, version, and protocol before responding.
- Apply all filters and pagination before enriching endpoints so large groups do not pay the cost of enriching unrelated links.
- Return deterministic ordering and filter facets derived from the complete Dubbo subset.
- Reject invalid page sizes and cap page size at 100.

## Performance

- The browser must not download all cross-repo links for the Dubbo screen.
- The initial response defaults to 25 service groups.
- The focused call map is bounded to endpoints belonging to one service group.
- Filtering and pagination execute on the daemon.

## Compatibility

- Existing `/api/groups/{group}/links`, Graph, Links, Topology, and MCP behavior remain unchanged.
- Older non-Dubbo link records continue to decode normally.
- Records without a valid fully-qualified Dubbo contract are excluded from the dedicated exact-call screen.

## Validation

- Go tests cover link decoding, contract parsing, filtering, grouping, facets, and pagination.
- React tests cover response-to-view-model grouping and call-map layout.
- Frontend type checking, unit tests, Vite build, Go tests, vet, coverage gate, and quality ratchet run before deployment.
- The rebuilt binary is installed locally and the real `ossac-platform` page is verified in a browser.
