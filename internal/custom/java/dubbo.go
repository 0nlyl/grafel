package java

import (
	"regexp"
	"strings"
)

var (
	dubboReferenceFieldRE = regexp.MustCompile(`(?s)@(DubboReference|Reference)\s*(?:\(([^)]*)\))?\s*(?:private|protected|public)?\s*(?:final\s+)?([\w.$]+)\s+(\w+)\s*[;=]`)
	dubboServiceClassRE   = regexp.MustCompile(`(?s)@(DubboService|Service)\s*(?:\(([^)]*)\))?\s*(?:@[\w.]+(?:\([^)]*\))?\s*)*(?:public\s+)?(?:(?:abstract|final)\s+)?class\s+(\w+)(?:\s+extends\s+[\w.$<>]+)?(?:\s+implements\s+([^\{]+))?\s*\{`)
	dubboBeanMethodRE     = regexp.MustCompile(`(?s)(ReferenceBean|ServiceBean)\s*<\s*([\w.$]+)\s*>\s+(\w+)\s*\([^)]*\)\s*\{`)
	dubboProviderMethodRE = regexp.MustCompile(`(?m)(?:public|protected)\s+(?:static\s+)?[\w.$<>?,\[\]]+\s+(\w+)\s*\(([^)]*)\)\s*(?:throws\s+[^{]+)?\{`)
)

type dubboInterfaceResolution struct {
	name       string
	candidates []string
	resolved   bool
}

// ExtractDubbo normalizes Apache Dubbo and legacy Alibaba Dubbo Java
// configuration into RPC client/service entities. Cross-repository matching is
// deferred to a link pass so unresolved group/version values are never guessed.
func ExtractDubbo(ctx PatternContext) PatternResult {
	if strings.ToLower(ctx.Language) != "java" || ctx.Framework != "dubbo" {
		return PatternResult{}
	}

	ctx.Source = maskJavaComments(ctx.Source)
	var result PatternResult
	seenRefs := make(map[string]bool)
	extractDubboReferenceAnnotations(ctx, &result, seenRefs)
	extractDubboServiceAnnotations(ctx, &result, seenRefs)
	extractDubboBeanMethods(ctx, &result, seenRefs)
	return result
}

func maskJavaComments(source string) string {
	masked := []byte(source)
	inString := false
	inChar := false
	escaped := false
	for index := 0; index < len(masked); index++ {
		current := masked[index]
		if inString || inChar {
			if escaped {
				escaped = false
				continue
			}
			if current == '\\' {
				escaped = true
				continue
			}
			if inString && current == '"' {
				inString = false
			} else if inChar && current == '\'' {
				inChar = false
			}
			continue
		}
		if current == '"' {
			inString = true
			continue
		}
		if current == '\'' {
			inChar = true
			continue
		}
		if current != '/' || index+1 >= len(masked) {
			continue
		}
		switch masked[index+1] {
		case '/':
			for index < len(masked) && masked[index] != '\n' {
				masked[index] = ' '
				index++
			}
		case '*':
			masked[index] = ' '
			index++
			masked[index] = ' '
			for index+1 < len(masked) && !(masked[index] == '*' && masked[index+1] == '/') {
				if masked[index] != '\n' && masked[index] != '\r' {
					masked[index] = ' '
				}
				index++
			}
			if index+1 < len(masked) {
				masked[index] = ' '
				masked[index+1] = ' '
				index++
			}
		}
	}
	return string(masked)
}

func extractDubboReferenceAnnotations(ctx PatternContext, result *PatternResult, seenRefs map[string]bool) {
	for _, match := range dubboReferenceFieldRE.FindAllStringSubmatchIndex(ctx.Source, -1) {
		annotation := ctx.Source[match[2]:match[3]]
		if !hasDubboAnnotationImport(ctx.Source, annotation) {
			continue
		}
		attrs := submatch(ctx.Source, match, 4)
		fieldType := ctx.Source[match[6]:match[7]]
		fieldName := ctx.Source[match[8]:match[9]]
		interfaceResolution := dubboInterface(ctx.Source, attrs, fieldType)
		addDubboEntity(ctx, result, seenRefs, "consumer", interfaceResolution, fieldName, "annotation", attrs, match[0], "")
		extractDubboConsumerCalls(ctx, result, seenRefs, interfaceResolution, fieldName, attrs)
	}
}

func extractDubboServiceAnnotations(ctx PatternContext, result *PatternResult, seenRefs map[string]bool) {
	for _, match := range dubboServiceClassRE.FindAllStringSubmatchIndex(ctx.Source, -1) {
		annotation := ctx.Source[match[2]:match[3]]
		if !hasDubboAnnotationImport(ctx.Source, annotation) {
			continue
		}
		attrs := submatch(ctx.Source, match, 4)
		className := ctx.Source[match[6]:match[7]]
		implemented := submatch(ctx.Source, match, 8)
		fallback := ""
		if implemented != "" {
			fallback = strings.TrimSpace(strings.Split(implemented, ",")[0])
		}
		interfaceResolution := dubboInterface(ctx.Source, attrs, fallback)
		addDubboEntity(ctx, result, seenRefs, "provider", interfaceResolution, className, "annotation", attrs, match[0], className)
		openBrace := match[1] - 1
		if openBrace >= 0 && openBrace < len(ctx.Source) {
			extractDubboProviderMethods(ctx, result, seenRefs, interfaceResolution, className, attrs, ctx.Source[openBrace:matchingBrace(ctx.Source, openBrace)], openBrace)
		}
	}
}

func extractDubboConsumerCalls(ctx PatternContext, result *PatternResult, seenRefs map[string]bool, interfaceResolution dubboInterfaceResolution, fieldName, attrs string) {
	callRE := regexp.MustCompile(`\b` + regexp.QuoteMeta(fieldName) + `\s*\.\s*(\w+)\s*\(([^)]*)\)`)
	for _, match := range callRE.FindAllStringSubmatchIndex(ctx.Source, -1) {
		methodName := ctx.Source[match[2]:match[3]]
		arguments := submatch(ctx.Source, match, 4)
		addDubboMethodEntity(ctx, result, seenRefs, "consumer", interfaceResolution, methodName, javaArity(arguments), fieldName+"."+methodName, attrs, match[0])
	}
}

func extractDubboProviderMethods(ctx PatternContext, result *PatternResult, seenRefs map[string]bool, interfaceResolution dubboInterfaceResolution, className, attrs, body string, bodyOffset int) {
	for _, match := range dubboProviderMethodRE.FindAllStringSubmatchIndex(body, -1) {
		methodName := body[match[2]:match[3]]
		parameters := submatch(body, match, 4)
		addDubboMethodEntity(ctx, result, seenRefs, "provider", interfaceResolution, methodName, javaArity(parameters), className+"."+methodName, attrs, bodyOffset+match[0])
	}
}

func addDubboMethodEntity(ctx PatternContext, result *PatternResult, seenRefs map[string]bool, role string, interfaceResolution dubboInterfaceResolution, methodName string, arity int, sourceName, attrs string, offset int) {
	if interfaceResolution.name == "" || methodName == "" {
		return
	}
	props := map[string]any{
		"rpc_framework": "dubbo", "rpc_role": role,
		"rpc_method": methodName, "rpc_arity": arity, "source_style": "annotation", "source_name": sourceName,
	}
	applyDubboInterfaceProperties(props, interfaceResolution)
	for _, key := range []string{"group", "version", "protocol"} {
		if value := dubboAttribute(attrs, key); value != "" {
			cleaned, resolved := dubboConfigValue(value)
			props[key], props[key+"_resolved"] = cleaned, resolved
		}
	}
	ref := "scope:operation:dubbo_" + role + ":" + ctx.FilePath + ":" + sourceName + ":" + methodName
	addEntity(result, seenRefs, SecondaryEntity{
		Name: "dubbo:" + interfaceResolution.name + "#" + methodName, Kind: "SCOPE.Operation",
		Subtype: "rpc_" + role + "_method", SourceFile: ctx.FilePath,
		LineStart: lineOf(ctx.Source, offset), LineEnd: lineOf(ctx.Source, offset),
		Provenance: "INFERRED_FROM_DUBBO", Ref: ref, Properties: props,
	})
}

func javaArity(arguments string) int {
	arguments = strings.TrimSpace(arguments)
	if arguments == "" {
		return 0
	}
	depth := 0
	count := 1
	for _, char := range arguments {
		switch char {
		case '(', '<', '[', '{':
			depth++
		case ')', '>', ']', '}':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				count++
			}
		}
	}
	return count
}

func extractDubboBeanMethods(ctx PatternContext, result *PatternResult, seenRefs map[string]bool) {
	for _, match := range dubboBeanMethodRE.FindAllStringSubmatchIndex(ctx.Source, -1) {
		beanType := ctx.Source[match[2]:match[3]]
		genericType := ctx.Source[match[4]:match[5]]
		if len(genericType) == 1 && genericType[0] >= 'A' && genericType[0] <= 'Z' {
			continue
		}
		methodName := ctx.Source[match[6]:match[7]]
		openBrace := match[1] - 1
		if openBrace < 0 || openBrace >= len(ctx.Source) || ctx.Source[openBrace] != '{' {
			continue
		}
		body := ctx.Source[openBrace:matchingBrace(ctx.Source, openBrace)]
		interfaceResolution := resolveJavaType(ctx.Source, genericType)
		if configured := dubboSetterValue(body, "setInterface(?:Class)?"); configured != "" {
			interfaceResolution = resolveJavaType(ctx.Source, cleanJavaValue(configured))
		}
		role := "consumer"
		implementation := ""
		if beanType == "ServiceBean" {
			role = "provider"
			implementation = cleanJavaValue(dubboSetterValue(body, "setRef"))
		}
		addDubboEntity(ctx, result, seenRefs, role, interfaceResolution, methodName, "java_config", dubboSetterAttributes(body), match[0], implementation)
	}
}

func addDubboEntity(ctx PatternContext, result *PatternResult, seenRefs map[string]bool, role string, interfaceResolution dubboInterfaceResolution, sourceName, sourceStyle, attrs string, offset int, implementation string) {
	if interfaceResolution.name == "" {
		return
	}
	subtype := "rpc_client"
	if role == "provider" {
		subtype = "rpc_service"
	}
	props := map[string]any{
		"rpc_framework": "dubbo",
		"rpc_role":      role,
		"source_style":  sourceStyle,
		"source_name":   sourceName,
	}
	applyDubboInterfaceProperties(props, interfaceResolution)
	for _, key := range []string{"group", "version", "protocol"} {
		if value := dubboAttribute(attrs, key); value != "" {
			cleaned, resolved := dubboConfigValue(value)
			props[key] = cleaned
			props[key+"_resolved"] = resolved
		}
	}
	if implementation != "" {
		props["implementation"] = implementation
	}
	ref := "scope:service:dubbo_" + role + ":" + ctx.FilePath + ":" + sourceName
	addEntity(result, seenRefs, SecondaryEntity{
		Name:       "dubbo:" + role + ":" + interfaceResolution.name,
		Kind:       "SCOPE.Service",
		Subtype:    subtype,
		SourceFile: ctx.FilePath,
		LineStart:  lineOf(ctx.Source, offset),
		LineEnd:    lineOf(ctx.Source, offset),
		Provenance: "INFERRED_FROM_DUBBO",
		Ref:        ref,
		Properties: props,
	})
}

func hasDubboAnnotationImport(source, annotation string) bool {
	suffix := "." + annotation + ";"
	return strings.Contains(source, "import org.apache.dubbo.config.annotation"+suffix) ||
		strings.Contains(source, "import com.alibaba.dubbo.config.annotation"+suffix)
}

func dubboInterface(source, attrs, fallback string) dubboInterfaceResolution {
	for _, key := range []string{"interfaceClass", "interfaceName", "interface"} {
		if value := dubboAttribute(attrs, key); value != "" {
			return resolveJavaType(source, cleanJavaValue(value))
		}
	}
	return resolveJavaType(source, fallback)
}

func resolveJavaType(source, typeName string) dubboInterfaceResolution {
	typeName = strings.TrimSpace(strings.TrimSuffix(typeName, ".class"))
	if typeName == "" || strings.Contains(typeName, "${") {
		return dubboInterfaceResolution{name: typeName}
	}
	if strings.Contains(typeName, ".") {
		return dubboInterfaceResolution{name: typeName, resolved: true}
	}
	pattern := regexp.MustCompile(`(?m)^import\s+([\w.]+\.` + regexp.QuoteMeta(typeName) + `);\s*$`)
	if match := pattern.FindStringSubmatch(source); len(match) == 2 {
		return dubboInterfaceResolution{name: match[1], resolved: true}
	}
	var candidates []string
	packagePattern := regexp.MustCompile(`(?m)^package\s+([\w.]+)\s*;`)
	if match := packagePattern.FindStringSubmatch(source); len(match) == 2 {
		candidates = append(candidates, match[1]+"."+typeName)
	}
	wildcardPattern := regexp.MustCompile(`(?m)^import\s+([\w.]+)\.\*\s*;\s*$`)
	for _, match := range wildcardPattern.FindAllStringSubmatch(source, -1) {
		candidate := match[1] + "." + typeName
		if !containsString(candidates, candidate) {
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) == 1 {
		return dubboInterfaceResolution{name: candidates[0], resolved: true}
	}
	if len(candidates) > 1 {
		return dubboInterfaceResolution{name: typeName, candidates: candidates}
	}
	return dubboInterfaceResolution{name: typeName}
}

func applyDubboInterfaceProperties(props map[string]any, resolution dubboInterfaceResolution) {
	props["interface_fqn"] = resolution.name
	if resolution.resolved {
		return
	}
	props["interface_fqn_resolved"] = false
	if len(resolution.candidates) > 0 {
		props["interface_candidates"] = strings.Join(resolution.candidates, ",")
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func dubboSetterAttributes(body string) string {
	parts := make([]string, 0, 3)
	setters := []struct{ key, setter string }{
		{"group", "setGroup"},
		{"version", "setVersion"},
		{"protocol", "setProtocol"},
	}
	for _, pair := range setters {
		if value := dubboSetterValue(body, pair.setter); value != "" {
			parts = append(parts, pair.key+"="+value)
		}
	}
	return strings.Join(parts, ",")
}

func dubboSetterValue(body, setter string) string {
	pattern := regexp.MustCompile(`\b` + setter + `\s*\(\s*([^\)]+)\s*\)`)
	if match := pattern.FindStringSubmatch(body); len(match) == 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func dubboAttribute(attrs, key string) string {
	if attrs == "" {
		return ""
	}
	pattern := regexp.MustCompile(`(?:^|,)\s*` + regexp.QuoteMeta(key) + `\s*=\s*([^,]+)`)
	if match := pattern.FindStringSubmatch(attrs); len(match) == 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func dubboConfigValue(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	cleaned := cleanJavaValue(trimmed)
	resolved := strings.HasPrefix(trimmed, `"`) && strings.HasSuffix(trimmed, `"`) && !strings.Contains(cleaned, "${")
	return cleaned, resolved
}

func cleanJavaValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, ".class")
	value = strings.Trim(value, `"`)
	return strings.TrimSpace(value)
}

func submatch(source string, match []int, index int) string {
	if index+1 >= len(match) || match[index] < 0 || match[index+1] < 0 {
		return ""
	}
	return source[match[index]:match[index+1]]
}
