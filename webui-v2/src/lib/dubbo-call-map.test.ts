import { describe, expect, it } from "vitest";

import type { DubboService } from "@/data/types";
import { buildDubboCallMap } from "./dubbo-call-map";

describe("buildDubboCallMap", () => {
  it("deduplicates and sorts endpoints around one service", () => {
    const service: DubboService = {
      interface: "com.example.DecisionService",
      simple_name: "DecisionService",
      consumers: [
        { id: "ffs::2", repo: "ffs" },
        { id: "aps::1", repo: "aps" },
        { id: "aps::1", repo: "aps" },
      ],
      providers: [
        { id: "des::2", repo: "des" },
        { id: "des::1", repo: "des" },
      ],
      link_count: 3,
    };

    const map = buildDubboCallMap(service);
    expect(map.consumers.map((endpoint) => endpoint.id)).toEqual(["aps::1", "ffs::2"]);
    expect(map.providers.map((endpoint) => endpoint.id)).toEqual(["des::1", "des::2"]);
    expect(map.service.interface).toBe("com.example.DecisionService");
  });
});
