package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"math/bits"
	"os"
	"slices"
	"strings"
)

type Grammar struct {
	States     []string            `json:"states"`
	Alphabet   []string            `json:"alphabet"`
	Transition map[string][]string `json:"transition"`
	Start      string              `json:"start"`
	Accept     []string            `json:"accept"`
}

type NFA struct {
	States      []string
	StateIndex  map[string]int
	SymbolIndex map[string]int
	Next        [][]uint64
	AcceptMask  uint64
	StartIndex  int
	Alphabet    []string
}

func buildNFA(g Grammar) (*NFA, error) {
	if len(g.States) == 0 {
		return nil, fmt.Errorf("No states!")
	}
	if len(g.States) > 64 {
		return nil, fmt.Errorf("Too many states! Max is 64")
	}
	if len(g.Alphabet) == 0 {
		return nil, fmt.Errorf("No alphabet!")
	}
	if len(g.Start) == 0 {
		return nil, fmt.Errorf("No start state!")
	}
	if !containsString(g.States, g.Start) {
		return nil, fmt.Errorf("Start state '%s' not in states", g.Start)
	}
	if len(g.Accept) == 0 {
		return nil, fmt.Errorf("No accept states!")
	}
	for _, a := range g.Accept {
		if !containsString(g.States, a) {
			return nil, fmt.Errorf("Accept state '%s' not in states", a)
		}
	}

	stateIndex := make(map[string]int, len(g.States))
	for i, s := range g.States {
		stateIndex[s] = i
	}
	if _, ok := stateIndex[g.Start]; !ok {
		return nil, fmt.Errorf("Start state %q not in states", g.Start)
	}

	symbolIndex := make(map[string]int, len(g.Alphabet))
	for i, a := range g.Alphabet {
		if len(a) == 0 {
			return nil, fmt.Errorf("empty symbol not allowed")
		}
		symbolIndex[a] = i
	}

	next := make([][]uint64, len(g.States))
	for i := range next {
		next[i] = make([]uint64, len(g.Alphabet))
	}

	for k, v := range g.Transition {
		parts := strings.Split(k, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("Invalid transition key: %s (expected 'state,symbol')", k)
		}
		fromState := parts[0]
		symbol := parts[1]

		fromIndex, ok := stateIndex[fromState]
		if !ok {
			return nil, fmt.Errorf("Transition from unknown state: %s", fromState)
		}
		symbolIndexValue, ok := symbolIndex[symbol]
		if !ok {
			return nil, fmt.Errorf("Transition with unknown symbol: %s", symbol)
		}

		for _, toState := range v {
			toIndex, ok := stateIndex[toState]
			if !ok {
				return nil, fmt.Errorf("Transition to unknown state: %s", toState)
			}
			next[fromIndex][symbolIndexValue] |= 1 << toIndex
		}
	}

	var acceptMask uint64
	for _, a := range g.Accept {
		i := stateIndex[a]
		acceptMask |= 1 << i
	}

	startIndex := stateIndex[g.Start]

	nfa := &NFA{
		States:      g.States,
		StateIndex:  stateIndex,
		SymbolIndex: symbolIndex,
		Next:        next,
		AcceptMask:  acceptMask,
		StartIndex:  startIndex,
		Alphabet:    g.Alphabet,
	}
	return nfa, nil
}

func (n *NFA) RunFast(input string, verbose bool) (bool, uint64, []uint64, []rune) {
	curr := uint64(1) << n.StartIndex
	var sets []uint64
	var syms []rune
	if verbose {
		sets = append(sets, curr)
	}
	cache := make(map[[2]uint64]uint64, 128)

	for _, r := range input {
		sym := string(r)
		ai, ok := n.SymbolIndex[sym]
		if !ok {
			return false, curr, sets, syms
		}
		key := [2]uint64{curr, uint64(ai)}
		if nextMask, ok := cache[key]; ok {
			curr = nextMask
		} else {
			var nextMask uint64
			c := curr
			for c != 0 {
				lsb := c & -c
				i := bits.TrailingZeros64(lsb)
				nextMask |= n.Next[i][ai]
				c &= c - 1
			}
			cache[key] = nextMask
			curr = nextMask
		}

		if curr == 0 {
			if verbose {
				sets = append(sets, curr)
				syms = append(syms, r)
			}
			return false, 0, sets, syms
		}
		if verbose {
			sets = append(sets, curr)
			syms = append(syms, r)
		}
	}
	accepted := (curr & n.AcceptMask) != 0
	return accepted, curr, sets, syms
}

func (n *NFA) RunWithPreds(input string) (bool, uint64, []map[int]uint64) {
	curr := uint64(1) << n.StartIndex

	// preds[step][state] = bitset de predecessores em 'step' que levam a 'state'
	// step 0 = antes de consumir input (só start)
	preds := make([]map[int]uint64, 0, len(input)+1)
	startPred := map[int]uint64{n.StartIndex: 0}
	preds = append(preds, startPred)

	for _, r := range input {
		sym := string(r)
		ai, ok := n.SymbolIndex[sym]
		if !ok {
			return false, curr, preds
		}

		var next uint64
		nextPredsMap := make(map[int]uint64)

		c := curr
		for c != 0 {
			lsb := c & -c
			i := bits.TrailingZeros64(lsb)
			nxt := n.Next[i][ai]
			next |= nxt

			// marca 'i' como predecessor de todos 'j' alcançados
			d := nxt
			for d != 0 {
				lsb2 := d & -d
				j := bits.TrailingZeros64(lsb2)
				nextPredsMap[j] |= 1 << i
				d &= d - 1
			}
			c &= c - 1
		}

		curr = next
		if curr == 0 {
			return false, 0, preds
		}
		preds = append(preds, nextPredsMap)
	}
	accepted := (curr & n.AcceptMask) != 0
	return accepted, curr, preds
}

func (n *NFA) ReconstructBranches(input string, finalMask uint64, preds []map[int]uint64, limit int) [][]string {
	if len(input) != len(preds)-1 {
		return nil
	}

	var branches [][]string
	var dfs func(pos int, state int, path []string)

	dfs = func(pos int, state int, path []string) {
		if len(branches) >= limit {
			return
		}
		path = append([]string{n.States[state]}, path...)
		if pos == 0 {
			branches = append(branches, path)
			return
		}
		predMap := preds[pos]
		predsMask, ok := predMap[state]
		if !ok {
			return
		}
		for predsMask != 0 {
			lsb := predsMask & -predsMask
			predState := bits.TrailingZeros64(lsb)
			dfs(pos-1, predState, path)
			predsMask &= predsMask - 1
		}
	}

	for fm := finalMask; fm != 0; fm &= fm - 1 {
		lsb := fm & -fm
		finalState := bits.TrailingZeros64(lsb)
		dfs(len(input), finalState, nil)
		if len(branches) >= limit {
			break
		}
	}
	return branches
}

func containsString(slice []string, str string) bool {
	return slices.Contains(slice, str)
}

func main() {
	file := flag.String("f", "states.json", "Grammar JSON file")
	verbose := flag.Bool("v", false, "Verbose: print state sets per step")
	printBranches := flag.Bool("branches", false, "Print branches for accepted strings")
	maxBranches := flag.Int("maxbranches", 32, "Max branches to print when -branches is set")
	dotOut := flag.String("dot", "", "Export reconstructed branches to Graphviz DOT (use with -branches)")
	flag.Parse()

	data, err := os.ReadFile(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	var grammar Grammar
	if err := json.Unmarshal(data, &grammar); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	nfa, err := buildNFA(grammar)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building NFA: %v\n", err)
		os.Exit(1)
	}

	in := bufio.NewScanner(os.Stdin)
	fmt.Println("NFA ready. Enter strings to test (Ctrl+D to end):")
	for {
		fmt.Print("> ")
		if !in.Scan() {
			break
		}
		line := strings.TrimSpace(in.Text())
		if line == "" {
			continue
		}

		accepted, finalMask, sets, syms := nfa.RunFast(line, *verbose)
		if accepted {
			fmt.Printf("Accepted: %q\n", line)
			if *printBranches {
				_, _, preds := nfa.RunWithPreds(line)
				branches := nfa.ReconstructBranches(line, finalMask, preds, *maxBranches)

				if len(branches) == 0 {
					fmt.Println("Branches: none")
				} else {
					fmt.Printf("Branches (max %d):\n", *maxBranches)
					for i, b := range branches {
						fmt.Printf("  %2d: %s\n", i+1, strings.Join(b, " -> "))
					}
				}

				// DOT export (if requested)
				if *dotOut != "" && len(branches) > 0 {
					var b strings.Builder
					b.WriteString("digraph NFArun {\nrankdir=LR;\nnode [shape=circle];\n")

					// doublecircle for accepting states
					accept := make(map[string]bool)
					for _, s := range grammar.Accept {
						accept[s] = true
					}

					// edges: each step i uses input rune i-1 as label
					runes := []rune(line)
					for _, path := range branches {
						for i := 0; i+1 < len(path); i++ {
							lbl := ""
							if i < len(runes) {
								lbl = string(runes[i])
							}
							fmt.Fprintf(&b, "\"%s\" -> \"%s\" [label=\"%s\"];\n", path[i], path[i+1], lbl)
						}
					}
					for s := range accept {
						fmt.Fprintf(&b, "\"%s\" [shape=doublecircle];\n", s)
					}
					b.WriteString("}\n")

					if err := os.WriteFile(*dotOut, []byte(b.String()), 0644); err != nil {
						fmt.Fprintf(os.Stderr, "DOT write error: %v\n", err)
					} else {
						fmt.Printf("DOT exported to %s (use: dot -Tpng %s -o run.png)\n", *dotOut, *dotOut)
					}
				}
			}
		} else {
			fmt.Printf("Rejected: %q\n", line)
		}

		if *verbose {
			fmt.Println("State sets per step:")
			table := tablewriter.NewWriter(os.Stdout)
			table.Header([]string{"Step", "Input", "States"})
			for i, set := range sets {
				var sym string
				if i == 0 {
					sym = "ε"
				} else if i-1 < len(syms) {
					sym = string(syms[i-1])
				}
				var states []string
				for j := 0; j < len(nfa.States); j++ {
					if (set & (1 << j)) != 0 {
						states = append(states, nfa.States[j])
					}
				}
				table.Append([]string{
					fmt.Sprintf("%d", i),
					sym,
					strings.Join(states, ", "),
				})
			}
			table.Render()
		}
	}

	if err := in.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}
