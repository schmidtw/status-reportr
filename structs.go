// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"sort"
	"strings"
	"time"

	"github.com/ryanuber/go-glob"
)

// Item represents a github issue, draft issue or pr in an easier to use form.
type Item struct {
	ID       string
	Archived bool
	Fields   map[string]Field
	Labels   []string
	DoneAt   time.Time
	ItemType string // ISSUE, PR
	Number   int
	URL      string
	Repo     struct {
		Name   string
		Slug   string
		URL    string
		Branch string
	}
}

// IsDone returns if the item is complete & is marked "done".
func (it Item) IsDone() bool {
	if status, ok := it.Fields["Status"]; ok {
		return status.Type == FIELD_TEXT && "done" == strings.ToLower(status.Text)
	}
	return false
}

// Done returns the time the item was completed at.
func (it Item) Done() time.Time {
	if status, ok := it.Fields["Status"]; ok {
		if status.Type == FIELD_TEXT && "done" == strings.ToLower(status.Text) {
			return it.DoneAt
		}
	}
	return time.Time{}
}

// HasLabel returns if the item has this label.
func (it Item) HasLabel(l string) bool {
	l = strings.TrimSpace(l)

	for _, label := range it.Labels {
		if glob.Glob(l, strings.TrimSpace(label)) {
			return true
		}
	}
	return false
}

// HasPrefix returns if the item title prefix matches the one specified.
func (it Item) HasPrefix(prefix string) bool {
	return glob.Glob(
		strings.TrimSpace(prefix)+"*",
		strings.TrimSpace(it.Title()),
	)
}

// IsBranch returns if the item is associated with the specified repo/branch.
func (it Item) IsBranch(org, repo, branch string) bool {
	slug := strings.TrimSpace(org) + "/" + strings.TrimSpace(repo)
	branch = strings.TrimSpace(branch)

	return len(it.Repo.Slug) > 0 &&
		len(it.Repo.Branch) > 0 &&
		glob.Glob(slug, strings.TrimSpace(it.Repo.Slug)) &&
		glob.Glob(branch, strings.TrimSpace(it.Repo.Branch))
}

// Title returns the title of the item, or the empty string.
func (it Item) Title() string {
	if status, ok := it.Fields["Title"]; ok {
		if status.Type == FIELD_TEXT {
			return status.Text
		}
	}
	return ""
}

const (
	FIELD_EMPTY int = iota
	FIELD_DATE
	FIELD_TEXT
	FIELD_NUMBER
	FIELD_ITERATION
)

// Field provides a single record that can represent any of the data types that
// can be present.
type Field struct {
	Type int
	Name string

	// One of these is valid.
	Date   time.Time
	Number float64
	Text   string

	// iteration
	Duration    time.Duration
	IterationId string
	StartDate   time.Time
	Title       string
}

// Items provides a handy way to deal with an array of items.
type Items []Item

// GetDone returns the subset list of items that are done.
func (list Items) GetDone() Items {
	var done Items
	for _, item := range list {
		if item.IsDone() {
			done = append(done, item)
		}
	}

	sort.SliceStable(done, func(i, j int) bool {
		return done[i].Done().Before(done[j].Done())
	})

	return done
}

// GetOlder returns the subset list of items that are older or equal to the time.
func (list Items) GetOlder(when time.Time) Items {
	var done Items
	for _, item := range list {
		if at := item.Done(); !at.Equal(time.Time{}) {
			if at.Before(when) || at.Equal(when) {
				done = append(done, item)
			}
		}
	}
	return done
}

// GetInRange returns the subset list of items that are inside the time window.
func (list Items) GetInRange(start, end time.Time) Items {
	if start.After(end) {
		tmp := start
		start = end
		end = tmp
	}

	var done Items
	for _, item := range list {
		at := item.Done()
		switch {
		case at.Before(end) && at.After(start):
			done = append(done, item)
		case at.Equal(start):
			done = append(done, item)
		}
	}
	return done
}

// ExtractByLabels returns the subset list of items have a matching label, and
// a separate list of left over items.
func (list Items) ExtractByLabels(labels ...string) (matching, remaining Items) {
	for _, item := range list {
		var match bool
		for _, label := range labels {
			if item.HasLabel(label) {
				match = true
				break
			}
		}

		if match {
			matching = append(matching, item)
		} else {
			remaining = append(remaining, item)
		}
	}

	return matching, remaining
}

// ExtractByPrefixes returns the subset list of items have a matching prefix, and
// a separate list of left over items.
func (list Items) ExtractByPrefixes(prefixes ...string) (matching, remaining Items) {
	for _, item := range list {
		var match bool
		for _, prefix := range prefixes {
			if item.HasPrefix(prefix) {
				match = true
				break
			}
		}

		if match {
			matching = append(matching, item)
		} else {
			remaining = append(remaining, item)
		}
	}

	return matching, remaining
}

// ExtractByBranch returns the subset list of items have a matching branch, and
// a separate list of left over items.
func (list Items) ExtractByBranch(org, repo, branch string) (matching, remaining Items) {
	for _, item := range list {
		if item.IsBranch(org, repo, branch) {
			matching = append(matching, item)
		} else {
			remaining = append(remaining, item)
		}
	}

	return matching, remaining
}

// GetUniqLabels returns a map of labels and the number of times they were
// encountered in the provided list.
func (list Items) GetUniqLabels() map[string]int {
	rv := make(map[string]int)

	for _, item := range list {
		for _, label := range item.Labels {
			rv[label]++
		}
	}

	return rv
}
