# Professional Network UI/UX Design Guidelines

The application should feel like a modern professional networking product inspired by the information density and interaction patterns of LinkedIn, without attempting to reproduce LinkedIn pixel-for-pixel.

## 1. Design Philosophy

Prioritize:

* Professional over playful
* Information density without clutter
* Clear content hierarchy
* Familiar social-network interaction patterns
* Fast scanning
* Minimal visual decoration
* Accessibility
* Responsive layouts
* Consistency through reusable components

Avoid:

* Excessive gradients
* Glassmorphism
* Huge rounded cards
* Oversized typography
* Heavy animations
* Excessive shadows
* Dashboard-like neon styling
* Decorative elements that distract from profile information

The product should feel calm, credible, information-rich, and utilitarian.

---

# 2. Layout

Use a centered desktop container:

```text
max-width: 1128–1200px
margin: auto
padding-inline: 16–24px
```

Typical desktop layout:

```text
┌──────────────────────────────────────────────┐
│               Global Navbar                  │
└──────────────────────────────────────────────┘

┌─────────────┬───────────────────┬─────────────┐
│             │                   │             │
│ Left Rail   │   Main Content    │ Right Rail  │
│             │                   │             │
│ 220-240px   │    550-650px      │ 280-320px   │
│             │                   │             │
└─────────────┴───────────────────┴─────────────┘
```

For profile pages, prefer:

```text
┌───────────────────────┬──────────────┐
│                       │              │
│      Profile          │ Suggestions  │
│      Content          │ / Metadata   │
│                       │              │
│      ~70%             │    ~30%      │
└───────────────────────┴──────────────┘
```

Responsive behavior:

```text
≥ 1024px    2–3 columns
768–1023px  main + optional side rail
< 768px     single column
```

Never horizontally shrink desktop cards until they become unreadable. Remove secondary rails instead.

---

# 3. Spacing System

Use an 4px-based spacing system.

```text
4px   micro spacing
8px   related elements
12px  compact component spacing
16px  default component padding
24px  section spacing
32px  large separation
48px  major page separation
```

Prefer:

```css
--space-1: 4px;
--space-2: 8px;
--space-3: 12px;
--space-4: 16px;
--space-6: 24px;
--space-8: 32px;
--space-12: 48px;
```

Do not choose arbitrary values such as `13px`, `19px`, or `27px` unless there is a strong reason.

---

# 4. Colors

Use semantic tokens rather than raw colors inside components.

Example:

```css
--color-bg-page: #f4f2ee;
--color-bg-surface: #ffffff;

--color-text-primary: #191919;
--color-text-secondary: #666666;
--color-text-muted: #767676;

--color-border: #dfdedb;
--color-border-strong: #c7c7c7;

--color-action: #1769aa;
--color-action-hover: #0f548c;
--color-action-subtle: #e8f3fa;

--color-success: #057642;
--color-danger: #b24020;
--color-warning: #915907;
```

The exact accent color is less important than maintaining the hierarchy:

```text
Primary text       almost black
Secondary text     medium gray
Background         warm/light gray
Cards              white
Borders            subtle gray
Actions            restrained blue
```

Do not make every clickable element blue.

Blue should primarily identify:

* links
* important actions
* selected navigation
* primary buttons

---

# 5. Typography

Use a clean system sans-serif stack.

```css
font-family:
  -apple-system,
  BlinkMacSystemFont,
  "Segoe UI",
  Roboto,
  Helvetica,
  Arial,
  sans-serif;
```

Recommended hierarchy:

```text
Profile name        24–28px / 600
Page heading        20–24px / 600
Section heading     18–20px / 600
Card title          16px / 600
Body                14–16px / 400
Metadata            13–14px / 400
Small labels        12px / 500
```

Use weight to establish hierarchy rather than constantly changing font size.

Avoid extremely bold typography.

Typical weights:

```text
400 normal
500 medium
600 semibold
```

---

# 6. Cards

Cards are one of the main primitives.

Use:

```css
background: var(--color-bg-surface);
border: 1px solid var(--color-border);
border-radius: 8px;
```

Prefer borders over heavy shadows.

Typical card:

```text
┌───────────────────────────────────────┐
│ Experience                       ✎    │
│                                       │
│ [Logo] Software Engineer II           │
│        Codecademy                     │
│        2023 — Present                 │
│        Bengaluru, India               │
│                                       │
│        Description...                 │
└───────────────────────────────────────┘
```

Card padding:

```text
16–24px
```

Section separators should generally be:

```css
border-top: 1px solid var(--color-border);
```

rather than separate floating cards for every tiny item.

---

# 7. Buttons

Use three primary button types.

## Primary

Filled action button.

```text
[ Connect ]
```

Use for the most important action on the screen.

## Secondary

Outlined button.

```text
[ Message ]
```

## Tertiary

Text / icon action.

```text
•••
```

General properties:

```css
min-height: 36px;
padding-inline: 16px;
border-radius: 999px;
font-weight: 600;
```

Buttons should usually be pill-shaped.

Never place three equally prominent primary buttons beside each other.

Hierarchy example:

```text
[ Connect ]  [ Message ]  [...]
   primary      secondary   tertiary
```

---

# 8. Profile Header

The profile header should establish identity immediately.

Structure:

```text
┌─────────────────────────────────────────┐
│             Cover Image                 │
│                                         │
│       ┌──────────┐                      │
│       │  Avatar  │                      │
│       └──────────┘                      │
│                                         │
│ Prasanna Kumar                          │
│ Backend Engineer                        │
│ Bengaluru, Karnataka, India             │
│                                         │
│ 500+ connections                        │
│                                         │
│ [Connect] [Message] [...]               │
└─────────────────────────────────────────┘
```

Prioritize:

1. Identity
2. Headline
3. Current role/company
4. Location
5. Network information
6. Primary actions

Do not overwhelm the header with every available piece of profile JSON.

---

# 9. Profile Sections

Recommended profile page ordering:

```text
Profile Header

About

Experience

Education

Skills

Projects

Certifications

Recommendations

Languages / Interests
```

Each section should follow the same structure:

```text
Section title

Primary information
Secondary metadata
Description

Optional action
```

Consistency matters more than making every section visually unique.

---

# 10. Experience Items

Use this hierarchy:

```text
[Company Logo]  Software Engineer II
                Codecademy
                Full-time
                Jan 2023 – Present · 3 yrs
                Bengaluru, India

                Description of responsibilities...

                Go · Kafka · PostgreSQL · Temporal
```

Visual priority:

```text
Job title
↓
Company
↓
Dates/location
↓
Description
↓
Skills
```

Dates and locations should visually recede using secondary text.

---

# 11. Navigation

Desktop navbar should remain compact.

Example:

```text
[Brand] [Search............]

 Home   Network   Jobs   Messaging   Notifications    [Avatar]
```

Target navbar height:

```text
52–56px
```

Navigation icon:

```text
20–24px
```

Labels:

```text
11–13px
```

Selected navigation should have a clear indicator, such as:

```text
Home
────
```

Do not rely exclusively on icon color for selected state.

---

# 12. Search

Search is a major navigation primitive.

Desktop:

```text
┌─────────────────────────┐
│ 🔍 Search                │
└─────────────────────────┘
```

Search should support:

* keyboard focus
* clear placeholder
* autocomplete
* recent searches
* loading state
* empty results
* keyboard navigation where practical

Search results should visually emphasize the matching entity name.

---

# 13. Avatars

Use consistent predefined sizes:

```text
24px   tiny
32px   navigation
40px   comments
48px   list item
56px   feed/profile suggestions
96px+  profile page
```

Use circles for people.

Prefer slightly rounded squares for companies.

Always provide fallback initials or placeholder imagery.

---

# 14. Icons

Use simple outline icons.

Recommended sizes:

```text
16px compact actions
20px controls
24px navigation
```

Avoid mixing:

* filled icons
* outline icons
* emoji
* multiple unrelated icon libraries

Use one icon system across the application.

Good options:

```text
Lucide
Phosphor
Material Symbols
```

---

# 15. Content Density

LinkedIn-like interfaces intentionally display considerable information.

Do not over-modernize by adding massive whitespace.

Correct:

```text
Name
Headline
Company · Dates
Description
```

Incorrect:

```text
          Name


          Headline


          Company


          Dates
```

Spacing should separate semantic groups rather than individual lines.

---

# 16. Interaction States

Every interactive component should define:

```text
default
hover
active
focus
disabled
loading
error
```

Example:

```css
.button:hover {
  background: var(--color-action-hover);
}

.button:focus-visible {
  outline: 2px solid var(--color-action);
  outline-offset: 2px;
}
```

Never remove focus outlines without replacing them with an accessible equivalent.

---

# 17. Loading

Prefer skeleton states for profile content.

Example:

```text
████████████
████████
████████████████████

██████████████████
██████████
```

Avoid a full-page spinner after the page shell has loaded.

Load independently:

```text
profile header
experience
education
recommendations
```

where the API architecture allows it.

---

# 18. Empty States

Empty states should be small and actionable.

Instead of:

```text
NO DATA
```

use:

```text
No education information available.
```

For editable profiles:

```text
No projects added yet.

[ Add project ]
```

Do not create huge illustrations for every empty state.

---

# 19. Accessibility

Accessibility should be part of component design rather than added later.

Requirements:

* keyboard navigation
* visible focus state
* semantic HTML
* meaningful button labels
* alt text for meaningful images
* sufficient text/background contrast
* never communicate state solely through color
* screen-reader-friendly controls
* minimum practical touch targets
* support browser zoom
* responsive text
* honor `prefers-reduced-motion`

Prefer:

```html
<button aria-label="More profile actions">
```

instead of:

```html
<div onclick="...">...</div>
```

---

# 20. Motion

Keep animation subtle.

Recommended duration:

```text
100–150ms hover
150–200ms dropdown
200–250ms modal
```

Use motion primarily to explain:

* state changes
* expanding content
* dropdown appearance
* modal transitions

Avoid:

* bouncing buttons
* parallax
* animated backgrounds
* excessive entrance animations
* spring animations on basic controls

---

# 21. Responsive Mobile UX

Mobile should become a single content stream.

```text
┌─────────────────────┐
│ Header              │
├─────────────────────┤
│ Profile             │
├─────────────────────┤
│ About               │
├─────────────────────┤
│ Experience          │
├─────────────────────┤
│ Education           │
├─────────────────────┤
│ Skills              │
└─────────────────────┘
```

Sidebars should not simply move underneath everything.

Ask whether secondary content is important enough to remain visible on mobile.

Primary actions may become sticky when appropriate.

---

# 22. Design Tokens

Do not scatter visual values throughout components.

Define:

```css
:root {
  --radius-sm: 4px;
  --radius-md: 8px;
  --radius-pill: 999px;

  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-6: 24px;
  --space-8: 32px;

  --font-xs: 12px;
  --font-sm: 14px;
  --font-md: 16px;
  --font-lg: 20px;
  --font-xl: 24px;

  --color-bg-page: #f4f2ee;
  --color-bg-surface: #ffffff;
  --color-text-primary: #191919;
  --color-text-secondary: #666666;
  --color-border: #dfdedb;

  --color-action: #1769aa;
}
```

Components should consume semantic tokens.

Prefer:

```css
color: var(--color-text-secondary);
```

over:

```css
color: #666;
```

---

# 23. Component Architecture

Build reusable primitives:

```text
Avatar
Button
IconButton
Card
Divider
Badge
Tabs
Dropdown
Modal
Tooltip
Skeleton
TextField
SearchInput
```

Then domain components:

```text
ProfileHeader
ExperienceItem
EducationItem
SkillItem
CompanyBadge
ConnectionSummary
ProfileSection
ProfileSidebar
```

Do not implement each profile section independently with duplicated CSS.

---

# 24. UX Priority

When displaying scraped/profile API data, optimize for:

```text
Scan → Understand → Explore
```

A user should be able to understand within a few seconds:

```text
Who is this person?
What do they do?
Where do they work?
What have they done previously?
What technologies/skills do they have?
```

Do not optimize primarily for displaying every field returned by the backend.

The API schema and UI information architecture should remain separate.

---

# 25. Overall Visual Target

The finished application should feel:

```text
Professional
Dense
Structured
Quiet
Trustworthy
Fast
Accessible
Content-first
```

It should not feel like:

```text
Marketing landing page
Admin dashboard
Crypto application
Portfolio template
Glassmorphism demo
Exact LinkedIn clone
```
