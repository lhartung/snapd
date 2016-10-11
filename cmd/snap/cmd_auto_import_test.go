// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016 Canonical Ltd
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
	"io/ioutil"
	"net/http"
	"path/filepath"

	. "gopkg.in/check.v1"

	snap "github.com/snapcore/snapd/cmd/snap"
	"github.com/snapcore/snapd/logger"
)

func makeMockMountInfo(c *C, content string) string {
	fn := filepath.Join(c.MkDir(), "mountinfo")
	err := ioutil.WriteFile(fn, []byte(content), 0644)
	c.Assert(err, IsNil)
	return fn
}

func (s *SnapSuite) TestAutoImportAssertsHappy(c *C) {
	fakeAssertData := []byte("my-assertion")

	n := 0
	total := 2
	s.RedirectClientToTestServer(func(w http.ResponseWriter, r *http.Request) {
		switch n {
		case 0:
			c.Check(r.Method, Equals, "POST")
			c.Check(r.URL.Path, Equals, "/v2/assertions")
			postData, err := ioutil.ReadAll(r.Body)
			c.Assert(err, IsNil)
			c.Check(postData, DeepEquals, fakeAssertData)
			fmt.Fprintln(w, `{"type": "sync", "result": {"ready": true, "status": "Done"}}`)
			n++
		case 1:
			c.Check(r.Method, Equals, "POST")
			c.Check(r.URL.Path, Equals, "/v2/create-user")
			postData, err := ioutil.ReadAll(r.Body)
			c.Assert(err, IsNil)
			c.Check(string(postData), Equals, `{"sudoer":true,"known":true}`)

			fmt.Fprintln(w, `{"type": "sync", "result": [{"username": "foo"}]}`)
			n++
		default:
			c.Fatalf("unexpected request: %v (expected %d got %d)", r, total, n)
		}

	})

	fakeAssertsFn := filepath.Join(c.MkDir(), "auto-import.assert")
	err := ioutil.WriteFile(fakeAssertsFn, fakeAssertData, 0644)
	c.Assert(err, IsNil)

	mockMountInfoFmt := `
24 0 8:18 / %s rw,relatime shared:1 - ext4 /dev/sdb2 rw,errors=remount-ro,data=ordered`
	content := fmt.Sprintf(mockMountInfoFmt, filepath.Dir(fakeAssertsFn))
	snap.MockMountInfoPath(makeMockMountInfo(c, content))

	l, err := logger.NewConsoleLog(s.stderr, 0)
	c.Assert(err, IsNil)
	logger.SetLogger(l)

	rest, err := snap.Parser().ParseArgs([]string{"auto-import"})
	c.Assert(err, IsNil)
	c.Assert(rest, DeepEquals, []string{})
	c.Check(s.Stdout(), Equals, `created user "foo"`+"\n")
	// matches because we may get a:
	//   "WARNING: cannot create syslog logger\n"
	// in the output
	c.Check(s.Stderr(), Matches, fmt.Sprintf("(?ms).*imported %s\n", fakeAssertsFn))
	c.Check(n, Equals, total)
}

func (s *SnapSuite) TestAutoImportAssertsNotImportedFromLoop(c *C) {
	fakeAssertData := []byte("bad-assertion")

	s.RedirectClientToTestServer(func(w http.ResponseWriter, r *http.Request) {
		// assertion is ignored, nothing is posted to this endpoint
		panic("not reached")
	})

	fakeAssertsFn := filepath.Join(c.MkDir(), "auto-import.assert")
	err := ioutil.WriteFile(fakeAssertsFn, fakeAssertData, 0644)
	c.Assert(err, IsNil)

	mockMountInfoFmtWithLoop := `
24 0 8:18 / %s rw,relatime shared:1 - squashfs /dev/loop1 rw,errors=remount-ro,data=ordered`
	content := fmt.Sprintf(mockMountInfoFmtWithLoop, filepath.Dir(fakeAssertsFn))
	snap.MockMountInfoPath(makeMockMountInfo(c, content))

	rest, err := snap.Parser().ParseArgs([]string{"auto-import"})
	c.Assert(err, IsNil)
	c.Assert(rest, DeepEquals, []string{})
	c.Check(s.Stdout(), Equals, "")
	c.Check(s.Stderr(), Equals, "")
}

func (s *SnapSuite) TestAutoImportCandidatesHappy(c *C) {
	fakeAssertsFn := filepath.Join(c.MkDir(), "auto-import.assert")
	err := ioutil.WriteFile(fakeAssertsFn, nil, 0644)
	c.Assert(err, IsNil)

	mountPoint := filepath.Dir(fakeAssertsFn)
	mockMountInfoFmtWithLoop := `
24 0 8:18 / %[1]s rw,relatime - ext3 /dev/meep2 rw,errors=remount-ro,data=ordered
24 0 8:18 / %[1]s rw,relatime opt:1 - ext4 /dev/meep3 rw,errors=remount-ro,data=ordered
24 0 8:18 / %[1]s rw,relatime opt:1 opt:2 - ext2 /dev/meep1 rw,errors=remount-ro,data=ordered
`

	content := fmt.Sprintf(mockMountInfoFmtWithLoop, mountPoint)
	snap.MockMountInfoPath(makeMockMountInfo(c, content))

	l, err := snap.AutoImportCandidates()
	c.Check(err, IsNil)
	c.Check(l, HasLen, 3)
}

func (s *SnapSuite) TestAutoImportCandidatesMissingSep(c *C) {
	mockMountInfo := `
24 0 8:18 / /mount/point rw,relatime invalid line missing the minus
`
	snap.MockMountInfoPath(makeMockMountInfo(c, mockMountInfo))

	_, err := snap.AutoImportCandidates()
	c.Check(err, ErrorMatches, `cannot parse line ".*": no separator '-' found`)
}

func (s *SnapSuite) TestAutoImportCandidatesTooShort(c *C) {
	mockMountInfo := `
too short
`
	snap.MockMountInfoPath(makeMockMountInfo(c, mockMountInfo))

	_, err := snap.AutoImportCandidates()
	c.Check(err, ErrorMatches, `cannot parse line ".*": too short`)
}