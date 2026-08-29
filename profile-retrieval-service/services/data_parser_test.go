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
