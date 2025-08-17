package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type StepDefinition struct {
	Pattern  string
	File     string
	Line     int
	IsImpl   bool
	Function string
}

type FeatureStep struct {
	Text string
	File string
	Line int
}

func main() {
	fmt.Println("=== Buffkit BDD Step Implementation Check ===\n")

	// Find all step definitions
	stepDefs, err := findStepDefinitions()
	if err != nil {
		fmt.Printf("Error finding step definitions: %v\n", err)
		os.Exit(1)
	}

	// Find all feature steps
	featureSteps, err := findFeatureSteps()
	if err != nil {
		fmt.Printf("Error finding feature steps: %v\n", err)
		os.Exit(1)
	}

	// Check for unimplemented steps
	unimplemented := findUnimplementedSteps(stepDefs)

	// Check for undefined steps (in features but not in step definitions)
	undefined := findUndefinedSteps(featureSteps, stepDefs)

	// Report results
	fmt.Printf("📊 Summary:\n")
	fmt.Printf("   Total step definitions: %d\n", len(stepDefs))
	fmt.Printf("   Implemented: %d\n", len(stepDefs)-len(unimplemented))
	fmt.Printf("   Unimplemented: %d\n", len(unimplemented))
	fmt.Printf("   Feature steps: %d\n", len(featureSteps))
	fmt.Printf("   Undefined steps: %d\n\n", len(undefined))

	if len(unimplemented) > 0 {
		fmt.Println("❌ Unimplemented Step Definitions:")
		for _, step := range unimplemented {
			fmt.Printf("   - %s:%d\n     Pattern: %s\n     Function: %s\n\n",
				step.File, step.Line, step.Pattern, step.Function)
		}
	}

	if len(undefined) > 0 {
		fmt.Println("⚠️  Undefined Steps (in features but no step definition):")
		// Group by similar patterns
		grouped := make(map[string][]FeatureStep)
		for _, step := range undefined {
			key := normalizeStep(step.Text)
			grouped[key] = append(grouped[key], step)
		}

		for pattern, steps := range grouped {
			fmt.Printf("   Pattern: %s\n", pattern)
			for _, step := range steps {
				fmt.Printf("      - %s:%d: %s\n", step.File, step.Line, step.Text)
			}
			fmt.Println()
		}
	}

	if len(unimplemented) == 0 && len(undefined) == 0 {
		fmt.Println("✅ All steps are implemented and defined!")
	}

	// Exit with error code if there are issues
	if len(unimplemented) > 0 || len(undefined) > 0 {
		os.Exit(1)
	}
}

func findStepDefinitions() ([]StepDefinition, error) {
	var stepDefs []StepDefinition

	// Pattern to match ctx.Step() calls
	stepPattern := regexp.MustCompile(`ctx\.Step\(\s*` + "`" + `([^` + "`" + `]+)` + "`" + `\s*,\s*([a-zA-Z0-9_.]+)\s*\)`)

	err := filepath.Walk("../features", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		lines := strings.Split(string(content), "\n")

		// Find all step definitions
		for i, line := range lines {
			if matches := stepPattern.FindStringSubmatch(line); len(matches) > 2 {
				stepDef := StepDefinition{
					Pattern:  matches[1],
					Function: matches[2],
					File:     path,
					Line:     i + 1,
					IsImpl:   true,
				}

				// Check if the function contains panic("not yet implemented")
				funcName := matches[2]
				if isUnimplemented(lines, funcName, i) {
					stepDef.IsImpl = false
				}

				stepDefs = append(stepDefs, stepDef)
			}
		}

		return nil
	})

	return stepDefs, err
}

func isUnimplemented(lines []string, funcName string, stepLine int) bool {
	// Look for the function definition
	funcPattern := regexp.MustCompile(`func\s+.*\s+` + regexp.QuoteMeta(funcName) + `\s*\(`)
	panicPattern := regexp.MustCompile(`panic\s*\(\s*"not yet implemented"\s*\)`)

	foundFunc := false
	braceCount := 0

	for i, line := range lines {
		if funcPattern.MatchString(line) {
			foundFunc = true
			braceCount = 0
		}

		if foundFunc {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			if panicPattern.MatchString(line) {
				return true
			}

			// End of function
			if braceCount == 0 && i > stepLine {
				break
			}
		}
	}

	return false
}

func findFeatureSteps() ([]FeatureStep, error) {
	var steps []FeatureStep

	// Patterns for Gherkin keywords
	stepKeywords := regexp.MustCompile(`^\s*(Given|When|Then|And|But)\s+(.+)$`)

	err := filepath.Walk("../features", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".feature") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		inScenario := false
		skipScenario := false

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)

			// Check for scenario start
			if strings.HasPrefix(trimmed, "Scenario:") || strings.HasPrefix(trimmed, "Scenario Outline:") {
				inScenario = true
				skipScenario = false
			}

			// Check for @skip tag
			if strings.HasPrefix(trimmed, "@skip") {
				skipScenario = true
			}

			// Skip if we're in a skipped scenario
			if skipScenario {
				continue
			}

			// Extract step if we're in a scenario
			if inScenario {
				if matches := stepKeywords.FindStringSubmatch(line); len(matches) > 2 {
					steps = append(steps, FeatureStep{
						Text: strings.TrimSpace(matches[2]),
						File: path,
						Line: lineNum,
					})
				}
			}
		}

		return scanner.Err()
	})

	return steps, err
}

func findUndefinedSteps(featureSteps []FeatureStep, stepDefs []StepDefinition) []FeatureStep {
	var undefined []FeatureStep

	for _, step := range featureSteps {
		found := false
		for _, def := range stepDefs {
			if matchesPattern(step.Text, def.Pattern) {
				found = true
				break
			}
		}
		if !found {
			undefined = append(undefined, step)
		}
	}

	return undefined
}

func matchesPattern(stepText, pattern string) bool {
	// Convert step definition pattern to regex
	// Handle common patterns:
	// - "([^"]*)" for quoted strings
	// - (\d+) for numbers
	// - Simple text matches

	regexPattern := "^" + regexp.QuoteMeta(pattern) + "$"
	regexPattern = strings.ReplaceAll(regexPattern, `\"([^\"]*)\"`, `"([^"]*)"`)
	regexPattern = strings.ReplaceAll(regexPattern, `\(\d\+\)`, `(\d+)`)
	regexPattern = strings.ReplaceAll(regexPattern, `\(\\d\+\)`, `(\d+)`)

	// Handle special regex characters that might be in the pattern
	regexPattern = strings.ReplaceAll(regexPattern, `\^`, "^")
	regexPattern = strings.ReplaceAll(regexPattern, `\$`, "$")

	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		// If we can't compile, try simple contains match
		return strings.Contains(stepText, strings.Trim(pattern, "^$"))
	}

	return regex.MatchString(stepText)
}

func findUnimplementedSteps(stepDefs []StepDefinition) []StepDefinition {
	var unimplemented []StepDefinition

	for _, def := range stepDefs {
		if !def.IsImpl {
			unimplemented = append(unimplemented, def)
		}
	}

	return unimplemented
}

func normalizeStep(step string) string {
	// Normalize step text to group similar steps
	normalized := step

	// Replace quoted strings with placeholder
	normalized = regexp.MustCompile(`"[^"]*"`).ReplaceAllString(normalized, `"..."`)

	// Replace numbers with placeholder
	normalized = regexp.MustCompile(`\b\d+\b`).ReplaceAllString(normalized, "N")

	// Replace URLs
	normalized = regexp.MustCompile(`https?://[^\s]+`).ReplaceAllString(normalized, "URL")

	// Replace paths
	normalized = regexp.MustCompile(`/[^\s]+`).ReplaceAllString(normalized, "/PATH")

	return normalized
}
