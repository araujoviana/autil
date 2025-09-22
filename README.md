# autil

autil is a tool written in Go for loading, visualizing, and evaluating automata defined in JSON.
The project aims to serve as a foundation for experimenting with finite automata, formal grammars, and future extensions in language analysis.

## 📦 Project Structure

* `main.go` – entry point; loads the grammar, does the stuff
* `states.json` – configuration file that describes the automaton/grammar

---

## 📄 JSON Format

Example of a simple automaton:

```json
{
  "states": ["q0", "q1", "q2"],
  "alphabet": ["a", "b"],
  "transition": {
    "q0,a": "q1",
    "q0,b": "q2",
    "q1,a": "q2",
    "q1,b": "q0",
    "q2,a": "q0",
    "q2,b": "q1"
  },
  "start": "q0",
  "accept": ["q2"]
}
```

* **states**: list of states.
* **alphabet**: valid symbols.
* **transition**: map of transitions in the form `"state,symbol": "next_state"`.
* **start**: initial state.
* **accept**: list of accepting states.

---

## ▶️ Running

1. Just compile or run directly:

```bash
go run .
```

---

## 🔧 Dependencies

* [promptui](https://github.com/manifoldco/promptui) – for user-friendly terminal interaction.
