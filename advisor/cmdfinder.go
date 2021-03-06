// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2018 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package advisor

import (
	"fmt"
)

var commandsFinder Finder = &boltFinder{}

type Command struct {
	Snap    string
	Command string
}

func FindCommand(command string) ([]Command, error) {
	return commandsFinder.Find(command)
}

const (
	minLen = 3
	maxLen = 256
)

// based on CommandNotFound.py:similar_words.py
func similarWords(word string) []string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz-_0123456789"
	similar := make(map[string]bool, 2*len(word)+2*len(word)*len(alphabet))

	// deletes
	for i := range word {
		similar[word[:i]+word[i+1:]] = true
	}
	// transpose
	for i := 0; i < len(word)-1; i++ {
		similar[word[:i]+word[i+1:i+2]+word[i:i+1]+word[i+2:]] = true
	}
	// replaces
	for i := range word {
		for _, r := range alphabet {
			similar[word[:i]+string(r)+word[i+1:]] = true
		}
	}
	// inserts
	for i := range word {
		for _, r := range alphabet {
			similar[word[:i]+string(r)+word[i:]] = true
		}
	}

	// convert for output
	ret := make([]string, 0, len(similar))
	for w := range similar {
		ret = append(ret, w)
	}

	return ret
}

func FindMispelledCommand(command string) ([]Command, error) {
	if len(command) < minLen || len(command) > maxLen {
		return nil, nil
	}
	alternatives := make([]Command, 0, 32)
	for _, w := range similarWords(command) {
		res, err := commandsFinder.Find(w)
		if err != nil {
			return nil, err
		}
		if len(res) > 0 {
			alternatives = append(alternatives, res...)
		}
	}

	return alternatives, nil
}

type Finder interface {
	Find(command string) ([]Command, error)
}

func ReplaceCommandsFinder(f Finder) (restore func()) {
	old := commandsFinder
	commandsFinder = f
	return func() {
		commandsFinder = old
	}
}

type boltFinder struct{}

func (bf *boltFinder) Find(command string) ([]Command, error) {
	return nil, fmt.Errorf("not implemented")
}
