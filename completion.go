package readline

import (
	"fmt"

	"github.com/reeflective/readline/internal/color"
	"github.com/reeflective/readline/internal/completion"
	"github.com/reeflective/readline/internal/keymap"
)

func (rl *Shell) completionCommands() commands {
	return map[string]func(){
		"complete":               rl.completeWord,
		"possible-completions":   rl.possibleCompletions,
		"insert-completions":     rl.insertCompletions,
		"menu-complete":          rl.menuComplete,
		"menu-complete-backward": rl.menuCompleteBackward,
		"delete-char-or-list":    rl.deleteCharOrList,

		"menu-complete-next-tag":   rl.menuCompleteNextTag,
		"menu-complete-prev-tag":   rl.menuCompletePrevTag,
		"accept-and-menu-complete": rl.acceptAndMenuComplete,
		"vi-registers-complete":    rl.viRegistersComplete,
		"menu-incremental-search":  rl.menuIncrementalSearch,
	}
}

//
// Commands ---------------------------------------------------------------------------
//

// Attempt completion on the current word.
// Currently identitical to menu-complete.
func (rl *Shell) completeWord() {
	rl.undo.SkipSave()

	// This completion function should attempt to insert the first
	// valid completion found, without printing the actual list.
	if !rl.completer.IsActive() {
		rl.startMenuComplete(rl.commandCompletion)
	}
	rl.completer.Select(1, 0)
}

// List possible completions for the current word.
func (rl *Shell) possibleCompletions() {
	rl.undo.SkipSave()

	rl.completer.Cancel(false, false)
	rl.keymaps.SetLocal(keymap.MenuSelect)
	rl.completer.GenerateWith(rl.commandCompletion)
}

func (rl *Shell) insertCompletions() {}

// Like complete-word, except that menu completion is used.
func (rl *Shell) menuComplete() {
	rl.undo.SkipSave()

	// No completions are being printed yet, so simply generate the completions
	// as if we just request them without immediately selecting a candidate.
	if !rl.completer.IsActive() {
		rl.startMenuComplete(rl.commandCompletion)
	} else {
		rl.completer.Select(1, 0)
	}
}

// Deletes the character under the cursor if not at the
// beginning or end of the line (like delete-char).
// If at the end of the line, behaves identically to
// possible-completions.
func (rl *Shell) deleteCharOrList() {
	switch {
	case rl.cursor.Pos() < rl.line.Len():
		rl.line.CutRune(rl.cursor.Pos())
	default:
		rl.possibleCompletions()
	}
}

// Identical to menu-complete, but moves backward through the
// list of possible completions, as if menu-complete had been
// given a negative argument.
func (rl *Shell) menuCompleteBackward() {
	rl.undo.SkipSave()

	// We don't do anything when not already completing.
	if !rl.completer.IsActive() {
		return
	}

	rl.completer.Select(-1, 0)
}

// In a menu completion, if there are several tags
// of completions, go to the first result of the next tag.
func (rl *Shell) menuCompleteNextTag() {
	rl.undo.SkipSave()

	if !rl.completer.IsActive() {
		return
	}

	rl.completer.SelectTag(true)
}

// In a menu completion, if there are several tags of
// completions, go to the first result of the previous tag.
func (rl *Shell) menuCompletePrevTag() {
	rl.undo.SkipSave()

	if !rl.completer.IsActive() {
		return
	}

	rl.completer.SelectTag(false)
}

// In a menu completion, insert the current completion
// into the buffer, and advance to the next possible completion.
func (rl *Shell) acceptAndMenuComplete() {
	rl.undo.SkipSave()

	// We don't do anything when not already completing.
	if !rl.completer.IsActive() {
		return
	}

	// Also return if no candidate
	if !rl.completer.IsInserting() {
		return
	}

	// First insert the current candidate.
	rl.completer.Cancel(false, false)

	// And cycle to the next one.
	rl.completer.Select(1, 0)
}

// Open a completion menu (similar to menu-complete) with all currently populated Vim registers.
func (rl *Shell) viRegistersComplete() {
	rl.undo.SkipSave()

	if !rl.completer.IsActive() {
		rl.startMenuComplete(rl.buffers.Complete)
	} else {
		rl.completer.Select(1, 0)
	}
}

// In a menu completion (wether a candidate is selected or not), start incremental-search
// (fuzzy search) on the results. Search backward incrementally for a specified string.
// The search is case-insensitive if the search string does not have uppercase letters
// and no numeric argument was given. The string may begin with ‘^’ to anchor the search
// to the beginning of the line. A restricted set of editing functions is available in the
// mini-buffer. Keys are looked up in the special isearch keymap, On each change in the
// mini-buffer, any currently selected candidate is dropped from the line and the menu.
// An interrupt signal, as defined by the stty setting, will stop the search and go back to the original line.
func (rl *Shell) menuIncrementalSearch() {
	rl.undo.SkipSave()

	if !rl.completer.IsActive() {
		rl.completer.GenerateWith(rl.commandCompletion)
	}

	rl.completer.IsearchStart("completions", false)
}

//
// Utilities --------------------------------------------------------------------------
//

// startMenuComplete generates a completion menu with completions
// generated from a given completer, without selecting a candidate.
func (rl *Shell) startMenuComplete(completer completion.Completer) {
	rl.undo.SkipSave()

	rl.keymaps.SetLocal(keymap.MenuSelect)
	rl.completer.GenerateWith(completer)
}

// commandCompletion generates the completions for commands/args/flags.
func (rl *Shell) commandCompletion() completion.Values {
	if rl.Completer == nil {
		return completion.Values{}
	}

	line, cursor := rl.completer.Line()
	comps := rl.Completer(*line, cursor.Pos())

	return comps.convert()
}

// historyCompletion manages the various completion/isearch modes related
// to history control. It can start the history completions, stop them, cycle
// through sources if more than one, and adjust the completion/isearch behavior.
func (rl *Shell) historyCompletion(forward, filterLine, incremental bool) {
	switch {
	case rl.keymaps.Local() == keymap.MenuSelect || rl.keymaps.Local() == keymap.Isearch || rl.completer.AutoCompleting():
		// If we are currently completing the last
		// history source, cancel history completion.
		if rl.histories.OnLastSource() {
			rl.histories.Cycle(true)
			rl.completer.ResetForce()
			rl.hint.Reset()

			return
		}

		// Else complete the next history source.
		rl.histories.Cycle(true)

		fallthrough

	default:
		// Notify if we don't have history sources at all.
		if rl.histories.Current() == nil {
			rl.hint.Set(fmt.Sprintf("%s%s%s %s", color.Dim, color.FgRed, "No command history source", color.Reset))
			return
		}

		// Generate the completions with specified behavior.
		completer := func() completion.Values {
			return rl.histories.Complete(forward, filterLine)
		}

		if incremental {
			rl.completer.GenerateWith(completer)
			rl.completer.IsearchStart(rl.histories.Name(), true)
		} else {
			rl.startMenuComplete(completer)
			rl.completer.AutocompleteForce()
		}
	}
}
