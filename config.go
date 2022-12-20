// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
)

// Config the general program config structure.  See default.yml for usage details.
type Config struct {
	Debug           bool   `yaml:"-"`                                       // If debugging information should be output.
	Url             string `yaml:"url" validate:"format=url"`               // The github url to use.
	Owner           string `yaml:"owner" validate:"empty=false"`            // The github org or owner of the project.
	Token           string `yaml:"token" validate:"empty=false"`            // The github token to use for access.
	Team            string `yaml:"team" validate:"empty=false"`             // The team name.
	Project         int    `yaml:"project_number"`                          // The github project number to work with.
	OutputDirectory string `yaml:"output_directory" validate:"empty=false"` // Where the reports are placed.

	Tuning       Tuning       `yaml:"tuning"`
	ReportWindow ReportWindow `yaml:"report_window"`
	LabelSection LabelSection `yaml:"label_section"`
	Unclassified Unclassified `yaml:"unclassified"`
	Summary      Summary      `yaml:"summary"`
	Sections     []Section    `yaml:"sections"` // User defined sections.
}

// The query tuning parameters.
type Tuning struct {
	IssueCount      int `yaml:"issue_count"`       // The number of issues to fetch in a single query.
	LabelCount      int `yaml:"label_count"`       // The number of labels to fetch in a single query.
	FieldValueCount int `yaml:"field_value_count"` // The number of field values to fetch in a single query.
}

// The report start and stop times to use.
type ReportWindow struct {
	// The number of days per report.  If the value is a multiple of 7, then the
	// StartOnWeekday value is honored if not empty.  Otherwise it is ignored.
	Days int `yaml:"days"`

	// The starting day of the report if not empty string and Days is a multiple
	// of 7.
	StartOnWeekday string `yaml:"start_on_weekday"`
}

// The label section configuration.
type LabelSection struct {
	Enabled     bool `yaml:"enabled"`      // Include the label section if enabled.
	RenderOrder int  `yaml:"render_order"` // The order to render the section relative to the others.
}

// How to handle unclassified items that were missed.
type Unclassified struct {
	Name        string `yaml:"name"`          // The name to use for the section.
	RenderOrder int    `yaml:"render_order"`  // The order to render the section relative to the others.
	OmitIfEmpty bool   `yaml:"omit_if_empty"` // If the section should be present if it is empty.
}

type Summary struct {
	Enabled     bool   `yaml:"enabled"`      // Include the label section if enabled.
	Name        string `yaml:"name"`         // The name to use for the section.
	RenderOrder int    `yaml:"render_order"` // The order to render the section relative to the others.
	Body        string `yaml:"body"`         // The body to populate
}

// Section captures the user configurable section information.
type Section struct {
	Name        string `yaml:"name"`          // The name to use for the section.
	RenderOrder int    `yaml:"render_order"`  // The order to render the section relative to the others.
	OmitIfEmpty bool   `yaml:"omit_if_empty"` // If the section should be present if it is empty.

	Match Match `yaml:"match_on"`
}

// Match defines the matching conditions to use for including an item in a section.
// Matching conditions are a logical OR, so any match includes the item.
type Match struct {
	Labels   []string `yaml:"labels"`   // A list of labels to match against.
	Prefixes []string `yaml:"prefixes"` // A list of prefixes to match against the commit message.

	Branches []Branch `yaml:"branches"`
}

// Branch defines the org/repo and branch to match against.  This allows for easy
// inclusion of specific areas of repos into sections.
type Branch struct {
	Org    string `yaml:"org"`    // The github org/owner value to match.
	Repo   string `yaml:"repo"`   // The github repo value to match.
	Branch string `yaml:"branch"` // The git branch to match.
}

// ExtractAndRender extracts the items that match and renders them to a writer.
// The unconsumed items are returned.
func (s Section) ExtractAndRender(list Items, w io.Writer) Items {
	mine, left := s.Extract(list)
	s.Render(mine, w)

	return left
}

// Extract extracts the items that match and returns the list of matching items
// and non-maching items.
func (s Section) Extract(list Items) (mine, left Items) {
	var tmp Items

	left = list
	tmp, left = left.ExtractByLabels(s.Match.Labels...)
	mine = tmp

	tmp, left = left.ExtractByPrefixes(s.Match.Prefixes...)
	mine = append(mine, tmp...)

	for _, b := range s.Match.Branches {
		tmp, left = left.ExtractByBranch(b.Org, b.Repo, b.Branch)
		mine = append(mine, tmp...)
	}
	return mine, left
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
