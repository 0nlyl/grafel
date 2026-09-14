package dashboard

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultDubboPageSize = 25
	maxDubboPageSize     = 100
)

type dubboContract struct {
	Interface string
	Group     string
	Version   string
	Protocol  string
	Method    string
	Arity     int
	HasArity  bool
}

type dubboQuery struct {
	Search       string
	ConsumerRepo string
	ProviderRepo string
	Group        string
	Version      string
	Protocol     string
	Page         int
	PageSize     int
}

type dubboSummary struct {
	Services  int `json:"services"`
	Consumers int `json:"consumers"`
	Providers int `json:"providers"`
	Repos     int `json:"repos"`
	Links     int `json:"links"`
}

type dubboFacets struct {
	ConsumerRepos []string `json:"consumer_repos"`
	ProviderRepos []string `json:"provider_repos"`
	Groups        []string `json:"groups"`
	Versions      []string `json:"versions"`
	Protocols     []string `json:"protocols"`
}

type dubboEndpoint struct {
	ID            string `json:"id"`
	Repo          string `json:"repo"`
	Name          string `json:"name,omitempty"`
	QualifiedName string `json:"qualified_name,omitempty"`
	File          string `json:"file,omitempty"`
	Line          int    `json:"line,omitempty"`
	ModulePath    string `json:"module_path,omitempty"`
}

type dubboMethod struct {
	Name      string `json:"name"`
	Arity     int    `json:"arity,omitempty"`
	HasArity  bool   `json:"has_arity,omitempty"`
	LinkCount int    `json:"link_count"`
}

type dubboService struct {
	Interface  string          `json:"interface"`
	SimpleName string          `json:"simple_name"`
	Group      string          `json:"group,omitempty"`
	Version    string          `json:"version,omitempty"`
	Protocol   string          `json:"protocol,omitempty"`
	Consumers  []dubboEndpoint `json:"consumers"`
	Providers  []dubboEndpoint `json:"providers"`
	Methods    []dubboMethod   `json:"methods,omitempty"`
	LinkCount  int             `json:"link_count"`
	Confidence float64         `json:"confidence,omitempty"`
	Match      string          `json:"match,omitempty"`
}

type dubboReport struct {
	Summary       dubboSummary   `json:"summary"`
	Facets        dubboFacets    `json:"facets"`
	Services      []dubboService `json:"services"`
	Page          int            `json:"page"`
	PageSize      int            `json:"page_size"`
	TotalServices int            `json:"total_services"`
	TotalPages    int            `json:"total_pages"`
}

type dubboServiceGroup struct {
	contract dubboContract
	links    []CrossRepoLink
}

func (s *Server) handleV2Dubbo(w http.ResponseWriter, r *http.Request) {
	group := r.PathValue("group")
	if group == "" {
		writeErr(w, http.StatusBadRequest, "group is required")
		return
	}
	query, err := parseDubboQuery(r.URL.Query())
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	grp, err := s.graphs.GetGroup(group)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	var moduleRoots map[string][]string
	if repoRefs, repoErr := repoPathsForGroup(group); repoErr == nil {
		moduleRoots = moduleRootsByRepo(repoRefs)
	}
	writeJSON(w, http.StatusOK, buildDubboReport(grp, query, moduleRoots))
}

func parseDubboQuery(values url.Values) (dubboQuery, error) {
	query := dubboQuery{
		Search:       strings.TrimSpace(values.Get("q")),
		ConsumerRepo: strings.TrimSpace(values.Get("consumer_repo")),
		ProviderRepo: strings.TrimSpace(values.Get("provider_repo")),
		Group:        strings.TrimSpace(values.Get("dubbo_group")),
		Version:      strings.TrimSpace(values.Get("version")),
		Protocol:     strings.TrimSpace(values.Get("protocol")),
		Page:         1,
		PageSize:     defaultDubboPageSize,
	}
	if raw := strings.TrimSpace(values.Get("page")); raw != "" {
		page, err := strconv.Atoi(raw)
		if err != nil || page < 1 {
			return dubboQuery{}, fmt.Errorf("page must be a positive integer")
		}
		query.Page = page
	}
	if raw := strings.TrimSpace(values.Get("page_size")); raw != "" {
		pageSize, err := strconv.Atoi(raw)
		if err != nil || pageSize < 1 {
			return dubboQuery{}, fmt.Errorf("page_size must be a positive integer")
		}
		if pageSize > maxDubboPageSize {
			pageSize = maxDubboPageSize
		}
		query.PageSize = pageSize
	}
	return query, nil
}

func parseDubboContract(identifier string) (dubboContract, bool) {
	if !strings.HasPrefix(identifier, "dubbo:") {
		return dubboContract{}, false
	}
	parts := strings.Split(strings.TrimPrefix(identifier, "dubbo:"), "|")
	if len(parts) < 4 {
		return dubboContract{}, false
	}
	contract := dubboContract{
		Interface: strings.TrimSpace(parts[0]),
		Group:     strings.TrimSpace(parts[1]),
		Version:   strings.TrimSpace(parts[2]),
		Protocol:  strings.TrimSpace(parts[3]),
	}
	if !strings.Contains(contract.Interface, ".") {
		return dubboContract{}, false
	}
	if len(parts) >= 5 {
		contract.Method = strings.TrimSpace(parts[4])
	}
	if len(parts) >= 6 && strings.TrimSpace(parts[5]) != "" {
		arity, err := strconv.Atoi(strings.TrimSpace(parts[5]))
		if err != nil || arity < 0 {
			return dubboContract{}, false
		}
		contract.Arity = arity
		contract.HasArity = true
	}
	return contract, true
}

func buildDubboReport(grp *DashGroup, query dubboQuery, moduleRoots map[string][]string) dubboReport {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = defaultDubboPageSize
	}
	if query.PageSize > maxDubboPageSize {
		query.PageSize = maxDubboPageSize
	}

	allGroups := make(map[string]*dubboServiceGroup)
	filteredGroups := make(map[string]*dubboServiceGroup)
	consumerIDs := make(map[string]struct{})
	providerIDs := make(map[string]struct{})
	repos := make(map[string]struct{})
	consumerRepos := make(map[string]struct{})
	providerRepos := make(map[string]struct{})
	groups := make(map[string]struct{})
	versions := make(map[string]struct{})
	protocols := make(map[string]struct{})
	exactLinks := 0

	if grp != nil {
		for _, link := range grp.Links {
			if !strings.EqualFold(link.Method, "dubbo") && !strings.EqualFold(link.Channel, "dubbo") {
				continue
			}
			contract, ok := parseDubboContract(link.Identifier)
			if !ok {
				continue
			}
			exactLinks++
			key := dubboServiceKey(contract)
			addDubboGroupLink(allGroups, key, contract, link)
			consumerRepo := crossRepoSlug(link.Source)
			providerRepo := crossRepoSlug(link.Target)
			consumerIDs[link.Source] = struct{}{}
			providerIDs[link.Target] = struct{}{}
			addNonEmpty(repos, consumerRepo)
			addNonEmpty(repos, providerRepo)
			addNonEmpty(consumerRepos, consumerRepo)
			addNonEmpty(providerRepos, providerRepo)
			addNonEmpty(groups, contract.Group)
			addNonEmpty(versions, contract.Version)
			addNonEmpty(protocols, contract.Protocol)

			if dubboLinkMatches(query, contract, consumerRepo, providerRepo) {
				addDubboGroupLink(filteredGroups, key, contract, link)
			}
		}
	}

	keys := make([]string, 0, len(filteredGroups))
	for key := range filteredGroups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left := filteredGroups[keys[i]].contract
		right := filteredGroups[keys[j]].contract
		return dubboContractLess(left, right)
	})

	totalServices := len(keys)
	totalPages := 0
	if totalServices > 0 {
		totalPages = (totalServices + query.PageSize - 1) / query.PageSize
	}
	start := (query.Page - 1) * query.PageSize
	if start > totalServices {
		start = totalServices
	}
	end := start + query.PageSize
	if end > totalServices {
		end = totalServices
	}

	services := make([]dubboService, 0, end-start)
	for _, key := range keys[start:end] {
		services = append(services, renderDubboService(grp, filteredGroups[key], moduleRoots))
	}

	return dubboReport{
		Summary: dubboSummary{
			Services:  len(allGroups),
			Consumers: len(consumerIDs),
			Providers: len(providerIDs),
			Repos:     len(repos),
			Links:     exactLinks,
		},
		Facets: dubboFacets{
			ConsumerRepos: sortedSet(consumerRepos),
			ProviderRepos: sortedSet(providerRepos),
			Groups:        sortedSet(groups),
			Versions:      sortedSet(versions),
			Protocols:     sortedSet(protocols),
		},
		Services:      services,
		Page:          query.Page,
		PageSize:      query.PageSize,
		TotalServices: totalServices,
		TotalPages:    totalPages,
	}
}

func renderDubboService(grp *DashGroup, group *dubboServiceGroup, moduleRoots map[string][]string) dubboService {
	links := enrichLinkEndpoints(grp, group.links, moduleRoots)
	consumers := make(map[string]dubboEndpoint)
	providers := make(map[string]dubboEndpoint)
	methods := make(map[string]*dubboMethod)
	confidence := 0.0
	match := ""
	for _, link := range links {
		consumers[link.Source] = dubboSourceEndpoint(link)
		providers[link.Target] = dubboTargetEndpoint(link)
		if link.Confidence > confidence {
			confidence = link.Confidence
		}
		if match == "" {
			match = link.Properties["match"]
		}
		contract, ok := parseDubboContract(link.Identifier)
		if ok && contract.Method != "" {
			key := contract.Method + "|" + strconv.Itoa(contract.Arity) + "|" + strconv.FormatBool(contract.HasArity)
			method := methods[key]
			if method == nil {
				method = &dubboMethod{Name: contract.Method, Arity: contract.Arity, HasArity: contract.HasArity}
				methods[key] = method
			}
			method.LinkCount++
		}
	}
	return dubboService{
		Interface:  group.contract.Interface,
		SimpleName: dubboSimpleName(group.contract.Interface),
		Group:      group.contract.Group,
		Version:    group.contract.Version,
		Protocol:   group.contract.Protocol,
		Consumers:  sortedDubboEndpoints(consumers),
		Providers:  sortedDubboEndpoints(providers),
		Methods:    sortedDubboMethods(methods),
		LinkCount:  len(group.links),
		Confidence: confidence,
		Match:      match,
	}
}

func dubboLinkMatches(query dubboQuery, contract dubboContract, consumerRepo, providerRepo string) bool {
	if query.Search != "" && !strings.Contains(strings.ToLower(contract.Interface), strings.ToLower(query.Search)) {
		return false
	}
	return matchesOptional(query.ConsumerRepo, consumerRepo) &&
		matchesOptional(query.ProviderRepo, providerRepo) &&
		matchesOptional(query.Group, contract.Group) &&
		matchesOptional(query.Version, contract.Version) &&
		matchesOptional(query.Protocol, contract.Protocol)
}

func matchesOptional(filter, value string) bool {
	return filter == "" || strings.EqualFold(filter, value)
}

func addDubboGroupLink(groups map[string]*dubboServiceGroup, key string, contract dubboContract, link CrossRepoLink) {
	group := groups[key]
	if group == nil {
		group = &dubboServiceGroup{contract: contract}
		groups[key] = group
	}
	group.links = append(group.links, link)
}

func dubboServiceKey(contract dubboContract) string {
	return strings.Join([]string{contract.Interface, contract.Group, contract.Version, contract.Protocol}, "|")
}

func dubboContractLess(left, right dubboContract) bool {
	leftKey := strings.ToLower(dubboServiceKey(left))
	rightKey := strings.ToLower(dubboServiceKey(right))
	return leftKey < rightKey
}

func dubboSimpleName(interfaceName string) string {
	if index := strings.LastIndex(interfaceName, "."); index >= 0 {
		return interfaceName[index+1:]
	}
	return interfaceName
}

func crossRepoSlug(id string) string {
	if index := strings.Index(id, "::"); index >= 0 {
		return id[:index]
	}
	return ""
}

func dubboSourceEndpoint(link CrossRepoLink) dubboEndpoint {
	return dubboEndpoint{ID: link.Source, Repo: crossRepoSlug(link.Source), Name: link.SourceName, QualifiedName: link.SourceQualifiedName, File: link.SourceFile, Line: link.SourceLine, ModulePath: link.SourceModulePath}
}

func dubboTargetEndpoint(link CrossRepoLink) dubboEndpoint {
	return dubboEndpoint{ID: link.Target, Repo: crossRepoSlug(link.Target), Name: link.TargetName, QualifiedName: link.TargetQualifiedName, File: link.TargetFile, Line: link.TargetLine, ModulePath: link.TargetModulePath}
}

func sortedDubboEndpoints(values map[string]dubboEndpoint) []dubboEndpoint {
	out := make([]dubboEndpoint, 0, len(values))
	for _, endpoint := range values {
		out = append(out, endpoint)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Repo != out[j].Repo {
			return out[i].Repo < out[j].Repo
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func sortedDubboMethods(values map[string]*dubboMethod) []dubboMethod {
	out := make([]dubboMethod, 0, len(values))
	for _, method := range values {
		out = append(out, *method)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Arity < out[j].Arity
	})
	return out
}

func sortedSet(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func addNonEmpty(values map[string]struct{}, value string) {
	if value != "" {
		values[value] = struct{}{}
	}
}
