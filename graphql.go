// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"time"

	gql "github.com/hasura/go-graphql-client"
)

// -----------------------------------------------------------------------------
//
// All the data structures below this line are Graphql focused & are designed:
//
//	- to match the structure of the data & query
//  - to be short lived and coverted into one of the structures above
//
// -----------------------------------------------------------------------------

// FieldCommon is a graphql focused structure for collecting the data.
type FieldCommon struct {
	Ignored struct {
		Name      string
		UpdatedAt time.Time
	} `graphql:"... on ProjectV2FieldCommon"`
}

// FieldTextValue is a graphql focused structure for collecting text field data.
type FieldTextValue struct {
	Field *FieldCommon
	Text  *string
}

// Get returns a simplified Field struct version of this object.
func (v FieldTextValue) Get() Field {
	if v.Field == nil || v.Text == nil {
		return Field{}
	}
	return Field{
		Type:      FIELD_TEXT,
		UpdatedAt: v.Field.Ignored.UpdatedAt,
		Name:      v.Field.Ignored.Name,
		Text:      *v.Text,
	}
}

// FieldDateValue is a graphql focused structure for collecting date field data.
type FieldDateValue struct {
	Field *FieldCommon
	Date  *time.Time
}

// Get returns a simplified Field struct version of this object.
func (v FieldDateValue) Get() Field {
	if v.Field == nil || v.Date == nil {
		return Field{}
	}
	return Field{
		Type:      FIELD_DATE,
		UpdatedAt: v.Field.Ignored.UpdatedAt,
		Name:      v.Field.Ignored.Name,
		Date:      *v.Date,
	}
}

// FieldNumberValue is a graphql focused structure for collecting date field data.
type FieldNumberValue struct {
	Field  *FieldCommon
	Number *float64
}

// Get returns a simplified Field struct version of this object.
func (v FieldNumberValue) Get() Field {
	if v.Field == nil || v.Number == nil {
		return Field{}
	}
	return Field{
		Type:      FIELD_NUMBER,
		UpdatedAt: v.Field.Ignored.UpdatedAt,
		Name:      v.Field.Ignored.Name,
		Number:    *v.Number,
	}
}

// FieldSingleSelectValue is a graphql focused structure for collecting date field data.
type FieldSingleSelectValue struct {
	Field *FieldCommon
	Name  *string
}

// Get returns a simplified Field struct version of this object.
func (v FieldSingleSelectValue) Get() Field {
	if v.Field == nil || v.Name == nil {
		return Field{}
	}
	return Field{
		Type:      FIELD_TEXT,
		UpdatedAt: v.Field.Ignored.UpdatedAt,
		Name:      v.Field.Ignored.Name,
		Text:      *v.Name,
	}
}

// FieldIterationValue is a graphql focused structure for collecting date field data.
type FieldIterationValue struct {
	Field       *FieldCommon
	Duration    int // days
	IterationId string
	StartDate   *time.Time
	Title       string
}

// Get returns a simplified Field struct version of this object.
func (v FieldIterationValue) Get() Field {
	if v.Field == nil || v.StartDate == nil {
		return Field{}
	}
	return Field{
		Type:        FIELD_ITERATION,
		UpdatedAt:   v.Field.Ignored.UpdatedAt,
		Name:        v.Field.Ignored.Name,
		Duration:    time.Hour * 24 * time.Duration(v.Duration),
		IterationId: v.IterationId,
		StartDate:   *v.StartDate,
		Title:       v.Title,
	}
}

// Label is a graphql focused structure for collecting date field data.
type Label struct {
	Name string
}

// FieldLabelValue is a graphql focused structure for collecting date field data.
type FieldLabelValue struct {
	Labels struct {
		Nodes []Label
	} `graphql:"labels(first: $labelCount)"`
}

// DraftIssue is a graphql focused structure for collecting date field data.
type DraftIssue struct {
	DraftIssue struct {
		UpdatedAt *time.Time
	} `graphql:"... on DraftIssue"`
}

// Issue is a graphql focused structure for collecting date field data.
type Issue struct {
	Issue struct {
		UpdatedAt  *time.Time
		Number     int
		URL        string
		Repository struct {
			Name          string
			NameWithOwner string
			URL           string
		}
	} `graphql:"... on Issue"`
}

// PullRequest is a graphql focused structure for collecting date field data.
type PullRequest struct {
	PullRequest struct {
		UpdatedAt   *time.Time
		Number      int
		URL         string
		BaseRefName string
		Repository  struct {
			Name          string
			NameWithOwner string
			URL           string
		}
	} `graphql:"... on PullRequest"`
}

// GqlItem is a graphql focused structure for collecting date field data.
type GqlItem struct {
	ID          string
	IsArchived  bool
	FieldValues struct {
		Nodes []struct {
			DateValue      FieldDateValue         `graphql:"... on ProjectV2ItemFieldDateValue"`
			IterationValue FieldIterationValue    `graphql:"... on ProjectV2ItemFieldIterationValue"`
			Labels         FieldLabelValue        `graphql:"... on ProjectV2ItemFieldLabelValue"`
			NumberValue    FieldNumberValue       `graphql:"... on ProjectV2ItemFieldNumberValue"`
			SelectValue    FieldSingleSelectValue `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
			TextValue      FieldTextValue         `graphql:"... on ProjectV2ItemFieldTextValue"`
		}
	} `graphql:"fieldValues(first: $fieldValuesCount)"`
	DI    DraftIssue  `graphql:"di:content"`
	Issue Issue       `graphql:"iss:content"`
	PR    PullRequest `graphql:"pr:content"`
}

// ToClean takes an item from github and normalizes it into the simplified Item
// structure.
func (g GqlItem) ToClean() Item {
	rv := Item{
		ID:       g.ID,
		Archived: g.IsArchived,
		Fields:   make(map[string]Field, len(g.FieldValues.Nodes)),
	}

	if g.DI.DraftIssue.UpdatedAt != nil {
		rv.UpdatedAt = *g.DI.DraftIssue.UpdatedAt
		rv.ItemType = "DRAFT_ISSUE"
	}
	if g.Issue.Issue.UpdatedAt != nil {
		rv.UpdatedAt = *g.Issue.Issue.UpdatedAt
		rv.ItemType = "ISSUE"
		rv.Number = g.Issue.Issue.Number
		rv.URL = g.Issue.Issue.URL
		rv.Repo.Name = g.Issue.Issue.Repository.Name
		rv.Repo.Slug = g.Issue.Issue.Repository.NameWithOwner
		rv.Repo.URL = g.Issue.Issue.Repository.URL
	}
	if g.PR.PullRequest.UpdatedAt != nil {
		rv.UpdatedAt = *g.PR.PullRequest.UpdatedAt
		rv.ItemType = "PR"
		rv.Number = g.PR.PullRequest.Number
		rv.URL = g.PR.PullRequest.URL
		rv.Repo.Name = g.PR.PullRequest.Repository.Name
		rv.Repo.Slug = g.PR.PullRequest.Repository.NameWithOwner
		rv.Repo.URL = g.PR.PullRequest.Repository.URL
		rv.Repo.Branch = g.PR.PullRequest.BaseRefName
	}

	for _, n := range g.FieldValues.Nodes {
		f := n.DateValue.Get()
		if f.Type != FIELD_EMPTY {
			rv.Fields[f.Name] = f
		}
		f = n.IterationValue.Get()
		if f.Type != FIELD_EMPTY {
			rv.Fields[f.Name] = f
		}
		f = n.NumberValue.Get()
		if f.Type != FIELD_EMPTY {
			rv.Fields[f.Name] = f
		}
		f = n.SelectValue.Get()
		if f.Type != FIELD_EMPTY {
			rv.Fields[f.Name] = f
		}
		f = n.TextValue.Get()
		if f.Type != FIELD_EMPTY {
			rv.Fields[f.Name] = f
		}

		for _, l := range n.Labels.Labels.Nodes {
			rv.Labels = append(rv.Labels, l.Name)
		}
	}

	return rv
}

// fetchProjectInfo uses the configuration provided owner/org and project number
// and gets the id to use.
func fetchProjectInfo(cfg Config, client *gql.Client) (string, error) {
	vars := map[string]any{
		"owner":  cfg.Owner,
		"number": cfg.Project,
	}
	var query struct {
		Organization struct {
			ProjectV2 struct {
				Id string
			} `graphql:"projectV2(number: $number)"`
		} `graphql:"organization(login: $owner)"`
	}

	if cfg.Debug {
		str, err := gql.ConstructQuery(&query, vars)
		if err != nil {
			return "", err
		}
		fmt.Printf("Fetching Project Info Request:\n%s\n", str)
	}
	if err := client.Query(context.Background(), &query, vars); err != nil {
		return "", err
	}

	return query.Organization.ProjectV2.Id, nil
}

func fetchIssues(cfg Config, id string, client *gql.Client) (Items, error) {
	var items Items

	vars := map[string]any{
		"count":            cfg.Tuning.IssueCount,
		"labelCount":       cfg.Tuning.LabelCount,
		"fieldValuesCount": cfg.Tuning.FieldValueCount,
		"projectId":        gql.ID(id),
		"after":            (*string)(nil),
	}

	more := true
	for more {
		var query struct {
			Node struct {
				ProjectV2 struct {
					Items struct {
						Nodes    []GqlItem
						PageInfo struct {
							HasNextPage bool
							EndCursor   string
						}
					} `graphql:"items(first: $count, after: $after)"`
				} `graphql:"... on ProjectV2"`
			} `graphql:"node(id: $projectId)"`
		}

		if cfg.Debug {
			str, err := gql.ConstructQuery(&query, vars)
			if err != nil {
				return nil, err
			}
			fmt.Printf("Fetching Issues Request:\n%s\n", str)
		}
		if err := client.Query(context.Background(), &query, vars); err != nil {
			return nil, err
		}

		for _, n := range query.Node.ProjectV2.Items.Nodes {
			items = append(items, n.ToClean())
		}

		more = query.Node.ProjectV2.Items.PageInfo.HasNextPage
		vars["after"] = query.Node.ProjectV2.Items.PageInfo.EndCursor
	}

	return items, nil
}

func archiveItem(cfg Config, projectId, itemId string, client *gql.Client) error {
	vars := map[string]any{
		"projectId": gql.ID(projectId),
		"id":        gql.ID(itemId),
	}
	var mutation struct {
		ArchiveProjectV2ItemPayload struct {
			ClientMutationId string
		} `graphql:"archiveProjectV2Item(input: {projectId: $projectId, itemId: $id})"`
	}
	if cfg.Debug {
		str, err := gql.ConstructMutation(&mutation, vars)
		if err != nil {
			return err
		}
		fmt.Printf("Archiving an Item Request:\n%s\n", str)
	}
	return client.Mutate(context.Background(), &mutation, vars)
}

func unarchiveItem(cfg Config, projectId, itemId string, client *gql.Client) error {
	vars := map[string]any{
		"projectId": gql.ID(projectId),
		"id":        gql.ID(itemId),
	}
	var mutation struct {
		ArchiveProjectV2ItemPayload struct {
			ClientMutationId string
		} `graphql:"unarchiveProjectV2Item(input: {projectId: $projectId, itemId: $id})"`
	}

	if cfg.Debug {
		str, err := gql.ConstructMutation(&mutation, vars)
		if err != nil {
			return err
		}
		fmt.Printf("Unarchiving an Item Request:\n%s\n", str)
	}
	return client.Mutate(context.Background(), &mutation, vars)
}
