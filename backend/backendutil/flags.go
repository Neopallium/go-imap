package backendutil

import (
	"github.com/emersion/go-imap"
)

// UpdateFlags executes a flag operation on the flag set current.
func UpdateFlags(current []string, op imap.FlagsOp, flags []string) []string {
	// Don't modify contents of 'flags' slice.  Only modify 'current'.
	// See https://github.com/golang/go/wiki/SliceTricks

	// Re-use current's backing store
	newFlags := current[:0]
	switch op {
	case imap.SetFlags:
		// keep recent flag
		for _, flag := range current {
			if flag == imap.RecentFlag {
				newFlags = append(newFlags, imap.RecentFlag)
				break
			}
		}
		// append new flags
		newFlags = append(newFlags, flags...)
	case imap.AddFlags:
		// keep current flags
		newFlags = current
		// Only add new flag if it isn't already in current list.
		for _, addFlag := range flags {
			found := false
			for _, flag := range current {
				if addFlag == flag {
					found = true
					break
				}
			}
			// new flag not found, add it.
			if !found {
				newFlags = append(newFlags, addFlag)
			}
		}
	case imap.RemoveFlags:
		// Filter current flags
		for _, flag := range current {
			remove := false
			for _, removeFlag := range flags {
				if removeFlag == flag {
					remove = true
				}
			}
			if !remove {
				newFlags = append(newFlags, flag)
			}
		}
	default:
		// Unknown operation, return current flags unchanged
		newFlags = current
	}
	return newFlags
}
