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

package main_test

import (
	"fmt"

	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/advisor"
	snap "github.com/snapcore/snapd/cmd/snap"
)

type sillyFinder struct{}

func (sf *sillyFinder) Find(command string) ([]advisor.Command, error) {
	switch command {
	case "hello":
		return []advisor.Command{
			{Snap: "hello", Command: "hello"},
			{Snap: "hello-wcm", Command: "hello"},
		}, nil
	case "error-please":
		return nil, fmt.Errorf("get failed")
	default:
		return nil, nil
	}
}

func (s *SnapSuite) TestAdviceCommandHappyText(c *C) {
	restore := advisor.ReplaceCommandsFinder(&sillyFinder{})
	defer restore()

	rest, err := snap.Parser().ParseArgs([]string{"advice-command", "hello"})
	c.Assert(err, IsNil)
	c.Assert(rest, DeepEquals, []string{})
	c.Assert(s.Stdout(), Equals, `The program "hello" can be found in the following snaps:
 * hello
 * hello-wcm
Try: snap install <selected snap>
`)
	c.Assert(s.Stderr(), Equals, "")
}

func (s *SnapSuite) TestAdviceCommandHappyJSON(c *C) {
	restore := advisor.ReplaceCommandsFinder(&sillyFinder{})
	defer restore()

	rest, err := snap.Parser().ParseArgs([]string{"advice-command", "--format=json", "hello"})
	c.Assert(err, IsNil)
	c.Assert(rest, DeepEquals, []string{})
	c.Assert(s.Stdout(), Equals, `[{"Snap":"hello","Command":"hello"},{"Snap":"hello-wcm","Command":"hello"}]`+"\n")
	c.Assert(s.Stderr(), Equals, "")
}

func (s *SnapSuite) TestAdviceCommandMisspellText(c *C) {
	restore := advisor.ReplaceCommandsFinder(&sillyFinder{})
	defer restore()

	for _, misspelling := range []string{"helo", "0hello", "hell0", "hello0"} {
		err := snap.AdviceCommand(misspelling, "pretty")
		c.Assert(err, IsNil)
		c.Assert(s.Stdout(), Equals, fmt.Sprintf(`No command "%s" found, did you mean:
 Command "hello" from snap "hello"
 Command "hello" from snap "hello-wcm"
`, misspelling))
		c.Assert(s.Stderr(), Equals, "")

		s.stdout.Reset()
		s.stderr.Reset()
	}
}
