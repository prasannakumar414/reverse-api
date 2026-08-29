package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

var (
	ErrInvalidProfileURL = errors.New("invalid linkedin profile url")
	ErrFetchFailed       = errors.New("failed to fetch profile from linkedin apis")
)

const DefaultLinkedInRequestMinInterval = 3 * time.Second

type ProfileService struct {
	client       *http.Client
	cookieHeader string
	csrfToken    string
	dataParser   *ProfileDataParser
	requestPacer *RequestPacer
}

type ProfileServiceConfig struct {
	Client             *http.Client
	RequestMinInterval time.Duration
}

type ProfileResult struct {
	ProfileURL     string            `json:"profile_url"`
	PublicID       string            `json:"public_id"`
	ProfileURN     string            `json:"profile_urn,omitempty"`
	MemberID       string            `json:"member_id,omitempty"`
	VersionTag     string            `json:"version_tag,omitempty"`
	Name           string            `json:"name,omitempty"`
	Headline       string            `json:"headline,omitempty"`
	Location       string            `json:"location,omitempty"`
	About          string            `json:"about,omitempty"`
	Experience     []Experience      `json:"experience,omitempty"`
	Education      []Education       `json:"education,omitempty"`
	Skills         []Skill           `json:"skills,omitempty"`
	Certifications []Certification   `json:"certifications,omitempty"`
	Languages      []Language        `json:"languages,omitempty"`
	Images         ProfileImages     `json:"images,omitempty"`
	SourceURLs     map[string]string `json:"source_urls"`
	APIErrors      []string          `json:"api_errors,omitempty"`
	Missing        []string          `json:"missing,omitempty"`
}

type Experience struct {
	Title          string   `json:"title,omitempty"`
	Company        string   `json:"company,omitempty"`
	EmploymentType string   `json:"employment_type,omitempty"`
	DateRange      string   `json:"date_range,omitempty"`
	Location       string   `json:"location,omitempty"`
	Description    string   `json:"description,omitempty"`
	Skills         []string `json:"skills,omitempty"`
}

type Education struct {
	School       string   `json:"school,omitempty"`
	Degree       string   `json:"degree,omitempty"`
	FieldOfStudy string   `json:"field_of_study,omitempty"`
	DateRange    string   `json:"date_range,omitempty"`
	Skills       []string `json:"skills,omitempty"`
}

type Skill struct {
	Name string `json:"name,omitempty"`
}

type Certification struct {
	Name         string   `json:"name,omitempty"`
	Issuer       string   `json:"issuer,omitempty"`
	Issued       string   `json:"issued,omitempty"`
	Expires      string   `json:"expires,omitempty"`
	CredentialID string   `json:"credential_id,omitempty"`
	URL          string   `json:"url,omitempty"`
	Skills       []string `json:"skills,omitempty"`
}

type Language struct {
	Name        string `json:"name,omitempty"`
	Proficiency string `json:"proficiency,omitempty"`
}

type ProfileImages struct {
	Profile    []ImageArtifact `json:"profile,omitempty"`
	Background []ImageArtifact `json:"background,omitempty"`
}

type ImageArtifact struct {
	URL       string `json:"url"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}

type apiResponse struct {
	key  string
	url  string
	data any
	err  error
}

type htmlResponse struct {
	key  string
	url  string
	html string
	err  error
}

type rscResponse struct {
	key  string
	url  string
	data string
	err  error
}

func NewProfileService(client *http.Client) *ProfileService {
	return NewProfileServiceWithConfig(ProfileServiceConfig{
		Client:             client,
		RequestMinInterval: DefaultLinkedInRequestMinInterval,
	})
}

func NewProfileServiceWithConfig(config ProfileServiceConfig) *ProfileService {
	if config.Client == nil {
		config.Client = &http.Client{
			Timeout: 20 * time.Second,
		}
	}

	cookieHeader := loadLinkedInCookieHeader()
	return &ProfileService{
		client:       config.Client,
		cookieHeader: cookieHeader,
		csrfToken:    loadLinkedInCSRFToken(cookieHeader),
		dataParser:   NewProfileDataParser(),
		requestPacer: NewRequestPacer(config.RequestMinInterval),
	}
}

func (s *ProfileService) Retrieve(ctx context.Context, profileURL string) (*ProfileResult, error) {
	baseURL, publicID, err := normalizeProfileURL(profileURL)
	if err != nil {
		return nil, err
	}

	sourceURLs := sourceURLsFor(publicID)
	htmlURLs := htmlSourceURLsFor(publicID)
	responses, apiErrors, err := s.fetchAPIs(ctx, sourceURLs)
	htmlDocuments, htmlErrors := s.fetchHTMLs(ctx, htmlURLs)
	apiErrors = append(apiErrors, htmlErrors...)
	sort.Strings(apiErrors)

	if err != nil && len(htmlDocuments) == 0 {
		return nil, err
	}

	result := &ProfileResult{
		ProfileURL: baseURL,
		PublicID:   publicID,
		SourceURLs: mergeSourceURLs(sourceURLs, htmlURLs, rscSourceURLsFor(publicID)),
		APIErrors:  apiErrors,
	}

	for _, document := range responses {
		mergeProfileIdentity(result, document)
		mergeProfile(result, document, publicID)
		mergeImages(result, document)
		mergeExperience(result, document)
		mergeEducation(result, document)
		mergeSkills(result, document)
		mergeCertifications(result, document)
		mergeLanguages(result, document)
	}
	for _, document := range htmlDocuments {
		s.dataParser.Merge(result, document, publicID)
	}
	if result.MemberID != "" {
		rscDocuments, rscErrors := s.fetchRSCs(ctx, rscSourceURLsFor(publicID), publicID, result.MemberID)
		result.APIErrors = append(result.APIErrors, rscErrors...)
		sort.Strings(result.APIErrors)
		for _, document := range rscDocuments {
			s.dataParser.Merge(result, document, publicID)
		}
	}

	result.Experience = dedupeExperience(result.Experience)
	result.Education = dedupeEducation(result.Education)
	result.Skills = dedupeSkills(result.Skills)
	result.Certifications = dedupeCertifications(result.Certifications)
	result.Languages = dedupeLanguages(result.Languages)
	result.Images.Profile = dedupeImages(result.Images.Profile)
	result.Images.Background = dedupeImages(result.Images.Background)
	result.Missing = missingFields(result)

	return result, nil
}

func sourceURLsFor(publicID string) map[string]string {
	escaped := url.PathEscape(publicID)
	queryEscaped := url.QueryEscape(publicID)
	return map[string]string{
		"profile_identity": "https://www.linkedin.com/voyager/api/graphql?includeWebMetadata=true&variables=(memberIdentity:" + queryEscaped + ")&queryId=voyagerIdentityDashProfiles.b5c27c04968c409fc0ed3546575b9b7a",
		"profile_view":     "https://www.linkedin.com/voyager/api/identity/profiles/" + escaped + "/profileView",
		"contact_info":     "https://www.linkedin.com/voyager/api/identity/profiles/" + escaped + "/profileContactInfo",
		"positions":        "https://www.linkedin.com/voyager/api/identity/profiles/" + escaped + "/positions?count=100&start=0",
		"educations":       "https://www.linkedin.com/voyager/api/identity/profiles/" + escaped + "/educations?count=100&start=0",
		"skills":           "https://www.linkedin.com/voyager/api/identity/profiles/" + escaped + "/skills?count=100&start=0",
		"certifications":   "https://www.linkedin.com/voyager/api/identity/profiles/" + escaped + "/certifications?count=100&start=0",
		"languages":        "https://www.linkedin.com/voyager/api/identity/profiles/" + escaped + "/languages?count=100&start=0",
	}
}

func htmlSourceURLsFor(publicID string) map[string]string {
	escaped := url.PathEscape(publicID)
	base := "https://www.linkedin.com/in/" + escaped
	return map[string]string{
		"flagship_profile_html":        base,
		"flagship_experience_html":     base + "/details/experience",
		"flagship_education_html":      base + "/details/education",
		"flagship_skills_html":         base + "/details/skills",
		"flagship_languages_html":      base + "/details/languages",
		"flagship_certifications":      "https://www.linkedin.com/flagship-web/in/" + escaped + "/details/certifications/",
		"flagship_certifications_html": base + "/details/certifications",
	}
}

func rscSourceURLsFor(publicID string) map[string]string {
	return map[string]string{
		"flagship_education_rsc": "https://www.linkedin.com/flagship-web/rsc-action/actions/pagination?sduiid=com.linkedin.sdui.pagers.profile.details.education",
		"flagship_about_rsc":     "https://www.linkedin.com/flagship-web/rsc-action/actions/component?componentId=com.linkedin.sdui.generated.profile.dsl.impl.profileCardsAboveActivity&sduiid=com.linkedin.sdui.generated.profile.dsl.impl.profileCardsAboveActivity",
		"flagship_skills_rsc_0":  "https://www.linkedin.com/flagship-web/rsc-action/actions/pagination?sduiid=com.linkedin.sdui.pagers.profile.details.skills",
		"flagship_skills_rsc_10": "https://www.linkedin.com/flagship-web/rsc-action/actions/pagination?sduiid=com.linkedin.sdui.pagers.profile.details.skills",
		"flagship_skills_rsc_20": "https://www.linkedin.com/flagship-web/rsc-action/actions/pagination?sduiid=com.linkedin.sdui.pagers.profile.details.skills",
		"flagship_skills_rsc_30": "https://www.linkedin.com/flagship-web/rsc-action/actions/pagination?sduiid=com.linkedin.sdui.pagers.profile.details.skills",
		"flagship_skills_rsc_40": "https://www.linkedin.com/flagship-web/rsc-action/actions/pagination?sduiid=com.linkedin.sdui.pagers.profile.details.skills",
	}
}

func mergeSourceURLs(groups ...map[string]string) map[string]string {
	merged := map[string]string{}
	for _, group := range groups {
		for key, value := range group {
			merged[key] = value
		}
	}
	return merged
}

func (s *ProfileService) fetchAPIs(ctx context.Context, sourceURLs map[string]string) ([]any, []string, error) {
	ch := make(chan apiResponse, len(sourceURLs))
	for key, apiURL := range sourceURLs {
		go func(key, apiURL string) {
			data, err := s.fetchAPI(ctx, apiURL)
			ch <- apiResponse{key: key, url: apiURL, data: data, err: err}
		}(key, apiURL)
	}

	var documents []any
	var errs []error
	var apiErrors []string
	for range sourceURLs {
		response := <-ch
		if response.err != nil {
			apiErr := fmt.Errorf("%s: %w", response.url, response.err)
			errs = append(errs, apiErr)
			apiErrors = append(apiErrors, apiErr.Error())
			continue
		}
		documents = append(documents, response.data)
	}

	if len(documents) == 0 {
		return nil, nil, errors.Join(append([]error{ErrFetchFailed}, errs...)...)
	}

	sort.Strings(apiErrors)
	return documents, apiErrors, nil
}

func (s *ProfileService) fetchHTMLs(ctx context.Context, sourceURLs map[string]string) ([]string, []string) {
	ch := make(chan htmlResponse, len(sourceURLs))
	for key, sourceURL := range sourceURLs {
		go func(key, sourceURL string) {
			document, err := s.fetchHTML(ctx, key, sourceURL)
			ch <- htmlResponse{key: key, url: sourceURL, html: document, err: err}
		}(key, sourceURL)
	}

	var documents []string
	var apiErrors []string
	for range sourceURLs {
		response := <-ch
		if response.err != nil {
			apiErrors = append(apiErrors, fmt.Errorf("%s: %w", response.url, response.err).Error())
			continue
		}
		documents = append(documents, response.html)
	}

	sort.Strings(apiErrors)
	return documents, apiErrors
}

func (s *ProfileService) fetchRSCs(ctx context.Context, sourceURLs map[string]string, publicID, memberID string) ([]string, []string) {
	ch := make(chan rscResponse, len(sourceURLs))
	for key, sourceURL := range sourceURLs {
		go func(key, sourceURL string) {
			document, err := s.fetchRSC(ctx, key, sourceURL, publicID, memberID)
			ch <- rscResponse{key: key, url: sourceURL, data: document, err: err}
		}(key, sourceURL)
	}

	var documents []string
	var apiErrors []string
	for range sourceURLs {
		response := <-ch
		if response.err != nil {
			apiErrors = append(apiErrors, fmt.Errorf("%s: %w", response.url, response.err).Error())
			continue
		}
		documents = append(documents, response.data)
	}

	sort.Strings(apiErrors)
	return documents, apiErrors
}

func (s *ProfileService) fetchAPI(ctx context.Context, apiURL string) (any, error) {
	if err := s.requestPacer.Wait(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.linkedin.normalized+json+2.1, application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("X-RestLi-Protocol-Version", "2.0.0")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")
	if s.cookieHeader != "" {
		req.Header.Set("Cookie", s.cookieHeader)
	}
	if s.csrfToken != "" {
		req.Header.Set("Csrf-Token", s.csrfToken)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("linkedin api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return data, nil
}

func (s *ProfileService) fetchHTML(ctx context.Context, key, sourceURL string) (string, error) {
	if err := s.requestPacer.Wait(ctx); err != nil {
		return "", err
	}

	method := http.MethodGet
	if key == "flagship_certifications" {
		method = http.MethodPost
	}
	req, err := http.NewRequestWithContext(ctx, method, sourceURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/x-component,*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")
	if s.cookieHeader != "" {
		req.Header.Set("Cookie", s.cookieHeader)
	}
	if s.csrfToken != "" {
		req.Header.Set("Csrf-Token", s.csrfToken)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 12<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("linkedin flagship-web returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return string(body), nil
}

func (s *ProfileService) fetchRSC(ctx context.Context, key, sourceURL, publicID, memberID string) (string, error) {
	if err := s.requestPacer.Wait(ctx); err != nil {
		return "", err
	}

	requestBody, referer := rscRequest(key, publicID, memberID)
	body := strings.NewReader(requestBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sourceURL, body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "text/x-component,*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", referer)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")
	if s.cookieHeader != "" {
		req.Header.Set("Cookie", s.cookieHeader)
	}
	if s.csrfToken != "" {
		req.Header.Set("Csrf-Token", s.csrfToken)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 12<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("linkedin flagship-web rsc returned %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return string(responseBody), nil
}

func rscRequest(key, publicID, memberID string) (string, string) {
	escaped := url.PathEscape(publicID)
	switch {
	case key == "flagship_about_rsc":
		return profileComponentRSCRequestBody(publicID, memberID), "https://www.linkedin.com/in/" + escaped + "/"
	case key == "flagship_education_rsc":
		return paginationRSCRequestBody("education", 0, publicID, memberID), "https://www.linkedin.com/in/" + escaped + "/details/education/"
	case strings.HasPrefix(key, "flagship_skills_rsc_"):
		start := 0
		_, _ = fmt.Sscanf(strings.TrimPrefix(key, "flagship_skills_rsc_"), "%d", &start)
		return paginationRSCRequestBody("skills", start, publicID, memberID), "https://www.linkedin.com/in/" + escaped + "/details/skills/"
	default:
		return "{}", "https://www.linkedin.com/in/" + escaped + "/"
	}
}

func paginationRSCRequestBody(section string, start int, publicID, memberID string) string {
	sectionTitle := strings.ToUpper(section[:1]) + section[1:]
	pagerID := "com.linkedin.sdui.pagers.profile.details." + section
	sectionRef := "com.linkedin.sdui.profile.card.ref" + memberID + sectionTitle + "DetailsSection"
	payload := map[string]any{
		"vanityName":                           publicID,
		"profileId":                            memberID,
		"start":                                start,
		"count":                                10,
		"detailSectionReplaceableComponentRef": sectionRef,
	}
	requestedArguments := map[string]any{
		"$type":              "proto.sdui.actions.requests.RequestedArguments",
		"requestedStateKeys": []any{},
		"payload":            payload,
		"requestMetadata": map[string]any{
			"$type": "proto.sdui.common.RequestMetadata",
		},
	}
	request := map[string]any{
		"pagerId": pagerID,
		"clientArguments": map[string]any{
			"$type":              "proto.sdui.actions.requests.RequestedArguments",
			"requestedStateKeys": []any{},
			"payload":            payload,
			"requestMetadata": map[string]any{
				"$type": "proto.sdui.common.RequestMetadata",
			},
			"states":           []any{},
			"screenId":         "com.linkedin.sdui.flagshipnav.profile.Profile" + sectionTitle + "Details",
			"knownTemplateIds": []any{},
		},
		"paginationRequest": map[string]any{
			"$type":   "proto.sdui.actions.requests.PaginationRequest",
			"pagerId": pagerID,
			"trigger": map[string]any{
				"$case": "itemDistanceTrigger",
				"itemDistanceTrigger": map[string]any{
					"$type":           "proto.sdui.actions.requests.ItemDistanceTrigger",
					"preloadDistance": 3,
					"preloadLength":   250,
				},
			},
			"retryCount":         2,
			"requestedArguments": requestedArguments,
		},
	}

	data, err := json.Marshal(request)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func profileComponentRSCRequestBody(publicID, memberID string) string {
	binding := func(name string) map[string]any {
		return map[string]any{
			"type": "com.linkedin.sdui.components.core.BindingImpl",
			"value": map[string]any{
				"key":       "ProfileComponentState" + name + publicID + "ProfileComponentState",
				"namespace": "MemoryNamespace",
			},
		}
	}
	request := map[string]any{
		"clientArguments": map[string]any{
			"payload": map[string]any{
				"isSelfView": false,
				"vanityName": publicID,
				"replaceableSectionArgs": map[string]any{
					"vanityName":                      publicID,
					"hideCardsForGoldenGate":          false,
					"shouldSetupReplaceableComponent": true,
					"vieweeProfileId":                 memberID,
					"isSelfView":                      false,
					"isSelfViewResolved":              false,
				},
				"profileComponentState": map[string]any{
					"profileId":                         publicID,
					"shouldRefreshScreenOnReappear":     binding("ShouldRefreshScreen"),
					"shouldFetchFromCache":              binding("FetchFromCache"),
					"shouldDisplayTabAnchors":           binding("ShouldDisplayTabAnchors"),
					"shouldReloadTopCardOnReappear":     binding("ShouldReloadTopCardOnReappear"),
					"deferredTopCardReloadProfileId":    binding("DeferredTopCardReloadProfileId"),
					"shouldDisplayStickyHeader":         binding("ShouldDisplayStickyHeader"),
					"shouldRefreshLanguageDetailScreen": binding("ShouldRefreshLanguageDetails"),
					"lastPerformedActionRef":            binding("LastPerformedActionRef"),
					"shouldFocusOnReappear":             binding("ShouldFocusOnReappear"),
					"shouldFocusFeaturedOnReappear":     binding("ShouldFocusFeaturedOnReappear"),
					"lastFeaturedActionRef":             binding("LastFeaturedActionRef"),
					"shouldHideProfileCards":            binding("ProfileHideCards"),
				},
			},
			"states": []any{},
			"requestMetadata": map[string]any{
				"$type": "proto.sdui.common.RequestMetadata",
			},
			"screenId":         "com.linkedin.sdui.flagshipnav.profile.Profile",
			"knownTemplateIds": []any{},
		},
	}

	data, err := json.Marshal(request)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func normalizeProfileURL(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", ErrInvalidProfileURL
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "", ErrInvalidProfileURL
	}

	host := strings.ToLower(parsed.Hostname())
	if host != "linkedin.com" && host != "www.linkedin.com" {
		return "", "", ErrInvalidProfileURL
	}

	parts := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	if len(parts) < 2 || parts[0] != "in" || parts[1] == "" {
		return "", "", ErrInvalidProfileURL
	}

	publicID, err := url.PathUnescape(parts[1])
	if err != nil || publicID == "" {
		return "", "", ErrInvalidProfileURL
	}

	baseURL := (&url.URL{
		Scheme: "https",
		Host:   "www.linkedin.com",
		Path:   path.Join("/", "in", publicID) + "/",
	}).String()

	return baseURL, publicID, nil
}

func mergeProfileIdentity(result *ProfileResult, document any) {
	for _, object := range flattenObjects(document) {
		entityURN := stringField(object, "entityUrn")
		if !strings.HasPrefix(entityURN, "urn:li:fsd_profile:") {
			continue
		}

		if result.ProfileURN == "" {
			result.ProfileURN = entityURN
			result.MemberID = strings.TrimPrefix(entityURN, "urn:li:fsd_profile:")
		}
		if result.VersionTag == "" {
			result.VersionTag = stringField(object, "versionTag")
		}
	}
}

func mergeProfile(result *ProfileResult, document any, publicID string) {
	for _, object := range flattenObjects(document) {
		if !isProfileObject(object, publicID) {
			continue
		}

		if result.Name == "" {
			firstName := stringField(object, "firstName")
			lastName := stringField(object, "lastName")
			result.Name = strings.TrimSpace(firstName + " " + lastName)
		}
		if result.Headline == "" {
			result.Headline = firstStringField(object, "headline", "occupation")
		}
		if result.Location == "" {
			result.Location = firstStringField(object, "locationName", "geoLocationName", "geoCountryName")
		}
		if result.About == "" {
			result.About = firstStringField(object, "summary", "about")
		}
	}
}

func mergeImages(result *ProfileResult, document any) {
	walk(document, nil, func(path []string, value any) {
		object, ok := value.(map[string]any)
		if !ok {
			return
		}

		rootURL := stringField(object, "rootUrl")
		if rootURL == "" {
			return
		}

		var artifacts []ImageArtifact
		for _, artifactValue := range arrayField(object, "artifacts") {
			artifact, ok := artifactValue.(map[string]any)
			if !ok {
				continue
			}
			segment := stringField(artifact, "fileIdentifyingUrlPathSegment")
			if segment == "" {
				continue
			}
			artifacts = append(artifacts, ImageArtifact{
				URL:       rootURL + segment,
				Width:     intField(artifact, "width"),
				Height:    intField(artifact, "height"),
				ExpiresAt: int64Field(artifact, "expiresAt"),
			})
		}

		if len(artifacts) == 0 {
			return
		}

		pathText := strings.ToLower(strings.Join(path, ".") + "." + rootURL)
		switch {
		case strings.Contains(pathText, "background") || strings.Contains(pathText, "profile-displaybackgroundimage"):
			result.Images.Background = append(result.Images.Background, artifacts...)
		case strings.Contains(pathText, "picture") || strings.Contains(pathText, "photo") || strings.Contains(pathText, "profile-displayphoto"):
			result.Images.Profile = append(result.Images.Profile, artifacts...)
		}
	})
}

func mergeExperience(result *ProfileResult, document any) {
	for _, object := range flattenObjects(document) {
		typeName := strings.ToLower(stringField(object, "$type"))
		if !strings.Contains(typeName, "position") && stringField(object, "companyName") == "" {
			continue
		}

		item := Experience{
			Title:          firstStringField(object, "title"),
			Company:        firstStringField(object, "companyName"),
			EmploymentType: firstStringField(object, "employmentType"),
			Location:       firstStringField(object, "locationName", "geoLocationName"),
			Description:    firstStringField(object, "description", "summary"),
			DateRange:      timePeriodString(object),
		}
		if item.Company == "" {
			item.Company = nestedName(object, "company")
		}
		item.Skills = namesFromNestedCollection(object, "skills")

		if item.Title != "" || item.Company != "" {
			result.Experience = append(result.Experience, item)
		}
	}
}

func mergeEducation(result *ProfileResult, document any) {
	for _, object := range flattenObjects(document) {
		typeName := strings.ToLower(stringField(object, "$type"))
		if !strings.Contains(typeName, "education") && stringField(object, "schoolName") == "" {
			continue
		}

		item := Education{
			School:       firstStringField(object, "schoolName"),
			Degree:       firstStringField(object, "degreeName"),
			FieldOfStudy: firstStringField(object, "fieldOfStudy"),
			DateRange:    timePeriodString(object),
			Skills:       namesFromNestedCollection(object, "skills"),
		}
		if item.School == "" {
			item.School = nestedName(object, "school")
		}

		if item.School != "" || item.Degree != "" {
			result.Education = append(result.Education, item)
		}
	}
}

func mergeSkills(result *ProfileResult, document any) {
	for _, object := range flattenObjects(document) {
		typeName := strings.ToLower(stringField(object, "$type"))
		entityURN := strings.ToLower(stringField(object, "entityUrn"))
		if !strings.Contains(typeName, "skill") && !strings.Contains(entityURN, "skill") {
			continue
		}

		name := firstStringField(object, "name")
		if name != "" {
			result.Skills = append(result.Skills, Skill{Name: name})
		}
	}
}

func mergeCertifications(result *ProfileResult, document any) {
	for _, object := range flattenObjects(document) {
		typeName := strings.ToLower(stringField(object, "$type"))
		if !strings.Contains(typeName, "certification") && stringField(object, "licenseNumber") == "" {
			continue
		}

		item := Certification{
			Name:         firstStringField(object, "name"),
			Issuer:       firstStringField(object, "authority", "issuer", "companyName"),
			CredentialID: firstStringField(object, "licenseNumber", "credentialId"),
			URL:          firstStringField(object, "url"),
			Skills:       namesFromNestedCollection(object, "skills"),
		}
		if item.Issuer == "" {
			item.Issuer = nestedName(object, "company")
		}
		item.Issued, item.Expires = issueExpireString(object)

		if item.Name != "" || item.CredentialID != "" {
			result.Certifications = append(result.Certifications, item)
		}
	}
}

func mergeLanguages(result *ProfileResult, document any) {
	for _, object := range flattenObjects(document) {
		typeName := strings.ToLower(stringField(object, "$type"))
		if !strings.Contains(typeName, "language") {
			continue
		}

		name := firstStringField(object, "name")
		if name == "" {
			continue
		}

		result.Languages = append(result.Languages, Language{
			Name:        name,
			Proficiency: firstStringField(object, "proficiency"),
		})
	}
}

func isProfileObject(object map[string]any, publicID string) bool {
	if strings.EqualFold(stringField(object, "publicIdentifier"), publicID) {
		return true
	}
	typeName := strings.ToLower(stringField(object, "$type"))
	return strings.Contains(typeName, ".profile") &&
		(stringField(object, "firstName") != "" || stringField(object, "headline") != "" || stringField(object, "summary") != "")
}

func flattenObjects(document any) []map[string]any {
	var objects []map[string]any
	walk(document, nil, func(_ []string, value any) {
		if object, ok := value.(map[string]any); ok {
			objects = append(objects, object)
		}
	})
	return objects
}

func walk(value any, currentPath []string, visit func([]string, any)) {
	visit(currentPath, value)

	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			walk(child, append(currentPath, key), visit)
		}
	case []any:
		for _, child := range typed {
			walk(child, currentPath, visit)
		}
	}
}

func firstStringField(object map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringField(object, key); value != "" {
			return value
		}
	}
	return ""
}

func stringField(object map[string]any, key string) string {
	value, ok := object[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		return textFromObject(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func textFromObject(object map[string]any) string {
	for _, key := range []string{"text", "name", "localizedName"} {
		if value, ok := object[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func arrayField(object map[string]any, key string) []any {
	value, ok := object[key].([]any)
	if !ok {
		return nil
	}
	return value
}

func intField(object map[string]any, key string) int {
	switch value := object[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	default:
		return 0
	}
}

func int64Field(object map[string]any, key string) int64 {
	switch value := object[key].(type) {
	case float64:
		return int64(value)
	case int64:
		return value
	case int:
		return int64(value)
	default:
		return 0
	}
}

func nestedName(object map[string]any, key string) string {
	nested, ok := object[key].(map[string]any)
	if !ok {
		return ""
	}
	return firstStringField(nested, "name", "schoolName", "companyName")
}

func namesFromNestedCollection(object map[string]any, key string) []string {
	var names []string
	walk(object[key], nil, func(_ []string, value any) {
		nested, ok := value.(map[string]any)
		if !ok {
			return
		}
		if name := firstStringField(nested, "name"); name != "" {
			names = append(names, name)
		}
	})
	return dedupeStrings(names)
}

func timePeriodString(object map[string]any) string {
	timePeriod, ok := object["timePeriod"].(map[string]any)
	if !ok {
		return ""
	}
	start := dateString(timePeriod["startDate"])
	end := dateString(timePeriod["endDate"])
	switch {
	case start != "" && end != "":
		return start + " - " + end
	case start != "":
		return start + " - Present"
	default:
		return ""
	}
}

func issueExpireString(object map[string]any) (string, string) {
	return dateString(object["issueDate"]), dateString(object["expirationDate"])
}

func dateString(value any) string {
	object, ok := value.(map[string]any)
	if !ok {
		return ""
	}

	year := intField(object, "year")
	month := intField(object, "month")
	if year == 0 {
		return ""
	}
	if month == 0 {
		return fmt.Sprintf("%04d", year)
	}
	return fmt.Sprintf("%04d-%02d", year, month)
}

func dedupeExperience(items []Experience) []Experience {
	seen := map[string]bool{}
	var out []Experience
	for _, item := range items {
		key := strings.ToLower(item.Title + "|" + item.Company + "|" + item.DateRange)
		if key == "||" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}

func dedupeEducation(items []Education) []Education {
	seen := map[string]bool{}
	var out []Education
	for _, item := range items {
		key := strings.ToLower(item.School + "|" + item.Degree + "|" + item.DateRange)
		if key == "||" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}

func dedupeSkills(items []Skill) []Skill {
	seen := map[string]bool{}
	var out []Skill
	for _, item := range items {
		key := strings.ToLower(item.Name)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}

func dedupeCertifications(items []Certification) []Certification {
	seen := map[string]bool{}
	var out []Certification
	for _, item := range items {
		key := strings.ToLower(item.Name + "|" + item.Issuer + "|" + item.CredentialID)
		if key == "||" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}

func dedupeLanguages(items []Language) []Language {
	seen := map[string]bool{}
	var out []Language
	for _, item := range items {
		key := strings.ToLower(item.Name)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}

func dedupeImages(items []ImageArtifact) []ImageArtifact {
	seen := map[string]bool{}
	var out []ImageArtifact
	for _, item := range items {
		if item.URL == "" || seen[item.URL] {
			continue
		}
		seen[item.URL] = true
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Width == out[j].Width {
			return out[i].URL < out[j].URL
		}
		return out[i].Width < out[j].Width
	})
	return out
}

func dedupeStrings(items []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		key := strings.ToLower(item)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}

func missingFields(result *ProfileResult) []string {
	var missing []string
	if result.Name == "" {
		missing = append(missing, "name")
	}
	if result.Headline == "" {
		missing = append(missing, "headline")
	}
	if result.Location == "" {
		missing = append(missing, "location")
	}
	if result.About == "" {
		missing = append(missing, "about")
	}
	if len(result.Experience) == 0 {
		missing = append(missing, "experience")
	}
	if len(result.Education) == 0 {
		missing = append(missing, "education")
	}
	if len(result.Skills) == 0 {
		missing = append(missing, "skills")
	}
	if len(result.Certifications) == 0 {
		missing = append(missing, "certifications")
	}
	if len(result.Languages) == 0 {
		missing = append(missing, "languages")
	}
	if len(result.Images.Profile) == 0 {
		missing = append(missing, "profile_images")
	}
	return missing
}

func loadLinkedInCookieHeader() string {
	if header := strings.TrimSpace(os.Getenv("LINKEDIN_COOKIE_HEADER")); header != "" {
		return header
	}

	var cookies []string
	if liAt := strings.TrimSpace(os.Getenv("LINKEDIN_LI_AT")); liAt != "" {
		cookies = append(cookies, "li_at="+liAt)
	}
	if jsessionID := strings.TrimSpace(os.Getenv("LINKEDIN_JSESSIONID")); jsessionID != "" {
		cookies = append(cookies, "JSESSIONID="+jsessionID)
	}
	return strings.Join(cookies, "; ")
}

func loadLinkedInCSRFToken(cookieHeader string) string {
	if token := strings.TrimSpace(os.Getenv("LINKEDIN_CSRF_TOKEN")); token != "" {
		return token
	}

	for _, cookie := range strings.Split(cookieHeader, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(cookie), "=")
		if !ok || name != "JSESSIONID" {
			continue
		}
		return strings.Trim(value, `"`)
	}
	return ""
}
