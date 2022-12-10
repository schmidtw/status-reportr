// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Config the general program config structure.  See default.yml for usage details.
type Config struct {
	Debug           bool   `yaml:"-"`                // If debugging information should be output.
	Url             string `yaml:"url"`              // The github url to use.
	Owner           string `yaml:"owner"`            // The github org or owner of the project.
	Token           string `yaml:"token"`            // The github token to use for access.
	Project         int    `yaml:"project_number"`   // The github project number to work with.
	OutputDirectory string `yaml:"output_directory"` // Where the reports are placed.

	// The query tuning parameters.
	Tuning struct {
		IssueCount      int `yaml:"issue_count"`       // The number of issues to fetch in a single query.
		LabelCount      int `yaml:"label_count"`       // The number of labels to fetch in a single query.
		FieldValueCount int `yaml:"field_value_count"` // The number of field values to fetch in a single query.
	} `yaml:"tuning"`

	// The report start and stop times to use.
	ReportWindow struct {
		// The number of days per report.  If the value is a multiple of 7, then the
		// StartOnWeekday value is honored if not empty.  Otherwise it is ignored.
		Days int `yaml:"days"`

		// The starting day of the report if not empty string and Days is a multiple
		// of 7.
		StartOnWeekday string `yaml:"start_on_weekday"`
	} `yaml:"report_window"`

	// The label section configuration.
	LabelSection struct {
		Enabled     bool `yaml:"enabled"`      // Include the label section if enabled.
		RenderOrder int  `yaml:"render_order"` // The order to render the section relative to the others.
	} `yaml:"label_section"`

	// How to handle unclassified items that were missed.
	Unclassified struct {
		Name        string `yaml:"name"`          // The name to use for the section.
		RenderOrder int    `yaml:"render_order"`  // The order to render the section relative to the others.
		OmitIfEmpty bool   `yaml:"omit_if_empty"` // If the section should be present if it is empty.
	} `yaml:"unclassified"`

	Sections []Section `yaml:"sections"` // User defined sections.
}

// Section captures the user configurable section information.
type Section struct {
	Name        string `yaml:"name"`          // The name to use for the section.
	RenderOrder int    `yaml:"render_order"`  // The order to render the section relative to the others.
	OmitIfEmpty bool   `yaml:"omit_if_empty"` // If the section should be present if it is empty.

	// Match defines the matching conditions to use for including an item in a section.
	// Matching conditions are a logical OR, so any match includes the item.
	Match struct {
		Label  []string `yaml:"labels"`   // A list of labels to match against.
		Prefix []string `yaml:"prefixes"` // A list of prefixes to match against the commit message.

		// Branch defines the org/repo and branch to match against.  This allows for easy
		// inclusion of specific areas of repos into sections.
		Branch []struct {
			Org    string `yaml:"org"`    // The github org/owner value to match.
			Repo   string `yaml:"repo"`   // The github repo value to match.
			Branch string `yaml:"branch"` // The git branch to match.
		} `yaml:"branches"`
	} `yaml:"match_on"`
}

// ExtractAndRender extracts the items that match and renders them to a writer.
// The unconsumed items are returned.
func (s Section) ExtractAndRender(list Items, w io.Writer) Items {
	var tmp Items
	var mine Items

	left := list
	tmp, left = left.ExtractByLabels(s.Match.Label...)
	mine = tmp

	tmp, left = left.ExtractByPrefixes(s.Match.Prefix...)
	mine = append(mine, tmp...)

	for _, branch := range s.Match.Branch {
		branchRe := strings.TrimSpace(branch.Branch)
		re, err := regexp.Compile(branchRe)
		if err != nil {
			str := fmt.Sprintf("section: %s has a branch with an invalid regex.  Org: '%s', Repo: '%s', Branch: '%s'.  Err: %s\n",
				s.Name, branch.Org, branch.Repo, branch, err)
			panic(str)
		}
		tmp, left = left.ExtractByBranch(branch.Org, branch.Repo, re)
		mine = append(mine, tmp...)
	}

	s.Render(mine, w)

	return left
}

// Render converts a list of items into a markdown document section.
func (s Section) Render(list Items, w io.Writer) {
	if s.OmitIfEmpty && len(list) == 0 {
		return
	}

	fmt.Fprintf(w, "\n## %s (%d)\n\n", s.Name, len(list))
	for _, item := range list {
		fmt.Fprintf(w, "- %s **[[#%d](%s)]** ([%s](%s))\n", item.Title(), item.Number, item.URL, item.Repo.Slug, item.Repo.URL)
	}
}
