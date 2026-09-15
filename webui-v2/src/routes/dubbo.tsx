import { useMemo, useState } from "react";
import { ArrowLeft, ArrowRight, Boxes, GitBranch, Network, RotateCcw, Search } from "lucide-react";
import { Link, useParams } from "react-router-dom";

import { Badge, Button, Card, CardBody, CardHeader, CardTitle, Input, useSetInsight } from "@/components/ui";
import type { InsightValue } from "@/components/ui";
import { Skeleton } from "@/components/ui/skeleton";
import type { DubboEndpoint, DubboFilters, DubboService } from "@/data/types";
import { useDubboReport } from "@/hooks/use-dubbo";
import { buildDubboCallMap } from "@/lib/dubbo-call-map";

const EMPTY_FILTERS: DubboFilters = { page: 1, page_size: 25 };

const DUBBO_INSIGHT: InsightValue = {
  storageKey: "dubbo",
  human: (
    <>
      This view shows exact Dubbo consumers and providers grouped by interface
      contract. It requests only Dubbo links from the daemon, so large groups do
      not need to load the generic code graph or every cross-repository link.
    </>
  ),
  agent: {
    tool: "grafel_cross_links",
    example: "Find every consumer and provider of DesControlCodeService before changing its Dubbo contract.",
  },
};

export default function DubboScreen() {
  useSetInsight(DUBBO_INSIGHT);
  const { groupId = "" } = useParams<{ groupId: string }>();
  const [draftSearch, setDraftSearch] = useState("");
  const [filters, setFilters] = useState<DubboFilters>(EMPTY_FILTERS);
  const [selected, setSelected] = useState<DubboService | null>(null);
  const { data, isLoading, isFetching, isError } = useDubboReport(groupId, filters);

  const applyFilters = (next: Partial<DubboFilters>) => {
    setSelected(null);
    setFilters((current) => ({ ...current, ...next, page: 1 }));
  };

  const reset = () => {
    setDraftSearch("");
    setSelected(null);
    setFilters(EMPTY_FILTERS);
  };

  return (
    <div className="flex h-full min-h-0 flex-col bg-bg">
      <div className="flex-1 min-h-0 overflow-y-auto ag-scroll px-4 py-4 space-y-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 className="text-xl font-semibold text-text">Dubbo call map</h1>
            <p className="mt-1 text-sm text-text-3">
              Exact consumer → interface → provider relationships only. Generic graph edges are not loaded.
            </p>
          </div>
          {isFetching && !isLoading ? <Badge>Refreshing</Badge> : null}
        </div>

        <FilterBar
          draftSearch={draftSearch}
          setDraftSearch={setDraftSearch}
          filters={filters}
          facets={data?.facets}
          applyFilters={applyFilters}
          onSearch={() => applyFilters({ q: draftSearch.trim() })}
          onReset={reset}
        />

        {isLoading ? (
          <LoadingState />
        ) : isError || !data ? (
          <Card><CardBody className="text-danger">Unable to load Dubbo relationships.</CardBody></Card>
        ) : (
          <>
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
              <SummaryCard label="Services" value={data.summary.services} />
              <SummaryCard label="Consumers" value={data.summary.consumers} />
              <SummaryCard label="Providers" value={data.summary.providers} />
              <SummaryCard label="Repositories" value={data.summary.repos} />
              <SummaryCard label="Exact links" value={data.summary.links} />
            </div>

            {selected ? (
              <FocusedCallMap groupId={groupId} service={selected} onBack={() => setSelected(null)} />
            ) : (
              <ServiceList services={data.services} groupId={groupId} onSelect={setSelected} />
            )}

            {!selected ? (
              <Pagination
                page={data.page}
                totalPages={data.total_pages}
                totalServices={data.total_services}
                onPage={(page) => setFilters((current) => ({ ...current, page }))}
              />
            ) : null}
          </>
        )}
      </div>
    </div>
  );
}

function FilterBar({
  draftSearch,
  setDraftSearch,
  filters,
  facets,
  applyFilters,
  onSearch,
  onReset,
}: {
  draftSearch: string;
  setDraftSearch: (value: string) => void;
  filters: DubboFilters;
  facets?: { consumer_repos: string[]; provider_repos: string[]; groups: string[]; versions: string[]; protocols: string[] };
  applyFilters: (next: Partial<DubboFilters>) => void;
  onSearch: () => void;
  onReset: () => void;
}) {
  return (
    <Card>
      <CardBody className="space-y-3">
        <form className="flex gap-2" onSubmit={(event) => { event.preventDefault(); onSearch(); }}>
          <Input value={draftSearch} onChange={(event) => setDraftSearch(event.target.value)} placeholder="Search interface, e.g. DesControlCodeService" />
          <Button type="submit"><Search size={14} />Search</Button>
          <Button type="button" variant="secondary" onClick={onReset}><RotateCcw size={14} />Reset</Button>
        </form>
        <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-5">
          <FilterSelect label="Consumer repo" value={filters.consumer_repo} options={facets?.consumer_repos} onChange={(value) => applyFilters({ consumer_repo: value })} />
          <FilterSelect label="Provider repo" value={filters.provider_repo} options={facets?.provider_repos} onChange={(value) => applyFilters({ provider_repo: value })} />
          <FilterSelect label="Dubbo group" value={filters.dubbo_group} options={facets?.groups} onChange={(value) => applyFilters({ dubbo_group: value })} />
          <FilterSelect label="Version" value={filters.version} options={facets?.versions} onChange={(value) => applyFilters({ version: value })} />
          <FilterSelect label="Protocol" value={filters.protocol} options={facets?.protocols} onChange={(value) => applyFilters({ protocol: value })} />
        </div>
      </CardBody>
    </Card>
  );
}

function FilterSelect({ label, value, options = [], onChange }: { label: string; value?: string; options?: string[]; onChange: (value: string) => void }) {
  return (
    <label className="space-y-1 text-xs text-text-3">
      <span>{label}</span>
      <select className="h-8 w-full rounded-md border border-border bg-surface px-2 text-sm text-text" value={value ?? ""} onChange={(event) => onChange(event.target.value)}>
        <option value="">All</option>
        {options.map((option) => <option key={option} value={option}>{option}</option>)}
      </select>
    </label>
  );
}

function ServiceList({ services, groupId, onSelect }: { services: DubboService[]; groupId: string; onSelect: (service: DubboService) => void }) {
  if (services.length === 0) {
    return <Card><CardBody className="text-sm text-text-3">No exact Dubbo services match the current filters.</CardBody></Card>;
  }
  return (
    <div className="space-y-3">
      {services.map((service) => (
        <Card key={`${service.interface}|${service.group}|${service.version}|${service.protocol}`}>
          <CardHeader className="flex-row items-center justify-between gap-3">
            <div className="min-w-0">
              <CardTitle className="truncate">{service.simple_name}</CardTitle>
              <div className="mt-1 truncate font-mono text-xs text-text-3">{service.interface}</div>
            </div>
            <Button variant="secondary" onClick={() => onSelect(service)}><Network size={14} />View call map</Button>
          </CardHeader>
          <CardBody className="space-y-3">
            <ContractBadges service={service} />
            <div className="grid items-stretch gap-2 lg:grid-cols-[1fr_auto_1fr_auto_1fr]">
              <EndpointSummary title="Consumers" endpoints={service.consumers} groupId={groupId} />
              <ArrowRight className="hidden self-center text-text-4 lg:block" size={18} />
              <div className="flex min-h-20 items-center justify-center rounded-lg border border-accent/30 bg-accent/10 px-3 text-center">
                <div><Boxes className="mx-auto mb-1 text-accent" size={18} /><div className="font-medium text-text">{service.simple_name}</div><div className="text-xs text-text-3">{service.link_count} exact link{service.link_count === 1 ? "" : "s"}</div></div>
              </div>
              <ArrowRight className="hidden self-center text-text-4 lg:block" size={18} />
              <EndpointSummary title="Providers" endpoints={service.providers} groupId={groupId} />
            </div>
          </CardBody>
        </Card>
      ))}
    </div>
  );
}

function FocusedCallMap({ groupId, service, onBack }: { groupId: string; service: DubboService; onBack: () => void }) {
  const callMap = useMemo(() => buildDubboCallMap(service), [service]);
  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between gap-3">
        <div>
          <Button variant="ghost" size="sm" onClick={onBack}><ArrowLeft size={14} />All services</Button>
          <CardTitle className="mt-2">{service.simple_name}</CardTitle>
          <div className="mt-1 font-mono text-xs text-text-3">{service.interface}</div>
        </div>
        <ContractBadges service={service} />
      </CardHeader>
      <CardBody>
        <div className="grid gap-4 xl:grid-cols-[minmax(240px,1fr)_80px_minmax(280px,1.15fr)_80px_minmax(240px,1fr)]">
          <EndpointColumn title="Consumers" subtitle="Where the remote service is injected or declared" endpoints={callMap.consumers} groupId={groupId} tone="consumer" />
          <FlowArrow label="invokes" />
          <div className="flex min-h-52 items-center justify-center rounded-xl border-2 border-accent/40 bg-accent/10 p-5 text-center">
            <div className="min-w-0"><Boxes className="mx-auto mb-3 text-accent" size={30} /><div className="text-lg font-semibold text-text">{service.simple_name}</div><div className="mt-2 break-all font-mono text-xs text-text-3">{service.interface}</div><div className="mt-4"><ContractBadges service={service} /></div></div>
          </div>
          <FlowArrow label="implemented by" />
          <EndpointColumn title="Providers" subtitle="Where the service implementation is exported" endpoints={callMap.providers} groupId={groupId} tone="provider" />
        </div>
      </CardBody>
    </Card>
  );
}

function FlowArrow({ label }: { label: string }) {
  return <div className="hidden items-center justify-center xl:flex"><div className="text-center text-xs text-text-4"><ArrowRight className="mx-auto mb-1" size={22} />{label}</div></div>;
}

function EndpointColumn({ title, subtitle, endpoints, groupId, tone }: { title: string; subtitle: string; endpoints: DubboEndpoint[]; groupId: string; tone: "consumer" | "provider" }) {
  return (
    <div className="space-y-2">
      <div><h3 className="font-semibold text-text">{title} <span className="text-text-4">{endpoints.length}</span></h3><p className="text-xs text-text-3">{subtitle}</p></div>
      {endpoints.map((endpoint) => <EndpointCard key={endpoint.id} endpoint={endpoint} groupId={groupId} tone={tone} />)}
    </div>
  );
}

function EndpointSummary({ title, endpoints, groupId }: { title: string; endpoints: DubboEndpoint[]; groupId: string }) {
  return (
    <div className="rounded-lg border border-border-soft bg-surface-2 p-3">
      <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-text-3">{title} · {endpoints.length}</div>
      <div className="space-y-1.5">{endpoints.slice(0, 3).map((endpoint) => <EndpointLink key={endpoint.id} endpoint={endpoint} groupId={groupId} />)}{endpoints.length > 3 ? <div className="text-xs text-text-4">+{endpoints.length - 3} more</div> : null}</div>
    </div>
  );
}

function EndpointCard({ endpoint, groupId, tone }: { endpoint: DubboEndpoint; groupId: string; tone: "consumer" | "provider" }) {
  return (
    <div className={`rounded-lg border p-3 ${tone === "consumer" ? "border-info/30 bg-info/5" : "border-success/30 bg-success/5"}`}>
      <div className="flex items-center gap-2"><GitBranch size={14} className="text-text-3" /><Badge>{endpoint.repo || "unknown repo"}</Badge></div>
      <div className="mt-2 break-all text-sm font-medium text-text">{endpoint.name || endpoint.qualified_name || endpoint.id}</div>
      {endpoint.file ? <div className="mt-1 break-all font-mono text-xs text-text-3">{endpoint.file}{endpoint.line ? `:${endpoint.line}` : ""}</div> : null}
      <Link className="mt-2 inline-flex text-xs text-accent hover:underline" to={`/g/${encodeURIComponent(groupId)}/graph?node=${encodeURIComponent(endpoint.id)}`}>Open in graph</Link>
    </div>
  );
}

function EndpointLink({ endpoint, groupId }: { endpoint: DubboEndpoint; groupId: string }) {
  return <Link className="block truncate text-sm text-text hover:text-accent" title={endpoint.id} to={`/g/${encodeURIComponent(groupId)}/graph?node=${encodeURIComponent(endpoint.id)}`}><span className="mr-2 text-xs text-text-4">{endpoint.repo}</span>{endpoint.name || endpoint.qualified_name || endpoint.id}</Link>;
}

function ContractBadges({ service }: { service: DubboService }) {
  return <div className="flex flex-wrap gap-1.5">{service.group ? <Badge>group: {service.group}</Badge> : null}{service.version ? <Badge>version: {service.version}</Badge> : null}{service.protocol ? <Badge>protocol: {service.protocol}</Badge> : null}{service.confidence === 1 ? <Badge>exact</Badge> : null}</div>;
}

function SummaryCard({ label, value }: { label: string; value: number }) {
  return <Card><CardBody><div className="text-2xl font-semibold text-text">{value.toLocaleString()}</div><div className="mt-1 text-xs uppercase tracking-wide text-text-3">{label}</div></CardBody></Card>;
}

function Pagination({ page, totalPages, totalServices, onPage }: { page: number; totalPages: number; totalServices: number; onPage: (page: number) => void }) {
  return <div className="flex items-center justify-between"><div className="text-sm text-text-3">{totalServices} matching services</div><div className="flex items-center gap-2"><Button variant="secondary" disabled={page <= 1} onClick={() => onPage(page - 1)}><ArrowLeft size={14} />Previous</Button><span className="text-sm text-text-3">Page {page} of {Math.max(totalPages, 1)}</span><Button variant="secondary" disabled={page >= totalPages} onClick={() => onPage(page + 1)}>Next<ArrowRight size={14} /></Button></div></div>;
}

function LoadingState() {
  return <div className="space-y-3"><div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">{Array.from({ length: 5 }, (_, index) => <Skeleton key={index} className="h-24" />)}</div>{Array.from({ length: 3 }, (_, index) => <Skeleton key={index} className="h-48" />)}</div>;
}
