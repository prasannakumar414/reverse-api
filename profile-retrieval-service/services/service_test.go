package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestHTMLSourceURLsIncludeAllDetailPages(t *testing.T) {
	sourceURLs := htmlSourceURLsFor("rithik-kadali-2627711a4")

	for _, key := range []string{
		"flagship_profile_html",
		"flagship_experience_html",
		"flagship_education_html",
		"flagship_skills_html",
		"flagship_languages_html",
		"flagship_certifications",
		"flagship_certifications_html",
	} {
		if sourceURLs[key] == "" {
			t.Fatalf("%s source URL is missing", key)
		}
	}
}

func TestRSCSourceURLsOnlyIncludeMissingSections(t *testing.T) {
	sourceURLs := rscSourceURLsForMissing([]string{"about", "skills", "languages"})

	for _, key := range []string{
		"flagship_about_rsc",
		"flagship_skills_rsc_0",
		"flagship_languages_rsc_0",
	} {
		if sourceURLs[key] == "" {
			t.Fatalf("%s source URL is missing", key)
		}
	}
	for _, key := range []string{
		"flagship_experience_rsc_0",
		"flagship_education_rsc_0",
		"flagship_certifications_rsc_0",
	} {
		if sourceURLs[key] != "" {
			t.Fatalf("%s should not be requested", key)
		}
	}
}

func TestRSCRequestSupportsDetailSections(t *testing.T) {
	body, referer := rscRequest(
		"flagship_certifications_rsc_0",
		"rithik-kadali-2627711a4",
		"ACoAAC_PkyoBrlR8cFB_T-IAPBuUNqzNeM6r7h8",
	)

	if body == "{}" {
		t.Fatal("RSC request body was not generated")
	}
	if referer != "https://www.linkedin.com/in/rithik-kadali-2627711a4/details/certifications/" {
		t.Fatalf("referer = %q", referer)
	}
}

func TestMergeAllSkillsFromRSCPaginatesUntilEmpty(t *testing.T) {
	responses := []string{
		`"aria-label":"Endorse React.js" "aria-label":"Endorse JavaScript"`,
		`"aria-label":"Endorse Go"`,
		`"children":["Nothing to see for now"]`,
	}
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if requestCount >= len(responses) {
			t.Fatalf("unexpected request %d", requestCount)
		}
		_, _ = w.Write([]byte(responses[requestCount]))
		requestCount++
	}))
	defer server.Close()

	service := NewProfileServiceWithConfig(ProfileServiceConfig{
		Client:             server.Client(),
		RequestMinInterval: -1,
	})
	result := &ProfileResult{Skills: []Skill{{Name: "Interpersonal Skills"}}}
	sourceURLs, apiErrors := service.mergeAllSkillsFromRSC(
		context.Background(),
		result,
		"jennifer-eunice-64a517230",
		"ACoAADm9v9oBYTVjIgLo5hV_c2Gavg4S6muKTVw",
		server.URL,
	)

	if len(apiErrors) != 0 {
		t.Fatalf("api errors = %#v", apiErrors)
	}
	if requestCount != 3 {
		t.Fatalf("request count = %d", requestCount)
	}
	for _, key := range []string{"flagship_skills_rsc_0", "flagship_skills_rsc_10", "flagship_skills_rsc_20"} {
		if sourceURLs[key] != server.URL {
			t.Fatalf("source URL %s = %q", key, sourceURLs[key])
		}
	}
	expected := []Skill{
		{Name: "React.js"},
		{Name: "JavaScript"},
		{Name: "Go"},
	}
	if !reflect.DeepEqual(result.Skills, expected) {
		t.Fatalf("skills = %#v, want %#v", result.Skills, expected)
	}
}
