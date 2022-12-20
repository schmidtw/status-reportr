// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtract(t *testing.T) {
	tests := []struct {
		description string
		section     Section
		expectMine  Items
		expectLeft  Items
	}{
		{
			description: "extract by label",
			section: Section{
				Match: Match{
					Labels: []string{"dogs", "deployment"},
				},
			},
			expectMine: Items{itemIssue88, itemIssue89},
			expectLeft: Items{itemPr24, itemPr23},
		}, {
			description: "extract by label glob",
			section: Section{
				Match: Match{
					Labels: []string{"*"},
				},
			},
			expectMine: Items{itemIssue88, itemIssue89},
			expectLeft: Items{itemPr24, itemPr23},
		}, {
			description: "extract by prefix",
			section: Section{
				Match: Match{
					Prefixes: []string{"Update Something"},
				},
			},
			expectMine: Items{itemPr24, itemPr23},
			expectLeft: Items{itemIssue88, itemIssue89},
		}, {
			description: "extract by globbed prefix",
			section: Section{
				Match: Match{
					Prefixes: []string{"Up*"},
				},
			},
			expectMine: Items{itemPr24, itemPr23},
			expectLeft: Items{itemIssue88, itemIssue89},
		}, {
			description: "extract by branch",
			section: Section{
				Match: Match{
					Branches: []Branch{
						{
							Org:    "*",
							Repo:   "*",
							Branch: "*",
						},
					},
				},
			},
			expectMine: Items{itemPr24, itemPr23},
			expectLeft: Items{itemIssue88, itemIssue89},
		}, {
			description: "extract none by branch",
			section: Section{
				Match: Match{
					Branches: []Branch{
						{
							Org:    "no-match",
							Repo:   "*",
							Branch: "*",
						},
					},
				},
			},
			expectLeft: Items{itemPr24, itemIssue88, itemIssue89, itemPr23},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			items := Items{itemPr24, itemIssue88, itemIssue89, itemPr23}

			mine, left := tc.section.Extract(items)

			require.Equal(len(tc.expectMine), len(mine))
			require.Equal(len(tc.expectLeft), len(left))
			assert.ElementsMatch(tc.expectMine, mine)
			assert.ElementsMatch(tc.expectLeft, left)
		})
	}
}
