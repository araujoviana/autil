package main

import (
	"encoding/json"
	"fmt"
	"github.com/manifoldco/promptui"
	"github.com/olekukonko/tablewriter"
	"os"
	"slices"
	"strings"
)

type Grammar struct {
	States     []string          `json:"states"`
	Alphabet   []string          `json:"alphabet"`
	Transition map[string]string `json:"transition"`
	Start      string            `json:"start"`
	Accept     []string          `json:"accept"`
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
		table.Append([]string{parts[0], parts[1], v})
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

		currentState := userGrammar.Start
		accepted := true

		for _, char := range input {
			if !containsString(userGrammar.Alphabet, string(char)) {
				warn := promptui.Styler(promptui.FGYellow)("⚠ Character not in alphabet: " + string(char))
				fmt.Println(warn)
				accepted = false
				break
			}

			transitionKey := fmt.Sprintf("%s,%c", currentState, char)
			nextState, exists := userGrammar.Transition[transitionKey]
			if !exists {
				errMsg := promptui.Styler(promptui.FGRed)("❌ No transition for " + transitionKey)
				fmt.Println(errMsg)
				accepted = false
				break
			}
			currentState = nextState
		}

		info := promptui.Styler(promptui.FGCyan)("ℹ Final state: " + currentState)
		fmt.Println(info)

		if accepted && containsString(userGrammar.Accept, currentState) {
			success := promptui.Styler(promptui.FGGreen)("✅ String accepted!")
			fmt.Println(success)
		} else {
			fail := promptui.Styler(promptui.FGRed)("❌ String rejected!")
			fmt.Println(fail)
		}

		divider := promptui.Styler(promptui.FGMagenta)(strings.Repeat("-", 30))
		fmt.Println(divider)

	}
}
