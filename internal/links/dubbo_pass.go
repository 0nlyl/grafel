package links

import (
	"sort"
	"strings"
)

const MethodDubbo = "dubbo"

type dubboHit struct {
	repo         string
	id           string
	role         string
	key          string
	interfaceKey string
	metadata     [3]dubboMetadata
	exact        bool
	identifier   string
	sourceFile   string
}

type dubboMetadata struct {
	value string
	known bool
}

func runDubboPass(graphs []repoGraph, paths Paths, rejects map[string]bool) (PassResult, error) {
	result := PassResult{Pass: MethodDubbo}
	if len(graphs) < 2 {
		_, _, err := replaceByMethod(paths.Links, newMethodSet(MethodDubbo), nil, rejects)
		return result, err
	}

	providers := map[string][]dubboHit{}
	providersByInterface := map[string][]dubboHit{}
	knownInterfaces := map[string]bool{}
	for _, graph := range graphs {
		for _, entity := range graph.Entities {
			if entity.QualifiedName != "" {
				knownInterfaces[entity.QualifiedName] = true
			}
		}
	}
	var consumers []dubboHit
	for _, graph := range graphs {
		for _, entity := range graph.Entities {
			hit, ok := newDubboHit(graph.Repo, entity, knownInterfaces)
			if !ok {
				continue
			}
			if hit.role == "provider" {
				if hit.exact {
					providers[hit.key] = append(providers[hit.key], hit)
				}
				providersByInterface[hit.interfaceKey] = append(providersByInterface[hit.interfaceKey], hit)
			} else {
				consumers = append(consumers, hit)
			}
		}
	}

	sort.Slice(consumers, func(i, j int) bool {
		if consumers[i].repo != consumers[j].repo {
			return consumers[i].repo < consumers[j].repo
		}
		return consumers[i].id < consumers[j].id
	})
	for key := range providers {
		sort.Slice(providers[key], func(i, j int) bool {
			if providers[key][i].repo != providers[key][j].repo {
				return providers[key][i].repo < providers[key][j].repo
			}
			return providers[key][i].id < providers[key][j].id
		})
	}
	for key := range providersByInterface {
		sort.Slice(providersByInterface[key], func(i, j int) bool {
			if providersByInterface[key][i].repo != providersByInterface[key][j].repo {
				return providersByInterface[key][i].repo < providersByInterface[key][j].repo
			}
			return providersByInterface[key][i].id < providersByInterface[key][j].id
		})
	}

	now := discoveredAt()
	var fresh []Link
	for _, consumer := range consumers {
		external := []dubboHit{}
		exact := false
		if consumer.exact {
			external = externalDubboProviders(providers[consumer.key], consumer.repo)
			if len(external) == 1 {
				exact = true
			}
		}
		if len(external) == 0 {
			for _, provider := range providersByInterface[consumer.interfaceKey] {
				if provider.repo != consumer.repo && dubboMetadataCompatible(consumer, provider) {
					external = append(external, provider)
				}
			}
		}
		result.Candidates += len(external)
		if len(external) != 1 {
			result.Skipped += len(external)
			continue
		}
		provider := external[0]
		source := entityKey(consumer.repo, consumer.id)
		target := entityKey(provider.repo, provider.id)
		channel := MethodDubbo
		identifier := consumer.identifier
		match := "interface_compatible_metadata"
		confidenceLevel := ConfidenceInferred
		if exact {
			match = "interface_group_version_protocol"
			confidenceLevel = ConfidenceResolved
		}
		link := Link{
			ID: MakeID(source, target, MethodDubbo), Source: source, Target: target,
			Relation: RelationCalls, Method: MethodDubbo, Confidence: ScoreDubbo(exact),
			Channel: &channel, Identifier: &identifier, DiscoveredAt: now,
			SourceLocations: [][]string{{consumer.sourceFile}, {provider.sourceFile}},
			Properties:      map[string]string{"match": match},
		}
		link.WithEdgeConfidence(confidenceLevel)
		fresh = append(fresh, link)
	}

	added, skipped, err := replaceByMethod(paths.Links, newMethodSet(MethodDubbo), fresh, rejects)
	if err != nil {
		return result, err
	}
	result.LinksAdded = added
	result.Skipped += skipped
	result.CrossRepoResolved = added
	return result, nil
}

func newDubboHit(repo string, entity entityNode, knownInterfaces map[string]bool) (dubboHit, bool) {
	if entity.Properties.Get("rpc_framework") != "dubbo" {
		return dubboHit{}, false
	}
	role := entity.Properties.Get("rpc_role")
	if role != "consumer" && role != "provider" {
		return dubboHit{}, false
	}
	interfaceName := strings.TrimSpace(entity.Properties.Get("interface_fqn"))
	if !strings.Contains(interfaceName, ".") {
		resolved := ""
		for _, candidate := range strings.Split(entity.Properties.Get("interface_candidates"), ",") {
			candidate = strings.TrimSpace(candidate)
			if candidate == "" || !knownInterfaces[candidate] {
				continue
			}
			if resolved != "" {
				return dubboHit{}, false
			}
			resolved = candidate
		}
		interfaceName = resolved
	}
	if interfaceName == "" || !strings.Contains(interfaceName, ".") {
		return dubboHit{}, false
	}
	values := []string{interfaceName}
	identifierValues := []string{interfaceName}
	metadata := [3]dubboMetadata{}
	exact := true
	for index, property := range []string{"group", "version", "protocol"} {
		value := strings.TrimSpace(entity.Properties.Get(property))
		known := value == "" || entity.Properties.Get(property+"_resolved") == "true"
		metadata[index] = dubboMetadata{value: value, known: known}
		if !known {
			exact = false
			identifierValues = append(identifierValues, "")
		} else {
			identifierValues = append(identifierValues, value)
		}
		values = append(values, value)
	}
	methodName := strings.TrimSpace(entity.Properties.Get("rpc_method"))
	if methodName != "" {
		arity := strings.TrimSpace(entity.Properties.Get("rpc_arity"))
		values = append(values, methodName, arity)
		identifierValues = append(identifierValues, methodName, arity)
	}
	key := strings.Join(values, "|")
	return dubboHit{
		repo: repo, id: entity.ID, role: role, key: key, exact: exact,
		interfaceKey: dubboInterfaceKey(interfaceName, methodName, strings.TrimSpace(entity.Properties.Get("rpc_arity"))),
		metadata:     metadata, identifier: "dubbo:" + strings.Join(identifierValues, "|"), sourceFile: entity.SourceFile,
	}, true
}

func dubboInterfaceKey(interfaceName, methodName, arity string) string {
	if methodName == "" {
		return interfaceName
	}
	return strings.Join([]string{interfaceName, methodName, arity}, "|")
}

func externalDubboProviders(providers []dubboHit, repo string) []dubboHit {
	external := make([]dubboHit, 0, len(providers))
	for _, provider := range providers {
		if provider.repo != repo {
			external = append(external, provider)
		}
	}
	return external
}

func dubboMetadataCompatible(consumer, provider dubboHit) bool {
	for index := range consumer.metadata {
		left := consumer.metadata[index]
		right := provider.metadata[index]
		if left.known && right.known && left.value != "" && right.value != "" && left.value != right.value {
			return false
		}
	}
	return true
}
