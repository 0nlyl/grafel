import type { DubboEndpoint, DubboService } from "@/data/types";

export interface DubboCallMap {
  consumers: DubboEndpoint[];
  service: Pick<DubboService, "interface" | "simple_name" | "group" | "version" | "protocol">;
  providers: DubboEndpoint[];
}

function uniqueEndpoints(endpoints: DubboEndpoint[]): DubboEndpoint[] {
  const byId = new Map<string, DubboEndpoint>();
  for (const endpoint of endpoints) byId.set(endpoint.id, endpoint);
  return [...byId.values()].sort(
    (left, right) => left.repo.localeCompare(right.repo) || left.id.localeCompare(right.id),
  );
}

export function buildDubboCallMap(service: DubboService): DubboCallMap {
  return {
    consumers: uniqueEndpoints(service.consumers),
    service: {
      interface: service.interface,
      simple_name: service.simple_name,
      group: service.group,
      version: service.version,
      protocol: service.protocol,
    },
    providers: uniqueEndpoints(service.providers),
  };
}
