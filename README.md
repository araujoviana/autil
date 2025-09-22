# autil

autil is a Go tool for loading, visualizing, and evaluating automata defined in JSON.
It’s a small base to experiment with DFAs/NFAs and simple language analysis.

## 📦 Project Structure

* `main.go` – entry point; loads the grammar and runs evaluations
* `states.json` – JSON that describes the automaton/grammar

---

## 📄 JSON Format

Example (NFA style — transitions map to **lists** of states):

```json
{
  "states": ["q0", "q1", "q2"],
  "alphabet": ["a", "b"],
  "transition": {
    "q0,a": ["q0", "q1"],
    "q0,b": ["q2"],
    "q1,a": ["q2"],
    "q1,b": ["q0"],
    "q2,a": ["q1"],
    "q2,b": []
  },
  "start": "q0",
  "accept": ["q2"]
}
```

* **states**: all states.
* **alphabet**: valid symbols (single-char strings).
* **transition**: `"state,symbol" -> [next_state, ...]`.
* **start**: initial state.
* **accept**: accepting states.

---

## ▶️ Running

Compile or run:

```bash
go run .
```

Useful flags:

```bash
# choose file (default: states.json)
go run . -f states.json

# verbose: show state sets per step
go run . -f states.json -v

# reconstruct and print branches (limited)
go run . -f states.json -branches -maxbranches 32

# export branches to Graphviz DOT (use with -branches)
go run . -f states.json -branches -dot run.dot
# then: dot -Tpng run.dot -o run.png
```

---

## 🔧 Dependencies

* [olekukonko/tablewriter](https://github.com/olekukonko/tablewriter) – pretty tables for verbose output.
