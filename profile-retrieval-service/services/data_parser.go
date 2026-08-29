package services

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

type ProfileDataParser struct{}

func NewProfileDataParser() *ProfileDataParser {
	return &ProfileDataParser{}
}

func (p *ProfileDataParser) Merge(result *ProfileResult, document string, publicID string) {
	if result == nil || document == "" {
		return
	}

	isRSC := isRSCDocument(document)
	lines := htmlTextLines(document)
	p.mergeProfile(result, document, lines, publicID)
	if isRSC {
		if about := aboutFromRSC(document); about != "" {
			result.About = about
		}
	}
	p.mergeImages(result, document)
	p.mergeExperience(result, lines)
	p.mergeEducation(result, lines, isRSC)
	p.mergeSkills(result, lines)
	if isRSC {
		p.mergeRSCSkills(result, document)
	}
}

func (p *ProfileDataParser) mergeProfile(result *ProfileResult, document string, lines []string, publicID string) {
	if result.Name == "" {
		result.Name = profileNameFromHTML(document)
	}
	if result.Headline == "" && result.Name != "" {
		result.Headline = headlineFromLines(lines, result.Name)
	}
	if result.Location == "" {
		result.Location = locationFromLines(lines)
	}
	if result.MemberID == "" {
		result.MemberID = firstRegexGroup(document, `recipient=([A-Za-z0-9_-]+)`)
	}
	if result.ProfileURN == "" && result.MemberID != "" {
		result.ProfileURN = "urn:li:fsd_profile:" + result.MemberID
	}
	if result.About == "" {
		result.About = aboutFromLines(lines, publicID)
	}
	if result.About == "" {
		result.About = aboutFromRSC(document)
	}
}

func (p *ProfileDataParser) mergeImages(result *ProfileResult, document string) {
	for _, match := range regexp.MustCompile(`imageSrcSet="([^"]+)"`).FindAllStringSubmatch(document, -1) {
		srcset := html.UnescapeString(match[1])
		artifacts := imagesFromSrcset(srcset)
		lower := strings.ToLower(srcset)
		switch {
		case strings.Contains(lower, "profile-displaybackgroundimage"):
			result.Images.Background = append(result.Images.Background, artifacts...)
		case strings.Contains(lower, "profile-displayphoto"):
			result.Images.Profile = append(result.Images.Profile, artifacts...)
		}
	}
}

func (p *ProfileDataParser) mergeExperience(result *ProfileResult, lines []string) {
	start := indexLine(lines, "Experience")
	if start < 0 {
		return
	}

	end := nextIndexAny(lines, start+1, "Ad Options", "More profiles for you", "People also viewed", "About")
	section := sliceUntil(lines, start+1, end)
	item := Experience{}
	var descriptions []string
	for i := 0; i < len(section); i++ {
		line := section[i]
		switch {
		case item.Title == "":
			item.Title = line
		case item.Company == "" && strings.Contains(line, "·"):
			parts := splitClean(line, "·")
			item.Company = parts[0]
			if len(parts) > 1 {
				item.EmploymentType = parts[1]
			}
		case item.DateRange == "" && looksLikeDateRange(line):
			item.DateRange = line
		case item.Location == "" && looksLikeLocation(line):
			item.Location = line
		case strings.Contains(strings.ToLower(line), " skills"):
			item.Skills = skillsFromSummaryLine(line)
		default:
			descriptions = append(descriptions, line)
		}
	}
	item.Description = strings.Join(descriptions, "\n")
	if item.Title != "" || item.Company != "" {
		result.Experience = append(result.Experience, item)
	}
}

func (p *ProfileDataParser) mergeEducation(result *ProfileResult, lines []string, allowUnsectioned bool) {
	start := indexLine(lines, "Education")
	end := len(lines)
	if start >= 0 {
		end = nextIndexAny(lines, start+1, "Ad Options", "More profiles for you", "People also viewed", "About")
	} else if !allowUnsectioned {
		return
	}
	section := sliceUntil(lines, start+1, end)
	if start < 0 {
		section = lines
	}
	if len(section) == 0 {
		return
	}

	if items := educationItemsFromLines(section); len(items) > 0 {
		result.Education = append(result.Education, items...)
		return
	}

	item := Education{School: section[0]}
	for _, line := range section[1:] {
		switch {
		case item.DateRange == "" && looksLikeDateRange(line):
			item.DateRange = line
		case item.Degree == "" && looksLikeEducationDegree(line):
			mergeDegreeAndField(&item, line)
		case strings.Contains(strings.ToLower(line), " skills"):
			item.Skills = skillsFromSummaryLine(line)
		case item.FieldOfStudy == "":
			item.FieldOfStudy = line
		}
	}
	if item.School != "" || item.Degree != "" {
		result.Education = append(result.Education, item)
	}
}

func educationItemsFromLines(lines []string) []Education {
	var items []Education
	for i := 0; i+1 < len(lines); i++ {
		school := strings.TrimSpace(lines[i])
		degree := strings.TrimSpace(lines[i+1])
		if school == "" || isChromeLine(school) || looksLikeRecommendationLine(school) || !looksLikeEducationDegree(degree) {
			continue
		}
		if strings.HasPrefix(strings.ToLower(school), "skills for ") {
			continue
		}

		item := Education{School: school}
		mergeDegreeAndField(&item, degree)
		for j := i + 2; j < len(lines) && j <= i+5; j++ {
			line := lines[j]
			lower := strings.ToLower(strings.TrimSpace(line))
			switch {
			case item.DateRange == "" && looksLikeDateRange(line):
				item.DateRange = normalizeDash(line)
			case lower == "skills:" && j+1 < len(lines):
				item.Skills = skillsFromSummaryLine(lines[j+1])
			case strings.Contains(lower, " skills") && strings.Contains(line, ","):
				item.Skills = skillsFromSummaryLine(line)
			case item.FieldOfStudy == "" && !looksLikeEducationDegree(line) && !looksLikeLocation(line):
				item.FieldOfStudy = line
			}
		}
		items = append(items, item)
	}
	return items
}

func mergeDegreeAndField(item *Education, value string) {
	parts := splitClean(value, ",")
	item.Degree = parts[0]
	if len(parts) > 1 {
		item.FieldOfStudy = strings.Join(parts[1:], ", ")
	}
}

func (p *ProfileDataParser) mergeSkills(result *ProfileResult, lines []string) {
	start := indexLine(lines, "Skills")
	if start < 0 {
		return
	}

	end := nextIndexAny(lines, start+1, "Ad Options", "More profiles for you", "People also viewed", "About")
	for _, line := range sliceUntil(lines, start+1, end) {
		if isSkillCategory(line) || isChromeLine(line) || looksLikeRecommendationLine(line) {
			continue
		}
		if len(line) > 80 || strings.Contains(line, "·") {
			continue
		}
		result.Skills = append(result.Skills, Skill{Name: line})
	}
}

func (p *ProfileDataParser) mergeRSCSkills(result *ProfileResult, document string) {
	for _, match := range regexp.MustCompile(`"Endorse ((?:[^"\\]|\\.)+)"`).FindAllStringSubmatch(document, -1) {
		if len(match) < 2 {
			continue
		}
		name := strings.TrimSpace(unquoteJSONString(match[1]))
		if name == "" {
			continue
		}
		result.Skills = append(result.Skills, Skill{Name: name})
	}
}

func htmlTextLines(document string) []string {
	if isRSCDocument(document) {
		return rscTextLines(document)
	}

	cleaned := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(document, " ")
	cleaned = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`(?i)<br\s*/?>|</p>|</div>|</h[1-6]>|</li>|</section>|</span>`).ReplaceAllString(cleaned, "\n")
	cleaned = regexp.MustCompile(`(?s)<[^>]+>`).ReplaceAllString(cleaned, "\n")
	cleaned = html.UnescapeString(cleaned)
	cleaned = strings.ReplaceAll(cleaned, "\u00a0", " ")

	var lines []string
	for _, line := range strings.Split(cleaned, "\n") {
		line = strings.TrimSpace(regexp.MustCompile(`[ \t]+`).ReplaceAllString(line, " "))
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func isRSCDocument(document string) bool {
	return !strings.Contains(document, "<") && strings.Contains(document, `"children"`)
}

func rscTextLines(document string) []string {
	var lines []string
	pattern := regexp.MustCompile(`"children":\["((?:[^"\\]|\\.)*)"\]|"children":"((?:[^"\\]|\\.)*)"|\],"((?:[^"\\]|\\.)*skills)"`)
	for _, match := range pattern.FindAllStringSubmatch(document, -1) {
		for _, group := range match[1:] {
			if group == "" {
				continue
			}
			line := unquoteJSONString(group)
			line = strings.TrimSpace(strings.ReplaceAll(line, "\u00a0", " "))
			if line != "" && line != "$undefined" {
				lines = append(lines, line)
			}
			break
		}
	}
	return dedupeStrings(lines)
}

func unquoteJSONString(value string) string {
	unquoted, err := strconv.Unquote(`"` + value + `"`)
	if err != nil {
		return value
	}
	return unquoted
}

func profileNameFromHTML(document string) string {
	title := firstRegexGroup(document, `(?is)<title>\s*([^<|]+?)\s*\|\s*LinkedIn\s*</title>`)
	return strings.TrimSpace(html.UnescapeString(title))
}

func headlineFromLines(lines []string, name string) string {
	for i, line := range lines {
		if line != name || i+1 >= len(lines) {
			continue
		}
		candidate := lines[i+1]
		if candidate != "" && !isChromeLine(candidate) && candidate != "He/Him" && !strings.HasPrefix(candidate, "· ") {
			return candidate
		}
	}
	return ""
}

func locationFromLines(lines []string) string {
	for i, line := range lines {
		if line == "Contact info" && i > 0 {
			for j := i - 1; j >= 0 && j >= i-4; j-- {
				if looksLikeLocation(lines[j]) {
					return strings.TrimSpace(strings.Split(lines[j], "·")[0])
				}
			}
		}
	}
	for _, line := range lines {
		if looksLikeLocation(line) {
			return strings.TrimSpace(strings.Split(line, "·")[0])
		}
	}
	return ""
}

func aboutFromLines(lines []string, publicID string) string {
	start := indexLine(lines, "About")
	if start < 0 {
		return ""
	}
	end := nextIndexAny(lines, start+1, "Experience", "Education", "Skills", "Licenses & certifications", "Ad Options", "More profiles for you", "Accessibility")
	section := sliceUntil(lines, start+1, end)
	var body []string
	for _, line := range section {
		if isChromeLine(line) || strings.Contains(line, publicID) {
			continue
		}
		body = append(body, line)
	}
	return strings.Join(body, "\n")
}

func aboutFromRSC(document string) string {
	var paragraphs []string
	pattern := regexp.MustCompile(`"children":\[(?:null|\[\s*"\$","br",null,\{\}\s*\]),"((?:[^"\\]|\\.)+)"\]`)
	for _, match := range pattern.FindAllStringSubmatch(document, -1) {
		if len(match) < 2 {
			continue
		}
		paragraph := strings.TrimSpace(unquoteJSONString(match[1]))
		if len(paragraph) < 30 || looksLikeRecommendationLine(paragraph) {
			continue
		}
		paragraphs = append(paragraphs, paragraph)
	}
	return strings.Join(dedupeStrings(paragraphs), "\n")
}

func imagesFromSrcset(srcset string) []ImageArtifact {
	var images []ImageArtifact
	for _, entry := range strings.Split(srcset, ",") {
		fields := strings.Fields(strings.TrimSpace(entry))
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "http") {
			continue
		}
		image := ImageArtifact{URL: fields[0]}
		if len(fields) > 1 && strings.HasSuffix(fields[1], "w") {
			if width, ok := atoi(strings.TrimSuffix(fields[1], "w")); ok {
				image.Width = width
			}
		}
		images = append(images, image)
	}
	return images
}

func firstRegexGroup(value, pattern string) string {
	matches := regexp.MustCompile(pattern).FindStringSubmatch(value)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func indexLine(lines []string, exact string) int {
	for i, line := range lines {
		if strings.EqualFold(line, exact) {
			return i
		}
	}
	return -1
}

func nextIndexAny(lines []string, start int, markers ...string) int {
	for i := start; i < len(lines); i++ {
		for _, marker := range markers {
			if strings.EqualFold(lines[i], marker) {
				return i
			}
		}
	}
	return len(lines)
}

func sliceUntil(lines []string, start, end int) []string {
	if start < 0 || start >= len(lines) {
		return nil
	}
	if end < start || end > len(lines) {
		end = len(lines)
	}
	return lines[start:end]
}

func splitClean(value, separator string) []string {
	parts := strings.Split(value, separator)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func looksLikeDateRange(line string) bool {
	line = strings.ToLower(normalizeDash(line))
	return strings.Contains(line, " - ") &&
		(strings.Contains(line, "present") || regexp.MustCompile(`\b(19|20)\d{2}\b`).MatchString(line))
}

func normalizeDash(line string) string {
	return strings.ReplaceAll(line, "–", "-")
}

func looksLikeEducationDegree(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "degree") ||
		strings.Contains(lower, "bachelor") ||
		strings.Contains(lower, "master") ||
		strings.Contains(lower, "diploma") ||
		strings.Contains(lower, "btech") ||
		strings.Contains(lower, "mtech")
}

func looksLikeLocation(line string) bool {
	lower := strings.ToLower(line)
	if strings.Contains(lower, " · ") {
		left := strings.TrimSpace(strings.Split(line, "·")[0])
		return looksLikeLocation(left)
	}
	if strings.Contains(lower, "india") || strings.Contains(lower, "united states") || strings.Contains(lower, "remote") || strings.Contains(lower, "hybrid") {
		return true
	}
	return strings.Count(line, ",") >= 1 && len(line) <= 80
}

func skillsFromSummaryLine(line string) []string {
	line = strings.TrimSpace(line)
	lower := strings.ToLower(line)
	if !strings.Contains(lower, "skill") {
		return nil
	}
	line = strings.TrimSpace(strings.TrimPrefix(line, "Skills:"))
	if strings.EqualFold(line, "skills") || line == "" {
		return nil
	}
	if idx := strings.Index(lower, " and +"); idx >= 0 {
		line = line[:idx]
	}
	line = regexp.MustCompile(`,\s*\+\d+\s+skills?$`).ReplaceAllString(line, "")
	return dedupeStrings(splitClean(line, ","))
}

func isSkillCategory(line string) bool {
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "all", "industry knowledge", "tools & technologies", "other skills", "show all skills":
		return true
	default:
		return false
	}
}

func isChromeLine(line string) bool {
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "home", "my network", "jobs", "messaging", "notifications", "me", "for business", "more", "message", "contact info", "ad options", "why am i seeing this ad?", "manage your ad preferences", "hide or report this ad", "i don’t want to see this ad in my feed", "don't want to see this", "tell us why you don’t want to see this", "submit", "show more", "about", "accessibility", "talent solutions", "community guidelines", "careers", "marketing solutions", "privacy & terms", "ad choices", "advertising", "sales solutions", "mobile", "small business", "safety center", "questions?", "visit our help center.", "manage your account and privacy", "go to your settings.", "recommendation transparency", "learn more about recommended content.", "select language":
		return true
	default:
		return strings.HasPrefix(strings.ToLower(line), "try premium") ||
			strings.HasPrefix(strings.ToLower(line), "linkedin corporation") ||
			strings.HasPrefix(strings.ToLower(line), "your feedback will help")
	}
}

func looksLikeRecommendationLine(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "more profiles for you") ||
		strings.Contains(lower, "people also viewed") ||
		strings.Contains(lower, "· 3rd") ||
		strings.Contains(lower, "· 2nd") ||
		strings.Contains(lower, "· 1st")
}

func atoi(value string) (int, bool) {
	var result int
	if value == "" {
		return 0, false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, false
		}
		result = result*10 + int(r-'0')
	}
	return result, true
}
