package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/manifoldco/promptui"
	"github.com/olekukonko/tablewriter"
)

type Grammar struct {
	States     []string            `json:"states"`
	Alphabet   []string            `json:"alphabet"`
	Transition map[string][]string `json:"transition"`
	Start      string              `json:"start"`
	Accept     []string            `json:"accept"`
}

type Branch struct {
	Path  []string
	State string
}

func containsString(slice []string, str string) bool {
	return slices.Contains(slice, str)
}

func main() {
	fmt.Println("Reading JSON States!")
	userJSON, err := os.ReadFile("states.json")
	if err != nil {
		fmt.Println("Error reading file:", err)
		panic(err)
	}
	fmt.Println("File read successfully!")

	userGrammar := Grammar{}
	err = json.Unmarshal(userJSON, &userGrammar)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		panic(err)
	}

	// Print parsed grammar
	fmt.Println("JSON parsed successfully!")
	pretty, _ := json.MarshalIndent(userGrammar, "", "  ")
	fmt.Println("Parsed Grammar:")
	fmt.Println(string(pretty))

	// Display transition table
	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"From", "Symbol", "To"})

	for k, v := range userGrammar.Transition {
		parts := strings.Split(k, ",")
		table.Append([]string{parts[0], parts[1], strings.Join(v, ", ")})
	}
	table.Render()

	for {
		prompt := promptui.Prompt{
			Label: "Enter a string to evaluate (or type 'exit' to quit)",
		}
		input, err := prompt.Run()
		if err != nil {
			fmt.Println("Prompt failed:", err)
			return
		}
		if input == "exit" {
			break
		}

		branches := []Branch{{Path: []string{userGrammar.Start}, State: userGrammar.Start}}
		accepted := false

		fmt.Println(promptui.Styler(promptui.FGCyan)("ℹ Initial state: " + userGrammar.Start))

		for _, char := range input {
			if !containsString(userGrammar.Alphabet, string(char)) {
				warn := promptui.Styler(promptui.FGYellow)(
					"⚠ Warning: Character '" + string(char) + "' not in alphabet. Skipping.")
				fmt.Println(warn)
				accepted = false
				break
			}

			fmt.Println(promptui.Styler(promptui.FGCyan)(
				fmt.Sprintf("Reading symbol '%c' from %d branches", char, len(branches)),
			))

			var wg sync.WaitGroup
			nextBranchesChan := make(chan Branch, 10)

			// process each branch in parallel
			for _, br := range branches {
				wg.Add(1)
				go func(br Branch) {
					defer wg.Done()
					transitionKey := fmt.Sprintf("%s,%c", br.State, char)
					if nextList, exists := userGrammar.Transition[transitionKey]; exists {
						for _, ns := range nextList {
							newPath := append(append([]string{}, br.Path...), ns)
							nextBranchesChan <- Branch{Path: newPath, State: ns}
						}
					}
				}(br)
			}

			wg.Wait()
			close(nextBranchesChan)

			var nextBranches []Branch
			for nb := range nextBranchesChan {
				nextBranches = append(nextBranches, nb)
			}

			if len(nextBranches) == 0 {
				errMsg := promptui.Styler(promptui.FGRed)("❌ No valid transitions")
				fmt.Println(errMsg)
				accepted = false
				break
			}

			branches = nextBranches
		}

		// Print all final branches
		if len(branches) > 0 {
			for i, br := range branches {
				pathStr := strings.Join(br.Path, " -> ")
				fmt.Printf("Branch %d: %s\n", i+1, pathStr)
				if containsString(userGrammar.Accept, br.State) {
					accepted = true
				}
			}
		}

		if accepted {
			success := promptui.Styler(promptui.FGGreen)("✅ String accepted!")
			fmt.Println(success)
		} else {
			fail := promptui.Styler(promptui.FGRed)("❌ String rejected!")
			fmt.Println(fail)
		}

		// Print divider
		divider := promptui.Styler(promptui.FGMagenta)(strings.Repeat("-", 30))
		fmt.Println(divider)
	}
}
