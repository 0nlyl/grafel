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
