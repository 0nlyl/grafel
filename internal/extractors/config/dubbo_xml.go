package config

import (
	"bytes"
	"encoding/xml"
	"io"
	"path/filepath"
	"strings"

	"github.com/cajasmota/grafel/internal/types"
)

func isDubboSpringXML(content []byte) bool {
	return bytes.Contains(bytes.ToLower(content), []byte("<dubbo:"))
}

func discoverDubboXML(repoRoot, rel string, spec configSpec, content []byte, configID string) ([]types.EntityRecord, []types.RelationshipRecord) {
	if spec.subtype != "dubbo_spring_xml" || !isDubboSpringXML(content) {
		return nil, nil
	}
	decoder := xml.NewDecoder(bytes.NewReader(content))
	repoTag := filepath.Base(repoRoot)
	var entities []types.EntityRecord
	var relationships []types.RelationshipRecord
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return entities, relationships
		}
		start, ok := token.(xml.StartElement)
		if !ok || (start.Name.Local != "reference" && start.Name.Local != "service") {
			continue
		}
		if !isDubboElementNamespace(start.Name.Space) {
			continue
		}
		attrs := xmlAttributes(start.Attr)
		interfaceName := attrs["interface"]
		if interfaceName == "" {
			continue
		}
		role := "consumer"
		subtype := "rpc_client"
		sourceName := attrs["id"]
		if start.Name.Local == "service" {
			role = "provider"
			subtype = "rpc_service"
			sourceName = attrs["ref"]
		}
		if sourceName == "" {
			sourceName = interfaceName
		}
		props := map[string]string{
			"rpc_framework": "dubbo",
			"rpc_role":      role,
			"interface_fqn": interfaceName,
			"source_style":  "xml",
			"source_name":   sourceName,
		}
		for _, key := range []string{"group", "version", "protocol"} {
			if value := attrs[key]; value != "" {
				props[key] = value
				props[key+"_resolved"] = boolString(!strings.Contains(value, "${"))
			}
		}
		if ref := attrs["ref"]; ref != "" {
			props["implementation"] = ref
		}
		line := 1 + bytes.Count(content[:decoder.InputOffset()], []byte("\n"))
		id := "scope:service:dubbo_" + role + ":" + filepath.ToSlash(rel) + ":" + sourceName
		entities = append(entities, types.EntityRecord{
			ID: id, Name: "dubbo:" + role + ":" + interfaceName,
			QualifiedName: repoTag + "::" + filepath.ToSlash(rel) + "#" + sourceName,
			Kind:          string(types.EntityKindService), Subtype: subtype, Language: "xml",
			SourceFile: filepath.ToSlash(rel), StartLine: line, EndLine: line,
			Signature: start.Name.Local + " " + interfaceName, Properties: props,
		})
		relProps := types.Props{{K: "rpc_framework", V: "dubbo"}, {K: "rpc_role", V: role}}
		relationships = append(relationships, types.RelationshipRecord{FromID: configID, ToID: id, Kind: string(types.RelationshipKindConfigures), Properties: relProps})
	}
	return entities, relationships
}

// isDubboElementNamespace reports whether a <reference>/<service> element was
// declared under one of Dubbo's Spring schema namespaces. The file-level
// `<dubbo:` gate only says the document contains Dubbo somewhere: a mixed
// Spring context routinely also carries Gemini Blueprint (<osgi:service>,
// <osgi:reference>), CXF (<jaxws:service>) or Camel elements whose local names
// collide exactly, and those carry an `interface` attribute too. Without the
// namespace check those become Dubbo contracts and can be linked to each other
// across repositories.
//
// An undeclared `dubbo:` prefix leaves encoding/xml with the literal prefix as
// the namespace, so that value is accepted as well.
func isDubboElementNamespace(space string) bool {
	switch space {
	case "http://dubbo.apache.org/schema/dubbo",
		"http://code.alibabatech.com/schema/dubbo",
		"dubbo":
		return true
	}
	return strings.HasSuffix(space, "/schema/dubbo")
}

func xmlAttributes(attrs []xml.Attr) map[string]string {
	values := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		values[attr.Name.Local] = strings.TrimSpace(attr.Value)
	}
	return values
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
