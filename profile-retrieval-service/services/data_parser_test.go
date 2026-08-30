package services

import (
	"reflect"
	"strings"
	"testing"
)

func TestProfileDataParserMergeProfileAndImages(t *testing.T) {
	document := `
		<html>
			<head>
				<title>Rithik Kadali | LinkedIn</title>
				<link rel="preload" as="image" imageSrcSet="https://media.licdn.com/dms/image/v2/background/profile-displaybackgroundimage-shrink_200_800/photo?e=1 800w, https://media.licdn.com/dms/image/v2/background/profile-displaybackgroundimage-shrink_350_1400/photo?e=1 1400w"/>
				<link rel="preload" as="image" imageSrcSet="https://media.licdn.com/dms/image/v2/profile/profile-displayphoto-scale_100_100/photo?e=1 100w, https://media.licdn.com/dms/image/v2/profile/profile-displayphoto-scale_400_400/photo?e=1 400w"/>
			</head>
			<body>
				<nav>Home</nav>
				<div>Rithik Kadali</div>
				<div>Generative AI Engineer | Python Developer | Software Engineer @ Wipro</div>
				<div>Wipro · JNTU Gurajada Vizianagaram</div>
				<div>Bengaluru, Karnataka, India</div>
				<div>Contact info</div>
				<a href="/messaging/thread/new/?recipient=ACoAAC_PkyoBrlR8cFB_T-IAPBuUNqzNeM6r7h8">Message</a>
			</body>
		</html>`

	result := &ProfileResult{}
	NewProfileDataParser().Merge(result, document, "rithik-kadali-2627711a4")

	if result.Name != "Rithik Kadali" {
		t.Fatalf("Name = %q", result.Name)
	}
	if result.Headline != "Generative AI Engineer | Python Developer | Software Engineer @ Wipro" {
		t.Fatalf("Headline = %q", result.Headline)
	}
	if result.Location != "Bengaluru, Karnataka, India" {
		t.Fatalf("Location = %q", result.Location)
	}
	if result.MemberID != "ACoAAC_PkyoBrlR8cFB_T-IAPBuUNqzNeM6r7h8" {
		t.Fatalf("MemberID = %q", result.MemberID)
	}
	if result.ProfileURN != "urn:li:fsd_profile:ACoAAC_PkyoBrlR8cFB_T-IAPBuUNqzNeM6r7h8" {
		t.Fatalf("ProfileURN = %q", result.ProfileURN)
	}
	if len(result.Images.Profile) != 2 {
		t.Fatalf("profile images length = %d", len(result.Images.Profile))
	}
	if len(result.Images.Background) != 2 {
		t.Fatalf("background images length = %d", len(result.Images.Background))
	}
	if result.Images.Profile[1].Width != 400 {
		t.Fatalf("profile image width = %d", result.Images.Profile[1].Width)
	}
}

func TestProfileDataParserMergeExperience(t *testing.T) {
	document := `
		<html><body>
			<section>
				<h2>Experience</h2>
				<div>Project Engineer</div>
				<div>Wipro · Full-time</div>
				<div>Mar 2023 - Present · 3 yrs 6 mos</div>
				<div>Bengaluru, Karnataka, India · Hybrid</div>
				<p>Built a KPI Analysis Chatbot using RAG and Agent AI.</p>
				<p>Developed scalable APIs with Python.</p>
				<div>Jenkins, Generative AI and +28 skills</div>
				<div>Ad Options</div>
			</section>
		</body></html>`

	result := &ProfileResult{}
	NewProfileDataParser().Merge(result, document, "rithik-kadali-2627711a4")

	if len(result.Experience) != 1 {
		t.Fatalf("experience length = %d", len(result.Experience))
	}
	item := result.Experience[0]
	if item.Title != "Project Engineer" {
		t.Fatalf("Title = %q", item.Title)
	}
	if item.Company != "Wipro" {
		t.Fatalf("Company = %q", item.Company)
	}
	if item.EmploymentType != "Full-time" {
		t.Fatalf("EmploymentType = %q", item.EmploymentType)
	}
	if item.DateRange != "Mar 2023 - Present · 3 yrs 6 mos" {
		t.Fatalf("DateRange = %q", item.DateRange)
	}
	if item.Location != "Bengaluru, Karnataka, India · Hybrid" {
		t.Fatalf("Location = %q", item.Location)
	}
	if !strings.Contains(item.Description, "Built a KPI Analysis Chatbot") {
		t.Fatalf("Description = %q", item.Description)
	}
	if !reflect.DeepEqual(item.Skills, []string{"Jenkins", "Generative AI"}) {
		t.Fatalf("Skills = %#v", item.Skills)
	}
}

func TestProfileDataParserMergeMultipleExperiences(t *testing.T) {
	document := `
		<html><body>
			<section>
				<h2>Experience</h2>
				<div>Junior Software Engineer</div>
				<div>H&amp;M · Full-time</div>
				<div>Apr 2026 - Present · 5 mos</div>
				<div>Bengaluru, Karnataka, India · On-site</div>
				<div>Software Engineer</div>
				<div>Apxor · Full-time</div>
				<div>Aug 2025 - Mar 2026 · 8 mos</div>
				<div>Hyderabad, Telangana, India · On-site</div>
				<div>Contributor</div>
				<div>GirlScript Summer of Code · Part-time</div>
				<div>May 2024 - Aug 2024 · 4 mos</div>
				<div>Gssoc'24 Contributor</div>
				<div>Champion badge</div>
				<div>Android Developer</div>
				<div>Incrivelsoft LLC. · Internship</div>
				<div>May 2024 - Jul 2024 · 3 mos</div>
				<div>Remote</div>
				<div>Java</div>
				<div>Ad Options</div>
			</section>
		</body></html>`

	result := &ProfileResult{}
	NewProfileDataParser().Merge(result, document, "")

	if len(result.Experience) != 4 {
		t.Fatalf("experience length = %d: %#v", len(result.Experience), result.Experience)
	}
	expected := []Experience{
		{
			Title:          "Junior Software Engineer",
			Company:        "H&M",
			EmploymentType: "Full-time",
			DateRange:      "Apr 2026 - Present · 5 mos",
			Location:       "Bengaluru, Karnataka, India · On-site",
		},
		{
			Title:          "Software Engineer",
			Company:        "Apxor",
			EmploymentType: "Full-time",
			DateRange:      "Aug 2025 - Mar 2026 · 8 mos",
			Location:       "Hyderabad, Telangana, India · On-site",
		},
		{
			Title:          "Contributor",
			Company:        "GirlScript Summer of Code",
			EmploymentType: "Part-time",
			DateRange:      "May 2024 - Aug 2024 · 4 mos",
			Description:    "Gssoc'24 Contributor\nChampion badge",
		},
		{
			Title:          "Android Developer",
			Company:        "Incrivelsoft LLC.",
			EmploymentType: "Internship",
			DateRange:      "May 2024 - Jul 2024 · 3 mos",
			Location:       "Remote",
			Description:    "Java",
		},
	}
	if !reflect.DeepEqual(result.Experience, expected) {
		t.Fatalf("experience = %#v, want %#v", result.Experience, expected)
	}
}

func TestProfileDataParserMergeExperienceWithoutEmploymentType(t *testing.T) {
	document := `
		<html><body>
			<section>
				<h2>Experience</h2>
				<div>Software Engineer</div>
				<div>Acme Labs</div>
				<div>Jan 2025 - Present</div>
				<div>Built internal developer tools.</div>
				<div>Contributor</div>
				<div>Open Source Community</div>
				<div>May 2024 - Dec 2024</div>
				<div>Remote</div>
				<div>Maintained Go libraries.</div>
				<div>Ad Options</div>
			</section>
		</body></html>`

	result := &ProfileResult{}
	NewProfileDataParser().Merge(result, document, "")

	expected := []Experience{
		{
			Title:       "Software Engineer",
			Company:     "Acme Labs",
			DateRange:   "Jan 2025 - Present",
			Description: "Built internal developer tools.",
		},
		{
			Title:       "Contributor",
			Company:     "Open Source Community",
			DateRange:   "May 2024 - Dec 2024",
			Location:    "Remote",
			Description: "Maintained Go libraries.",
		},
	}
	if !reflect.DeepEqual(result.Experience, expected) {
		t.Fatalf("experience = %#v, want %#v", result.Experience, expected)
	}
}

func TestProfileDataParserMergeMultipleRolesAtSameCompany(t *testing.T) {
	document := `
		<html><body>
			<section>
				<h2>Experience</h2>
				<div>Quality Assurance Automation Engineer</div>
				<div>Infor · Full-time</div>
				<div>Feb 2024 - Present · 2 yrs 7 mos</div>
				<div>Hyderabad, Telangana, India · Hybrid</div>
				<div>QA Automation, Playwright and +3 skills</div>
				<div>Ivy</div>
				<div>Full-time · 1 yr 6 mos</div>
				<div>Test Engineer</div>
				<div>Aug 2023 - Jan 2024 · 6 mos</div>
				<div>Hybrid</div>
				<div>Skills:</div>
				<div>QA Automation, Quality Control, +16 skills</div>
				<div>Trainee Test Engineer</div>
				<div>Aug 2022 - Aug 2023 · 1 yr 1 mo</div>
				<div>Hyderabad, Telangana, India</div>
				<div>Skills:</div>
				<div>User Experience (UX), Quality Assurance Testing, +4 skills</div>
				<div>Ad Options</div>
			</section>
		</body></html>`

	result := &ProfileResult{}
	NewProfileDataParser().Merge(result, document, "pavan-sai-potnuru-8aa709172")

	expected := []Experience{
		{
			Title:          "Quality Assurance Automation Engineer",
			Company:        "Infor",
			EmploymentType: "Full-time",
			DateRange:      "Feb 2024 - Present · 2 yrs 7 mos",
			Location:       "Hyderabad, Telangana, India · Hybrid",
			Skills:         []string{"QA Automation", "Playwright"},
		},
		{
			Title:          "Test Engineer",
			Company:        "Ivy",
			EmploymentType: "Full-time",
			DateRange:      "Aug 2023 - Jan 2024 · 6 mos",
			Location:       "Hybrid",
			Skills:         []string{"QA Automation", "Quality Control"},
		},
		{
			Title:          "Trainee Test Engineer",
			Company:        "Ivy",
			EmploymentType: "Full-time",
			DateRange:      "Aug 2022 - Aug 2023 · 1 yr 1 mo",
			Location:       "Hyderabad, Telangana, India",
			Skills:         []string{"User Experience (UX)", "Quality Assurance Testing"},
		},
	}
	if !reflect.DeepEqual(result.Experience, expected) {
		t.Fatalf("experience = %#v, want %#v", result.Experience, expected)
	}
}

func TestProfileDataParserUsesListItemsForGroupedRoles(t *testing.T) {
	document := `
		<html><body>
			<h2>Experience</h2>
			<ul><li>
				<div>Cognizant</div>
				<div>3 yrs 3 mos</div>
				<ul>
					<li><p>Associate</p><p>Full-time</p><p>Jan 2026 - Present · 8 mos</p><p>Hyderabad, Telangana, India</p></li>
					<li><p>Performance Test Engineer</p><p>Jun 2023 - Present · 3 yrs 3 mos</p><p>Led performance testing initiatives.</p></li>
					<li><p>Program Analyst</p><p>Full-time</p><p>Jul 2024 - Dec 2025 · 1 yr 6 mos</p><p>Hyderabad, Telangana, India · Hybrid</p></li>
					<li><p>Programmer Analyst Trainee</p><p>Full-time</p><p>Jun 2023 - Jun 2024 · 1 yr 1 mo</p><p>Hyderabad, Telangana, India · Hybrid</p><p>Skills:</p><p>HP Performance Center, LoadRunner, +1 skill</p></li>
				</ul>
			</li></ul>
			<div>Ad Options</div>
		</body></html>`

	result := &ProfileResult{}
	NewProfileDataParser().Merge(result, document, "sureshramana219")

	expected := []Experience{
		{
			Title:          "Associate",
			Company:        "Cognizant",
			EmploymentType: "Full-time",
			DateRange:      "Jan 2026 - Present · 8 mos",
			Location:       "Hyderabad, Telangana, India",
		},
		{
			Title:          "Performance Test Engineer",
			Company:        "Cognizant",
			EmploymentType: "Full-time",
			DateRange:      "Jun 2023 - Present · 3 yrs 3 mos",
			Description:    "Led performance testing initiatives.",
		},
		{
			Title:          "Program Analyst",
			Company:        "Cognizant",
			EmploymentType: "Full-time",
			DateRange:      "Jul 2024 - Dec 2025 · 1 yr 6 mos",
			Location:       "Hyderabad, Telangana, India · Hybrid",
		},
		{
			Title:          "Programmer Analyst Trainee",
			Company:        "Cognizant",
			EmploymentType: "Full-time",
			DateRange:      "Jun 2023 - Jun 2024 · 1 yr 1 mo",
			Location:       "Hyderabad, Telangana, India · Hybrid",
			Skills:         []string{"HP Performance Center", "LoadRunner"},
		},
	}
	if !reflect.DeepEqual(result.Experience, expected) {
		t.Fatalf("experience = %#v, want %#v", result.Experience, expected)
	}
}

func TestProfileDataParserMergeEducationAndSkills(t *testing.T) {
	document := `
		<html><body>
			<section>
				<h2>Education</h2>
				<div>JNTU Gurajada Vizianagaram</div>
				<div>Bachelor of Technology - BTech</div>
				<div>Computer Science</div>
				<div>2018 - 2022</div>
				<div>Python, Data Structures and +3 skills</div>
				<div>Skills</div>
				<div>All</div>
				<div>Industry Knowledge</div>
				<div>Interpersonal Skills</div>
				<div>Python</div>
				<div>LangChain</div>
				<div>People also viewed</div>
			</section>
		</body></html>`

	result := &ProfileResult{}
	NewProfileDataParser().Merge(result, document, "rithik-kadali-2627711a4")

	if len(result.Education) != 1 {
		t.Fatalf("education length = %d", len(result.Education))
	}
	education := result.Education[0]
	if education.School != "JNTU Gurajada Vizianagaram" {
		t.Fatalf("School = %q", education.School)
	}
	if education.Degree != "Bachelor of Technology - BTech" {
		t.Fatalf("Degree = %q", education.Degree)
	}
	if education.FieldOfStudy != "Computer Science" {
		t.Fatalf("FieldOfStudy = %q", education.FieldOfStudy)
	}
	if education.DateRange != "2018 - 2022" {
		t.Fatalf("DateRange = %q", education.DateRange)
	}
	if !reflect.DeepEqual(education.Skills, []string{"Python", "Data Structures"}) {
		t.Fatalf("Education skills = %#v", education.Skills)
	}
	if !reflect.DeepEqual(result.Skills, []Skill{{Name: "Python"}, {Name: "LangChain"}}) {
		t.Fatalf("Skills = %#v", result.Skills)
	}
}

func TestProfileDataParserMergeRSCEducation(t *testing.T) {
	document := `
		11:["$","p",null,{"children":["JNTU Gurajada Vizianagaram"]}]
		12:["$","p",null,{"children":["Bachelor of Technology - BTech, Computer Science"]}]
		13:["$","$Ld",null,{"textProps":{"children":["Jun 2019 – Apr 2022"]}}]
		14:["$","p",null,{"children":[["$","$1a","text-attr-0",{"children":["$undefined",["$","strong",null,{"children":["Skills:"]}],[["$","span",null,{"children":" "}],"SQL, Java, +2 skills",null]]}]]}]
		16:["$","p",null,{"children":["State Board of Technical Education and Training (SBTET), Andhra Pradesh"]}]
		17:["$","p",null,{"children":["Diploma of Education, Computer Science"]}]
		18:["$","$Ld",null,{"textProps":{"children":["Jun 2016 – Apr 2019"]}}]
		19:["$","p",null,{"children":[["$","$1a","text-attr-0",{"children":["$undefined",["$","strong",null,{"children":["Skills:"]}],[["$","span",null,{"children":" "}],"SQL, Java, +2 skills",null]]}]]}]
	`

	result := &ProfileResult{}
	NewProfileDataParser().Merge(result, document, "rithik-kadali-2627711a4")

	if len(result.Education) != 2 {
		t.Fatalf("education length = %d %#v", len(result.Education), result.Education)
	}
	first := result.Education[0]
	if first.School != "JNTU Gurajada Vizianagaram" {
		t.Fatalf("first school = %q", first.School)
	}
	if first.Degree != "Bachelor of Technology - BTech" {
		t.Fatalf("first degree = %q", first.Degree)
	}
	if first.FieldOfStudy != "Computer Science" {
		t.Fatalf("first field = %q", first.FieldOfStudy)
	}
	if first.DateRange != "Jun 2019 - Apr 2022" {
		t.Fatalf("first date range = %q", first.DateRange)
	}
	if !reflect.DeepEqual(first.Skills, []string{"SQL", "Java"}) {
		t.Fatalf("first skills = %#v", first.Skills)
	}
}

func TestProfileDataParserMergeRSCAbout(t *testing.T) {
	document := `
		9:["$","$Le",null,{"textProps":{"children":["About"]}}]
		a:["$","$L14",null,{"textProps":{"fontSize":"small","children":[
			[["$","$15","0",{"children":[null,"Software Professional at Wipro Limited with hands-on experience across Python, Automation, GenAI, and Security, working on large-scale enterprise projects for Microsoft and Ericsson."]}]],
			["$","$15","1",{"children":[["$","br",null,{}],""]}],
			["$","$15","2",{"children":[["$","br",null,{}],"I’ve contributed to Ericsson’s GenAI initiatives, where I helped build a KPI Analysis Chatbot using RAG and Agent AI."]}]
		]}}]
	`

	result := &ProfileResult{}
	NewProfileDataParser().Merge(result, document, "rithik-kadali-2627711a4")

	if !strings.Contains(result.About, "Software Professional at Wipro Limited") {
		t.Fatalf("About = %q", result.About)
	}
	if !strings.Contains(result.About, "KPI Analysis Chatbot") {
		t.Fatalf("About = %q", result.About)
	}
}

func TestProfileDataParserMergeRSCExpandableAbout(t *testing.T) {
	document := `
		9:["$","$Le",null,{"textProps":{"children":["About"]}}]
		10:["$","span",null,{"children":["Jennifer started a new position as Junior Software Engineer at H&M"]}]
		11:["$","$L1a",null,{"textProps":{"fontFamily":"sans","fontSize":"small","children":["Student of Computer Science and Engeenering at JNTU-GV, UCEV"],"linkColorTokens":"$undefined","lineClamp":3,"hasShowMore":false,"expandButtonText":"more","shouldCollapseNewLines":false}}]
	`

	result := &ProfileResult{}
	NewProfileDataParser().MergeRSC(result, "flagship_about_rsc", document, "jennifer-eunice-64a517230")

	if result.About != "Student of Computer Science and Engeenering at JNTU-GV, UCEV" {
		t.Fatalf("About = %q", result.About)
	}
	if strings.Contains(result.About, "started a new position") {
		t.Fatalf("About includes highlight text: %q", result.About)
	}
}

func TestProfileDataParserMergeRSCSkills(t *testing.T) {
	document := `
		"aria-label":"Endorse Red Hat Enterprise Linux (RHEL)"
		"children":["Red Hat Enterprise Linux (RHEL)"]
		"aria-label":"Endorse pytest"
		"children":["pytest"]
		"aria-label":"Endorse Jenkins"
		"children":["Jenkins"]
	`

	result := &ProfileResult{}
	NewProfileDataParser().Merge(result, document, "rithik-kadali-2627711a4")

	expected := []Skill{
		{Name: "Red Hat Enterprise Linux (RHEL)"},
		{Name: "pytest"},
		{Name: "Jenkins"},
	}
	if !reflect.DeepEqual(result.Skills, expected) {
		t.Fatalf("Skills = %#v", result.Skills)
	}
}

func TestProfileDataParserMergeCertifications(t *testing.T) {
	document := `
		<html><body>
			<section>
				<h2>Licenses &amp; certifications</h2>
				<div>Microsoft Certified: Azure AI Engineer Associate</div>
				<div>Microsoft</div>
				<div>Issued Jul 2024 · Expires Jul 2025</div>
				<div>Credential ID ABC-123</div>
				<div>Skills: Azure AI, Python and +2 skills</div>
				<div>AWS Academy Graduate - AWS Academy Cloud Foundations</div>
				<div>Issued Oct 2021</div>
				<div>People also viewed</div>
			</section>
		</body></html>`

	result := &ProfileResult{}
	NewProfileDataParser().Merge(result, document, "rithik-kadali-2627711a4")

	if len(result.Certifications) != 2 {
		t.Fatalf("certifications length = %d %#v", len(result.Certifications), result.Certifications)
	}
	item := result.Certifications[0]
	if item.Name != "Microsoft Certified: Azure AI Engineer Associate" || item.Issuer != "Microsoft" {
		t.Fatalf("certification = %#v", item)
	}
	if item.Issued != "Jul 2024" || item.Expires != "Jul 2025" || item.CredentialID != "ABC-123" {
		t.Fatalf("certification metadata = %#v", item)
	}
	if !reflect.DeepEqual(item.Skills, []string{"Azure AI", "Python"}) {
		t.Fatalf("certification skills = %#v", item.Skills)
	}
	if result.Certifications[1].Name != "AWS Academy Graduate - AWS Academy Cloud Foundations" {
		t.Fatalf("second certification = %#v", result.Certifications[1])
	}
}

func TestProfileDataParserMergeLanguages(t *testing.T) {
	document := `
		<html><body>
			<section>
				<h2>Languages</h2>
				<div>English</div>
				<div>Full professional proficiency</div>
				<div>Telugu</div>
				<div>Native or bilingual proficiency</div>
				<div>People also viewed</div>
			</section>
		</body></html>`

	result := &ProfileResult{}
	NewProfileDataParser().Merge(result, document, "rithik-kadali-2627711a4")

	expected := []Language{
		{Name: "English", Proficiency: "Full professional proficiency"},
		{Name: "Telugu", Proficiency: "Native or bilingual proficiency"},
	}
	if !reflect.DeepEqual(result.Languages, expected) {
		t.Fatalf("languages = %#v", result.Languages)
	}
}

func TestProfileDataParserMergeRSCOnlyTouchesRequestedSection(t *testing.T) {
	document := `
		1:["$","p",null,{"children":["Languages"]}]
		2:["$","p",null,{"children":["Nothing to see for now"]}]
	`

	result := &ProfileResult{}
	NewProfileDataParser().MergeRSC(
		result,
		"flagship_languages_rsc_0",
		document,
		"rithik-kadali-2627711a4",
	)

	if len(result.Education) != 0 {
		t.Fatalf("language RSC created education entries: %#v", result.Education)
	}
	if len(result.Languages) != 0 {
		t.Fatalf("empty language RSC created language entries: %#v", result.Languages)
	}
}
