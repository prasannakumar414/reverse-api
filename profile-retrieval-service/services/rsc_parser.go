package services

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ErrUnsupportedRSCLayout indicates that a valid Flight payload no longer has
// the component ownership needed to extract profile records safely.
var ErrUnsupportedRSCLayout = errors.New("unsupported linkedin rsc layout")

var rscDateRangePattern = regexp.MustCompile(`(?i)^(?:[a-z]{3}\s+)?(?:19|20)\d{2}\s+-\s+(?:present|(?:[a-z]{3}\s+)?(?:19|20)\d{2})(?:\s+·.*)?$`)

type flightDocument struct {
	rows map[string]any
}

type flightElement struct {
	value []any
	props map[string]any
}

type flightTextSource uint8

const (
	flightChildText flightTextSource = iota
	flightTextProps
)

type flightTextChunk struct {
	text   string
	source flightTextSource
}

// decodeFlightDocument parses all model chunks before references are resolved.
// This mirrors the two-pass parser/resolver design used by rsc-to-json while
// retaining the original React element tuples needed by the section extractors.
func decodeFlightDocument(document string) (*flightDocument, error) {
	rows := map[string]any{}
	scanner := bufio.NewScanner(strings.NewReader(document))
	scanner.Buffer(make([]byte, 64*1024), 12<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		separator := strings.IndexByte(line, ':')
		if separator <= 0 || separator == len(line)-1 {
			continue
		}

		rowID := line[:separator]
		if _, err := strconv.ParseUint(rowID, 16, 64); err != nil {
			continue
		}

		payload := line[separator+1:]
		if payload[0] != '[' && payload[0] != '{' && payload[0] != '"' {
			continue
		}

		var value any
		if err := json.Unmarshal([]byte(payload), &value); err != nil {
			return nil, fmt.Errorf("decode flight row %s: %w", rowID, err)
		}
		rows[strings.ToLower(rowID)] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan flight document: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%w: no decodable flight rows", ErrUnsupportedRSCLayout)
	}
	if _, ok := rows["0"]; !ok {
		return nil, fmt.Errorf("%w: root flight row is missing", ErrUnsupportedRSCLayout)
	}
	return &flightDocument{rows: rows}, nil
}

func asFlightElement(value any) (flightElement, bool) {
	items, ok := value.([]any)
	if !ok || len(items) < 4 || items[0] != "$" {
		return flightElement{}, false
	}
	props, ok := items[3].(map[string]any)
	if !ok {
		return flightElement{}, false
	}
	return flightElement{value: items, props: props}, true
}

func (element flightElement) elementType() string {
	if len(element.value) < 2 {
		return ""
	}
	typeName, _ := element.value[1].(string)
	return typeName
}

func (document *flightDocument) resolveReference(value string) (any, string, bool) {
	if !strings.HasPrefix(value, "$L") {
		return nil, "", false
	}

	reference := value[2:]
	rowID, _, _ := strings.Cut(reference, ":")
	if rowID == "" {
		return nil, "", false
	}
	if _, err := strconv.ParseUint(rowID, 16, 64); err != nil {
		return nil, "", false
	}
	resolved, ok := document.rows[strings.ToLower(rowID)]
	return resolved, strings.ToLower(rowID), ok
}

func (document *flightDocument) findElements(predicate func(flightElement) bool) []flightElement {
	var found []flightElement
	visiting := map[string]bool{}
	var walk func(any)
	walk = func(value any) {
		switch typed := value.(type) {
		case string:
			resolved, rowID, ok := document.resolveReference(typed)
			if !ok || visiting[rowID] {
				return
			}
			visiting[rowID] = true
			walk(resolved)
			delete(visiting, rowID)
		case []any:
			if element, ok := asFlightElement(typed); ok {
				if predicate(element) {
					found = append(found, element)
					return
				}
				walk(element.props["children"])
				return
			}
			for _, child := range typed {
				walk(child)
			}
		case map[string]any:
			walk(typed["children"])
		}
	}
	walk(document.rows["0"])
	return found
}

func (document *flightDocument) renderedText(value any) []flightTextChunk {
	var chunks []flightTextChunk
	visiting := map[string]bool{}
	var walk func(any, flightTextSource)
	walk = func(current any, source flightTextSource) {
		switch typed := current.(type) {
		case string:
			if resolved, rowID, ok := document.resolveReference(typed); ok {
				if visiting[rowID] {
					return
				}
				visiting[rowID] = true
				walk(resolved, source)
				delete(visiting, rowID)
				return
			}
			text := strings.TrimSpace(strings.ReplaceAll(typed, "\u00a0", " "))
			if text == "" || strings.HasPrefix(text, "$") {
				return
			}
			chunks = append(chunks, flightTextChunk{text: text, source: source})
		case []any:
			if element, ok := asFlightElement(typed); ok {
				if textProps, ok := element.props["textProps"].(map[string]any); ok {
					walk(textProps["children"], flightTextProps)
					return
				}
				walk(element.props["children"], source)
				return
			}
			for _, child := range typed {
				walk(child, source)
			}
		case map[string]any:
			if textProps, ok := typed["textProps"].(map[string]any); ok {
				walk(textProps["children"], flightTextProps)
				return
			}
			walk(typed["children"], source)
		}
	}
	walk(value, flightChildText)
	return chunks
}

func (document *flightDocument) referenceTextBlocks(value any, skippedElementType string) [][]flightTextChunk {
	var blocks [][]flightTextChunk
	var walk func(any)
	walk = func(current any) {
		switch typed := current.(type) {
		case string:
			resolved, _, ok := document.resolveReference(typed)
			if !ok {
				return
			}
			if chunks := document.renderedText(resolved); len(chunks) > 0 {
				blocks = append(blocks, chunks)
			}
		case []any:
			if element, ok := asFlightElement(typed); ok {
				if skippedElementType != "" && element.elementType() == skippedElementType {
					return
				}
				walk(element.props["children"])
				return
			}
			for _, child := range typed {
				walk(child)
			}
		case map[string]any:
			walk(typed["children"])
		}
	}
	walk(value)
	return blocks
}

func (document *flightDocument) hasRenderedText(target string) bool {
	for _, chunk := range document.renderedText(document.rows["0"]) {
		if strings.EqualFold(chunk.text, target) {
			return true
		}
	}
	return false
}

func (document *flightDocument) educationItems() ([]Education, error) {
	roots := document.findElements(func(element flightElement) bool {
		tracking, _ := element.props["viewTrackingSpecs"].(map[string]any)
		return tracking["viewName"] == "education-lockup-view"
	})
	if len(roots) == 0 {
		if document.hasRenderedText("Nothing to see for now") {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: education item components not found", ErrUnsupportedRSCLayout)
	}

	items := make([]Education, 0, len(roots))
	for index, root := range roots {
		blocks := document.referenceTextBlocks(root.value, "")
		if len(blocks) == 0 {
			return nil, fmt.Errorf("%w: education item %d has no rendered content", ErrUnsupportedRSCLayout, index)
		}

		primary, secondary := splitFlightText(blocks[0])
		if len(primary) == 0 {
			return nil, fmt.Errorf("%w: education item %d has no school", ErrUnsupportedRSCLayout, index)
		}
		item := Education{School: primary[0]}
		if len(primary) > 1 {
			mergeDegreeAndField(&item, primary[1])
		}
		if len(secondary) > 0 {
			dateRange, err := parseRSCDateRange(secondary[0])
			if err != nil {
				return nil, fmt.Errorf("education item %d: %w", index, err)
			}
			item.DateRange = dateRange
		}
		mergeRSCDetailBlocks(&item.Skills, nil, blocks[1:])
		items = append(items, item)
	}
	return items, nil
}

func (document *flightDocument) experienceItems() ([]Experience, error) {
	roots := document.findElements(func(element flightElement) bool {
		componentKey, _ := element.props["componentKey"].(string)
		return strings.HasPrefix(componentKey, "entity-collection-item-")
	})
	if len(roots) == 0 {
		if document.hasRenderedText("Nothing to see for now") {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: experience item components not found", ErrUnsupportedRSCLayout)
	}

	var items []Experience
	for index, root := range roots {
		roleLists := findFlightElements(document, root.value, "ul")
		if len(roleLists) == 0 {
			item, err := document.standaloneExperience(root, index)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
			continue
		}

		headerBlocks := document.referenceTextBlocks(root.value, "ul")
		if len(headerBlocks) == 0 {
			return nil, fmt.Errorf("%w: experience group %d has no company header", ErrUnsupportedRSCLayout, index)
		}
		companyLines, _ := splitFlightText(headerBlocks[0])
		if len(companyLines) == 0 {
			return nil, fmt.Errorf("%w: experience group %d has no company", ErrUnsupportedRSCLayout, index)
		}
		company := companyLines[0]
		groupEmploymentType := ""
		if len(companyLines) > 1 {
			groupEmploymentType, _ = employmentTypeFromSummary(companyLines[1])
		}

		roles := findFlightElements(document, roleLists[0].value, "li")
		if len(roles) == 0 {
			return nil, fmt.Errorf("%w: experience group %d has no role items", ErrUnsupportedRSCLayout, index)
		}
		for roleIndex, role := range roles {
			roleItem, employmentType, err := document.groupedExperience(role, company, groupEmploymentType, index, roleIndex)
			if err != nil {
				return nil, err
			}
			if employmentType != "" {
				groupEmploymentType = employmentType
			}
			items = append(items, roleItem)
		}
	}
	return items, nil
}

func (document *flightDocument) standaloneExperience(root flightElement, index int) (Experience, error) {
	blocks := document.referenceTextBlocks(root.value, "")
	if len(blocks) == 0 {
		return Experience{}, fmt.Errorf("%w: experience item %d has no rendered content", ErrUnsupportedRSCLayout, index)
	}
	primary, secondary := splitFlightText(blocks[0])
	if len(primary) < 2 || len(secondary) == 0 {
		return Experience{}, fmt.Errorf("%w: experience item %d summary is incomplete", ErrUnsupportedRSCLayout, index)
	}

	companyParts := splitClean(primary[1], "·")
	item := Experience{Title: primary[0], Company: companyParts[0]}
	if len(companyParts) > 1 {
		item.EmploymentType, _ = canonicalEmploymentType(companyParts[1])
	}
	dateIndex := 0
	if employmentType, ok := canonicalEmploymentType(secondary[0]); ok {
		item.EmploymentType = employmentType
		dateIndex++
	}
	if dateIndex >= len(secondary) {
		return Experience{}, fmt.Errorf("%w: experience item %d has no date range", ErrUnsupportedRSCLayout, index)
	}
	dateRange, err := parseRSCDateRange(secondary[dateIndex])
	if err != nil {
		return Experience{}, fmt.Errorf("experience item %d: %w", index, err)
	}
	item.DateRange = dateRange
	if len(primary) > 2 {
		item.Location = primary[2]
	} else if dateIndex+1 < len(secondary) {
		item.Location = secondary[dateIndex+1]
	}
	mergeRSCDetailBlocks(&item.Skills, &item.Description, blocks[1:])
	return item, nil
}

func (document *flightDocument) groupedExperience(root flightElement, company, inheritedEmploymentType string, groupIndex, roleIndex int) (Experience, string, error) {
	blocks := document.referenceTextBlocks(root.value, "")
	if len(blocks) == 0 {
		return Experience{}, inheritedEmploymentType, fmt.Errorf("%w: experience group %d role %d has no rendered content", ErrUnsupportedRSCLayout, groupIndex, roleIndex)
	}
	primary, secondary := splitFlightText(blocks[0])
	if len(primary) == 0 || len(secondary) == 0 {
		return Experience{}, inheritedEmploymentType, fmt.Errorf("%w: experience group %d role %d summary is incomplete", ErrUnsupportedRSCLayout, groupIndex, roleIndex)
	}

	item := Experience{Title: primary[0], Company: company, EmploymentType: inheritedEmploymentType}
	dateValue := ""
	locationValue := ""
	for _, value := range secondary {
		if employmentType, ok := canonicalEmploymentType(value); ok {
			item.EmploymentType = employmentType
			continue
		}
		if dateValue == "" {
			dateValue = value
			continue
		}
		if locationValue == "" {
			locationValue = value
		}
	}
	if dateValue == "" {
		return Experience{}, inheritedEmploymentType, fmt.Errorf("%w: experience group %d role %d has no date range", ErrUnsupportedRSCLayout, groupIndex, roleIndex)
	}
	dateRange, err := parseRSCDateRange(dateValue)
	if err != nil {
		return Experience{}, inheritedEmploymentType, fmt.Errorf("experience group %d role %d: %w", groupIndex, roleIndex, err)
	}
	item.DateRange = dateRange
	if len(primary) > 1 {
		item.Location = primary[1]
	} else {
		item.Location = locationValue
	}
	mergeRSCDetailBlocks(&item.Skills, &item.Description, blocks[1:])
	return item, item.EmploymentType, nil
}

func findFlightElements(document *flightDocument, value any, elementType string) []flightElement {
	var found []flightElement
	visiting := map[string]bool{}
	var walk func(any)
	walk = func(current any) {
		switch typed := current.(type) {
		case string:
			resolved, rowID, ok := document.resolveReference(typed)
			if !ok || visiting[rowID] {
				return
			}
			visiting[rowID] = true
			walk(resolved)
			delete(visiting, rowID)
		case []any:
			if element, ok := asFlightElement(typed); ok {
				if element.elementType() == elementType {
					found = append(found, element)
					return
				}
				walk(element.props["children"])
				return
			}
			for _, child := range typed {
				walk(child)
			}
		case map[string]any:
			walk(typed["children"])
		}
	}
	walk(value)
	return found
}

func splitFlightText(chunks []flightTextChunk) ([]string, []string) {
	var primary []string
	var secondary []string
	for _, chunk := range chunks {
		switch chunk.source {
		case flightTextProps:
			secondary = append(secondary, chunk.text)
		default:
			primary = append(primary, chunk.text)
		}
	}
	return primary, secondary
}

func mergeRSCDetailBlocks(skills *[]string, description *string, blocks [][]flightTextChunk) {
	var descriptions []string
	for _, block := range blocks {
		for _, chunk := range block {
			line := strings.TrimSpace(chunk.text)
			if line == "" || strings.EqualFold(line, "Skills:") {
				continue
			}
			if parsedSkills := skillsFromSummaryLine(line); len(parsedSkills) > 0 {
				*skills = append(*skills, parsedSkills...)
				continue
			}
			if description != nil {
				descriptions = append(descriptions, line)
			}
		}
	}
	if description != nil {
		*description = strings.Join(descriptions, "\n")
	}
}

func canonicalEmploymentType(value string) (string, bool) {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "full-time", "part-time", "self-employed", "freelance", "contract", "internship", "apprenticeship", "seasonal", "temporary":
		return value, true
	default:
		return "", false
	}
}

func employmentTypeFromSummary(value string) (string, bool) {
	parts := splitClean(value, "·")
	if len(parts) == 0 {
		return "", false
	}
	return canonicalEmploymentType(parts[0])
}

func parseRSCDateRange(value string) (string, error) {
	value = normalizeDash(strings.TrimSpace(value))
	if !rscDateRangePattern.MatchString(value) {
		return "", fmt.Errorf("%w: invalid date range %q", ErrUnsupportedRSCLayout, value)
	}
	return value, nil
}
