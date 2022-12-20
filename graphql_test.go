// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	gql "github.com/hasura/go-graphql-client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	issue88 = `
{
  "data": {
    "node": {
      "items": {
        "nodes": [
          {
            "id": "some-id",
            "isArchived": false,
            "fieldValues": {
              "nodes": [
                {},
                {},
                {
                  "labels": {
                    "nodes": [
                      {
                        "name": "deployment"
                      }
                    ]
                  }
                },
				{
                  "field": {
                    "name": "Title"
                  },
                  "text": "An example item title."
                },
				{
                  "field": {
                    "name": "Status"
                  },
                  "name": "Todo"
                }
              ]
            },
            "iss": {
              "closedAt": "2022-08-04T22:16:25Z",
              "number": 88,
              "url": "https://github.com/org/repo/issues/88",
              "repository": {
                "name": "repo",
                "nameWithOwner": "org/repo",
                "url": "https://github.com/org/repo"
              }
            },
            "pr": {}
          }
        ],
        "pageInfo": {
          "hasNextPage": false,
          "endCursor": "MQ"
        }
      }
    }
  }
}`

	issue89 = `
{
  "data": {
    "node": {
      "items": {
        "nodes": [
          {
            "id": "some-id",
            "isArchived": false,
            "fieldValues": {
              "nodes": [
				{
                  "field": {
                    "name": "Goal"
                  },
				  "date": "2022-08-05T20:19:08Z"
                },
				{
                  "field": {
                    "name": "Iteration"
                  },
				  "duration": 14,
				  "iterationId": "random id",
				  "startDate": "2022-08-05T20:19:08Z",
				  "title": "title"
                },
                {
                  "labels": {
                    "nodes": [
                      {
                        "name": "deployment"
                      }
                    ]
                  }
                },
				{
                  "field": {
                    "name": "Title"
                  },
                  "text": "An example item title."
                },
				{
                  "field": {
                    "name": "Priority"
                  },
				  "number": 123.456
                },
				{
                  "field": {
                    "name": "Status"
                  },
                  "name": "Todo"
                }
              ]
            },
            "iss": {
              "closedAt": "2022-08-04T22:16:25Z",
              "number": 89,
              "url": "https://github.com/org/repo/issues/89",
              "repository": {
                "name": "repo",
                "nameWithOwner": "org/repo",
                "url": "https://github.com/org/repo"
              }
            },
            "pr": {}
          }
        ],
        "pageInfo": {
          "hasNextPage": false,
          "endCursor": "MQ"
        }
      }
    }
  }
}`

	pr23 = `
{
  "data": {
    "node": {
      "items": {
        "nodes": [
          {
            "id": "id123",
            "isArchived": false,
            "fieldValues": {
              "nodes": [
                {},
                {
                  "field": {
                    "name": "Title"
                  },
                  "text": "Update Something"
                },
                {
                  "field": {
                    "name": "Status"
                  },
                  "name": "Todo"
                }
              ]
            },
            "iss": {},
            "pr": {
              "closedAt": "2022-12-01T09:01:53Z",
              "number": 23,
              "url": "https://github.com/org/repo/pull/23",
              "baseRefName": "main",
              "repository": {
                "name": "repo",
                "nameWithOwner": "org/repo",
                "url": "https://github.com/org/repo"
              }
            }
          }
        ]
      }
    }
  }
}`

	pr24 = `
{
  "data": {
    "node": {
      "items": {
        "nodes": [
          {
            "id": "id124",
            "isArchived": false,
            "fieldValues": {
              "nodes": [
                {},
                {
                  "field": {
                    "name": "Title"
                  },
                  "text": "Update Something"
                },
                {
                  "field": {
                    "name": "Status"
                  },
                  "name": "Todo"
                }
              ]
            },
            "iss": {},
            "pr": {
              "mergedAt": "2022-12-01T09:01:53Z",
              "number": 24,
              "url": "https://github.com/org/repo/pull/24",
              "baseRefName": "main",
              "repository": {
                "name": "repo",
                "nameWithOwner": "org/repo",
                "url": "https://github.com/org/repo"
              }
            }
          }
        ]
      }
    }
  }
}`
)

var itemIssue88 = Item{
	ID: "some-id",
	Fields: map[string]Field{
		"Title": Field{
			Type: FIELD_TEXT,
			Name: "Title",
			Text: "An example item title.",
		},
		"Status": Field{
			Type: FIELD_TEXT,
			Name: "Status",
			Text: "Todo",
		},
	},
	Labels:   []string{"deployment"},
	DoneAt:   mustParseTime("2022-08-04T22:16:25Z"),
	ItemType: "ISSUE",
	Number:   88,
	URL:      "https://github.com/org/repo/issues/88",
	Repo: struct {
		Name   string
		Slug   string
		URL    string
		Branch string
	}{
		Name: "repo",
		Slug: "org/repo",
		URL:  "https://github.com/org/repo",
	},
}

var itemIssue89 = Item{
	ID: "some-id",
	Fields: map[string]Field{
		"Title": Field{
			Type: FIELD_TEXT,
			Name: "Title",
			Text: "An example item title.",
		},
		"Iteration": Field{
			Type:        FIELD_ITERATION,
			Name:        "Iteration",
			Title:       "title",
			Duration:    time.Hour * 24 * 14,
			IterationId: "random id",
			StartDate:   mustParseTime("2022-08-05T20:19:08Z"),
		},
		"Goal": Field{
			Type: FIELD_DATE,
			Name: "Goal",
			Date: mustParseTime("2022-08-05T20:19:08Z"),
		},
		"Priority": Field{
			Type:   FIELD_NUMBER,
			Name:   "Priority",
			Number: 123.456,
		},
		"Status": Field{
			Type: FIELD_TEXT,
			Name: "Status",
			Text: "Todo",
		},
	},
	Labels:   []string{"deployment"},
	DoneAt:   mustParseTime("2022-08-04T22:16:25Z"),
	ItemType: "ISSUE",
	Number:   89,
	URL:      "https://github.com/org/repo/issues/89",
	Repo: struct {
		Name   string
		Slug   string
		URL    string
		Branch string
	}{
		Name: "repo",
		Slug: "org/repo",
		URL:  "https://github.com/org/repo",
	},
}

var itemPr23 = Item{
	ID: "id123",
	Fields: map[string]Field{
		"Title": Field{
			Type: FIELD_TEXT,
			Name: "Title",
			Text: "Update Something",
		},
		"Status": Field{
			Type: FIELD_TEXT,
			Name: "Status",
			Text: "Todo",
		},
	},
	DoneAt:   mustParseTime("2022-12-01T09:01:53Z"),
	ItemType: "PR",
	Number:   23,
	URL:      "https://github.com/org/repo/pull/23",
	Repo: struct {
		Name   string
		Slug   string
		URL    string
		Branch string
	}{
		Name:   "repo",
		Slug:   "org/repo",
		URL:    "https://github.com/org/repo",
		Branch: "main",
	},
}

var itemPr24 = Item{
	ID: "id124",
	Fields: map[string]Field{
		"Title": Field{
			Type: FIELD_TEXT,
			Name: "Title",
			Text: "Update Something",
		},
		"Status": Field{
			Type: FIELD_TEXT,
			Name: "Status",
			Text: "Todo",
		},
	},
	DoneAt:   mustParseTime("2022-12-01T09:01:53Z"),
	ItemType: "PR",
	Number:   24,
	URL:      "https://github.com/org/repo/pull/24",
	Repo: struct {
		Name   string
		Slug   string
		URL    string
		Branch string
	}{
		Name:   "repo",
		Slug:   "org/repo",
		URL:    "https://github.com/org/repo",
		Branch: "main",
	},
}

func mustParseTime(timeString string) time.Time {
	t, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		panic(err)
	}
	return t
}

func TestFetchIssues(t *testing.T) {
	unknown := errors.New("unknown")
	tests := []struct {
		description string
		responses   []string
		expect      Items
		expectErr   error
	}{
		{
			description: "basic test issue",
			responses:   []string{issue88},
			expect:      Items{itemIssue88},
		}, {
			description: "basic test issue with more types",
			responses:   []string{issue89},
			expect:      Items{itemIssue89},
		}, {
			description: "basic test pr",
			responses:   []string{pr23},
			expect:      Items{itemPr23},
		}, {
			description: "basic test pr alt date",
			responses:   []string{pr24},
			expect:      Items{itemPr24},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			var i int
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				//body, _ := io.ReadAll(r.Body)
				//fmt.Println(string(body))
				io.Copy(io.Discard, r.Body)
				r.Body.Close()

				require.True(i < len(tc.responses))
				fmt.Fprintln(w, tc.responses[i])
				i++
			}))
			defer ts.Close()

			items, err := fetchIssues("id", gql.NewClient(ts.URL, nil), 10, 10, 10)

			if errors.Is(tc.expectErr, unknown) {
				assert.Nil(items)
				assert.Error(err)
				return
			}

			assert.NoError(err)
			assert.Empty(cmp.Diff(tc.expect, items))
		})
	}
}

func TestFetchProjectInfo(t *testing.T) {
	unknown := errors.New("unknown")
	tests := []struct {
		description string
		owner       string
		project     int
		responses   []string
		expect      string
		expectErr   error
	}{
		{
			description: "basic test issue",
			owner:       "example",
			project:     55,
			responses: []string{`
{
  "data": {
    "organization": {
      "projectV2": {
        "id": "projectId"
      }
    }
  }
}`,
			},
			expect: "projectId",
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			var i int
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				//body, _ := io.ReadAll(r.Body)
				//fmt.Println(string(body))
				io.Copy(io.Discard, r.Body)
				r.Body.Close()

				require.True(i < len(tc.responses))
				fmt.Fprintln(w, tc.responses[i])
				i++
			}))
			defer ts.Close()

			got, err := fetchProjectInfo(tc.owner, tc.project, gql.NewClient(ts.URL, nil))

			if errors.Is(tc.expectErr, unknown) {
				assert.Equal("", got)
				assert.Error(err)
				return
			}

			assert.NoError(err)
			assert.Equal(tc.expect, got)
		})
	}
}

func TestArchiveItem(t *testing.T) {
	unknown := errors.New("unknown")
	tests := []struct {
		description string
		project     string
		item        string
		responses   []string
		expectErr   error
	}{
		{
			description: "basic test issue",
			project:     "id-55",
			item:        "item3",
			responses: []string{`
{
	"data": {
		"archiveProjectV2Item": {
			"clientMutationId":null
		}
	}
}`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			var i int
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				//body, _ := io.ReadAll(r.Body)
				//fmt.Println(string(body))
				io.Copy(io.Discard, r.Body)
				r.Body.Close()

				require.True(i < len(tc.responses))
				fmt.Fprintln(w, tc.responses[i])
				i++
			}))
			defer ts.Close()

			err := archiveItem(tc.project, tc.item, gql.NewClient(ts.URL, nil))

			if errors.Is(tc.expectErr, unknown) {
				assert.Error(err)
				return
			}

			assert.NoError(err)
		})
	}
}
