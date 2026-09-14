import { keepPreviousData, useQuery } from "@tanstack/react-query";

import type { DubboFilters } from "@/data/types";
import { api } from "@/lib/api";

export function useDubboReport(groupId: string, filters: DubboFilters) {
  return useQuery({
    queryKey: ["dubbo", groupId, filters] as const,
    queryFn: () => api.getDubboReport(groupId, filters),
    enabled: groupId.length > 0,
    placeholderData: keepPreviousData,
    staleTime: 30_000,
  });
}
