package services

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestSingleRSCSourceURLsOnlyIncludeNonPaginatedSections(t *testing.T) {
	sourceURLs := singleRSCSourceURLsForMissing([]string{"about", "experience", "education", "skills", "languages"})

	for _, key := range []string{
		"flagship_about_rsc",
		"flagship_languages_rsc_0",
	} {
		if sourceURLs[key] == "" {
			t.Fatalf("%s source URL is missing", key)
		}
	}
	for _, key := range []string{
		"flagship_experience_rsc_0",
		"flagship_education_rsc_0",
		"flagship_skills_rsc_0",
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

func TestMergeAllProfileItemsFromRSCPaginatesUntilEmpty(t *testing.T) {
	tests := []struct {
		name       string
		section    string
		response   string
		resultSize func(*ProfileResult) int
	}{
		{
			name:    "experience",
			section: "experience",
			response: `0:["$","div",null,{"children":[["$","$L3",null,{"componentKey":"entity-collection-item-one","children":["$","div",null,{"children":["$L1"]}]}]]}]
1:["$","div",null,{"children":[["$","p",null,{"children":["Engineer"]}],["$","p",null,{"children":["Acme · Full-time"]}],["$","$Ltext",null,{"textProps":{"children":["Jan 2024 - Present"]}}]]}]`,
			resultSize: func(result *ProfileResult) int { return len(result.Experience) },
		},
		{
			name:    "education",
			section: "education",
			response: `0:["$","div",null,{"children":[["$","$L3",null,{"componentKey":"education-one","viewTrackingSpecs":{"viewName":"education-lockup-view"},"children":["$","div",null,{"children":["$L1"]}]}]]}]
1:["$","div",null,{"children":[["$","p",null,{"children":["State University"]}],["$","p",null,{"children":["Computer Science"]}],["$","$Ltext",null,{"textProps":{"children":["2020 - 2024"]}}]]}]`,
			resultSize: func(result *ProfileResult) int { return len(result.Education) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responses := []string{
				tt.response,
				`0:["$","div",null,{"children":["Nothing to see for now"]}]`,
			}
			var starts []int
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					ClientArguments struct {
						Payload struct {
							Start int `json:"start"`
						} `json:"payload"`
					} `json:"clientArguments"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Fatal(err)
				}
				starts = append(starts, request.ClientArguments.Payload.Start)
				_, _ = w.Write([]byte(responses[requestCount]))
				requestCount++
			}))
			defer server.Close()

			service := NewProfileServiceWithConfig(ProfileServiceConfig{Client: server.Client(), RequestMinInterval: -1})
			result := &ProfileResult{}
			sourceURLs, apiErrors := service.mergeAllProfileItemsFromRSC(
				context.Background(), result, "profile", "member", tt.section, server.URL,
			)

			if len(apiErrors) != 0 {
				t.Fatalf("api errors = %#v", apiErrors)
			}
			if !reflect.DeepEqual(starts, []int{0, 10}) {
				t.Fatalf("starts = %#v", starts)
			}
			if tt.resultSize(result) != 1 {
				t.Fatalf("result = %#v", result)
			}
			for _, start := range starts {
				key := fmt.Sprintf("flagship_%s_rsc_%d", tt.section, start)
				if sourceURLs[key] != server.URL {
					t.Fatalf("source URL %s = %q", key, sourceURLs[key])
				}
			}
		})
	}
}
