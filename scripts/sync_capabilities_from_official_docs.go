package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/citizenofai/agent-sdk-go/pkg/model"
)

type capSet struct {
	vision    bool
	documents bool
}

type providerSource struct {
	providerID string
	patterns   []*regexp.Regexp
	allow      func(string) bool
	infer      func(string) capSet
}

var (
	spaceRegexp = regexp.MustCompile(`\s+`)

	providerSectionRegexps = map[string]*regexp.Regexp{
		"mistral":   regexp.MustCompile(`(?ms)(\t"mistral": \{\n)(.*?)(\t\},\n)`),
		"openai":    regexp.MustCompile(`(?ms)(\t"openai": \{\n)(.*?)(\t\},\n)`),
		"anthropic": regexp.MustCompile(`(?ms)(\t"anthropic": \{\n)(.*?)(\t\},\n)`),
		"gemini":    regexp.MustCompile(`(?ms)(\t"gemini": \{\n)(.*?)(\t\},\n)`),
	}

	capabilityEntryRegexp = regexp.MustCompile(`\{prefix: "([^"]+)", caps: map\[Capability\]bool\{([^}]*)\}\},`)

	mistralAllowRegexp   = regexp.MustCompile(`^(?:mistral-(?:large|medium|small)-[0-9]{4}|mistral-ocr-[0-9]{4}|magistral-(?:small|medium)-[0-9]{4}|ministral-(?:3b|8b|14b)-[0-9]{4}|pixtral(?:-[0-9]{4})?|ocr-3(?:-[0-9]{2}-[0-9]{2})?)$`)
	openAIAllowRegexp    = regexp.MustCompile(`^(?:gpt-(?:4o|4\.1|4\.5|5(?:\.[0-9]+)?)(?:-[a-z0-9.]+)?|gpt-4-turbo(?:-[a-z0-9.]+)?|gpt-4-vision(?:-[a-z0-9.]+)?|o[134](?:-[a-z0-9.]+)?)$`)
	anthropicAllowRegexp = regexp.MustCompile(`^claude-(?:opus|sonnet|haiku|3)(?:-[a-z0-9.]+)+$`)
	geminiAllowRegexp    = regexp.MustCompile(`^(?:gemini-(?:1\.5|2(?:\.5)?|3(?:\.[0-9]+)?)(?:-[a-z0-9.]+)*|gemini-(?:flash|pro)-latest)$`)
)

var knownNoDigitModelAliases = map[string]struct{}{
	"claude-haiku-latest":  {},
	"claude-opus-latest":   {},
	"claude-sonnet-latest": {},
	"gemini-flash-latest":  {},
	"gemini-pro-latest":    {},
}

var providerSources = []providerSource{
	{
		providerID: "mistral",
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(mistral-[a-z0-9.-]+)`),
			regexp.MustCompile(`(?i)(magistral-[a-z0-9.-]+)`),
			regexp.MustCompile(`(?i)(ministral-[a-z0-9.-]+)`),
			regexp.MustCompile(`(?i)(pixtral-[a-z0-9.-]+)`),
			regexp.MustCompile(`(?i)(mistral-ocr-[a-z0-9.-]+)`),
			regexp.MustCompile(`(?i)(ocr-3(?:-[a-z0-9.-]+)?)`),
		},
		allow: mistralAllowRegexp.MatchString,
		infer: inferCapabilitiesMistral,
	},
	{
		providerID: "openai",
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(gpt-[a-z0-9.-]+)`),
			regexp.MustCompile(`(?i)(o[134](?:-[a-z0-9.-]+)?)`),
		},
		allow: openAIAllowRegexp.MatchString,
		infer: inferCapabilitiesOpenAI,
	},
	{
		providerID: "anthropic",
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(claude-[a-z0-9.-]+)`),
		},
		allow: anthropicAllowRegexp.MatchString,
		infer: inferCapabilitiesAnthropic,
	},
	{
		providerID: "gemini",
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(gemini-[a-z0-9.-]+)`),
		},
		allow: geminiAllowRegexp.MatchString,
		infer: inferCapabilitiesGemini,
	},
}

func main() {
	capabilitiesPath := flag.String("capabilities", "pkg/model/capabilities.go", "Path to capabilities.go")
	timeoutSec := flag.Int("timeout", 20, "HTTP timeout in seconds")
	writeChanges := flag.Bool("write", false, "Write changes to capabilities.go (default: report only)")
	verbose := flag.Bool("verbose", false, "Print missing model IDs per provider")
	flag.Parse()

	capsFileBytes, err := os.ReadFile(*capabilitiesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read capabilities file: %v\n", err)
		os.Exit(1)
	}
	capsFile := string(capsFileBytes)

	httpClient := &http.Client{Timeout: timeDurationSeconds(*timeoutSec)}

	missing := map[string]map[string]capSet{}
	for _, src := range providerSources {
		sourceURL, ok := model.OfficialProviderModelDocsURL(src.providerID)
		if !ok || strings.TrimSpace(sourceURL) == "" {
			fmt.Fprintf(os.Stderr, "missing official docs URL for provider: %s\n", src.providerID)
			os.Exit(1)
		}

		content, fetchErr := fetchPage(httpClient, sourceURL)
		if fetchErr != nil {
			fmt.Fprintf(os.Stderr, "fetch %s models page failed: %v\n", src.providerID, fetchErr)
			os.Exit(1)
		}

		modelIDs := extractModelIDs(content, src.patterns, src.allow)
		for _, modelID := range modelIDs {
			if model.ProviderSupports(src.providerID, modelID, model.CapabilityVision) ||
				model.ProviderSupports(src.providerID, modelID, model.CapabilityDocuments) {
				continue
			}

			inferred := src.infer(modelID)
			if !inferred.vision && !inferred.documents {
				continue
			}
			if _, ok := missing[src.providerID]; !ok {
				missing[src.providerID] = map[string]capSet{}
			}
			missing[src.providerID][modelID] = inferred
		}
	}

	if len(missing) == 0 {
		fmt.Println("capabilities.go is already aligned with official provider docs")
		return
	}

	for providerID, models := range missing {
		fmt.Printf("provider=%s missing_multimodal_models=%d\n", providerID, len(models))
		if *verbose {
			modelIDs := make([]string, 0, len(models))
			for id := range models {
				modelIDs = append(modelIDs, id)
			}
			sort.Strings(modelIDs)
			for _, id := range modelIDs {
				fmt.Printf("  - %s\n", id)
			}
		}
	}

	if !*writeChanges {
		fmt.Println("report-only mode: re-run with -write to update capabilities.go")
		os.Exit(1)
	}

	updated, updateErr := applyMissingAndSortByProvider(capsFile, missing)
	if updateErr != nil {
		fmt.Fprintf(os.Stderr, "update capabilities file content: %v\n", updateErr)
		os.Exit(1)
	}

	if err := os.WriteFile(*capabilitiesPath, []byte(updated), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write capabilities file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("updated capabilities.go with models from official provider docs")
}

func fetchPage(client *http.Client, url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "agent-sdk-go-capability-sync/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return extractDocumentCorpus(body)
}

func extractDocumentCorpus(raw []byte) (string, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(raw))
	if err != nil {
		return "", err
	}

	var b strings.Builder
	body := doc.Find("body")
	if body.Length() == 0 {
		body = doc.Selection
	}

	b.WriteString(body.Text())
	b.WriteString(" ")

	body.Find("code, pre, kbd, samp").Each(func(_ int, s *goquery.Selection) {
		b.WriteString(s.Text())
		b.WriteString(" ")
	})

	body.Find("a").Each(func(_ int, s *goquery.Selection) {
		if href, ok := s.Attr("href"); ok {
			b.WriteString(href)
			b.WriteString(" ")
		}
	})

	lower := strings.ToLower(b.String())
	return spaceRegexp.ReplaceAllString(lower, " "), nil
}

func extractModelIDs(content string, patterns []*regexp.Regexp, allowFn func(string) bool) []string {
	set := map[string]struct{}{}
	for _, pattern := range patterns {
		matches := pattern.FindAllString(content, -1)
		for _, m := range matches {
			candidate := normalizeModelID(m)
			if candidate == "" {
				continue
			}
			if !allowFn(candidate) {
				continue
			}
			set[candidate] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func normalizeModelID(modelID string) string {
	trimmed := strings.TrimSpace(strings.ToLower(modelID))
	trimmed = strings.Trim(trimmed, "\"'`),.;:!?[]{}<>")
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, " ") {
		return ""
	}
	if strings.HasPrefix(trimmed, "http") {
		return ""
	}
	if len(trimmed) > 96 {
		return ""
	}
	if !containsDigit(trimmed) && !isKnownNoDigitModelAlias(trimmed) {
		return ""
	}
	if !isTokenCharsetValid(trimmed) {
		return ""
	}
	if hasAnyNoiseToken(trimmed) {
		return ""
	}
	return trimmed
}

func isKnownNoDigitModelAlias(s string) bool {
	_, ok := knownNoDigitModelAliases[s]
	return ok
}

func containsDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func isTokenCharsetValid(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func hasAnyNoiseToken(s string) bool {
	noise := []string{
		"logo", "card", "color", "font", "table", "grid", "cta", "supported",
		"resource", "theme", "bullet", "description", "overview", "button",
		"not-supported", "experimental", "icon", "javascript", "react", "tailwind",
	}
	for _, token := range noise {
		if strings.Contains(s, token) {
			return true
		}
	}
	return false
}

func timeDurationSeconds(sec int) time.Duration {
	if sec <= 0 {
		sec = 20
	}
	return time.Duration(sec) * time.Second
}

func inferCapabilitiesMistral(modelID string) capSet {
	m := strings.ToLower(modelID)
	if strings.HasPrefix(m, "mistral-ocr") || strings.HasPrefix(m, "ocr-3") {
		return capSet{documents: true}
	}
	if strings.HasPrefix(m, "mistral-large") ||
		strings.HasPrefix(m, "mistral-medium") ||
		strings.HasPrefix(m, "mistral-small") ||
		strings.HasPrefix(m, "magistral-") ||
		strings.HasPrefix(m, "ministral-") ||
		strings.HasPrefix(m, "pixtral-") ||
		m == "pixtral" {
		return capSet{vision: true}
	}
	return capSet{}
}

func inferCapabilitiesOpenAI(modelID string) capSet {
	m := strings.ToLower(modelID)
	if strings.HasPrefix(m, "gpt-4") || strings.HasPrefix(m, "gpt-5") {
		return capSet{vision: true, documents: true}
	}
	if strings.HasPrefix(m, "o1") || strings.HasPrefix(m, "o3") || strings.HasPrefix(m, "o4") {
		return capSet{vision: true}
	}
	return capSet{}
}

func inferCapabilitiesAnthropic(modelID string) capSet {
	if strings.HasPrefix(strings.ToLower(modelID), "claude-") {
		return capSet{vision: true, documents: true}
	}
	return capSet{}
}

func inferCapabilitiesGemini(modelID string) capSet {
	m := strings.ToLower(modelID)
	if strings.HasPrefix(m, "gemini-1.5") ||
		strings.HasPrefix(m, "gemini-2") ||
		strings.HasPrefix(m, "gemini-3") ||
		m == "gemini-flash-latest" ||
		m == "gemini-pro-latest" {
		return capSet{vision: true, documents: true}
	}
	return capSet{}
}

func applyMissingAndSortByProvider(content string, missing map[string]map[string]capSet) (string, error) {
	providerOrder := model.KnownCapabilityProviders()
	updated := content
	for _, providerID := range providerOrder {
		re, ok := providerSectionRegexps[providerID]
		if !ok {
			return "", fmt.Errorf("missing section regexp for provider: %s", providerID)
		}
		match := re.FindStringSubmatch(updated)
		if len(match) != 4 {
			return "", fmt.Errorf("provider section not found in capabilities.go: %s", providerID)
		}

		existing := parseProviderEntries(match[2])
		for prefix, caps := range missing[providerID] {
			existing[prefix] = caps
		}

		rewrittenBody := renderSortedProviderEntries(existing)
		replacement := match[1] + rewrittenBody + match[3]
		updated = strings.Replace(updated, match[0], replacement, 1)
	}
	return updated, nil
}

func parseProviderEntries(sectionBody string) map[string]capSet {
	entries := map[string]capSet{}
	matches := capabilityEntryRegexp.FindAllStringSubmatch(sectionBody, -1)
	for _, m := range matches {
		if len(m) != 3 {
			continue
		}
		capsText := m[2]
		entries[m[1]] = capSet{
			vision:    strings.Contains(capsText, "CapabilityVision: true"),
			documents: strings.Contains(capsText, "CapabilityDocuments: true"),
		}
	}
	return entries
}

func renderSortedProviderEntries(entries map[string]capSet) string {
	prefixes := make([]string, 0, len(entries))
	for prefix := range entries {
		prefixes = append(prefixes, prefix)
	}
	sort.Strings(prefixes)

	var b strings.Builder
	for _, prefix := range prefixes {
		caps := entries[prefix]
		b.WriteString("\t\t{prefix: ")
		fmt.Fprintf(&b, "%q", prefix)
		b.WriteString(", caps: map[Capability]bool{")
		if caps.vision {
			b.WriteString("CapabilityVision: true")
			if caps.documents {
				b.WriteString(", ")
			}
		}
		if caps.documents {
			b.WriteString("CapabilityDocuments: true")
		}
		b.WriteString("}},\n")
	}
	return b.String()
}
