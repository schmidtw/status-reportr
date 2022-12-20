// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/goschtalt/goschtalt"
	_ "github.com/goschtalt/yaml-decoder"
	_ "github.com/goschtalt/yaml-encoder"
	gql "github.com/hasura/go-graphql-client"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/oauth2"
	"gopkg.in/dealancer/validate.v2"
)

var errConfig = errors.New("invalid configuration value")

//go:embed default.yml
var defaultConfig string

type CLI struct {
	Debug     bool     `optional:"" help:"Run in debug mode."`
	Show      bool     `optional:"" short:"s" help:"Show the configuration and exit."`
	Files     []string `optional:"" short:"f" name:"file" help:"Specific configuration files or directories."`
	DryRun    bool     `optional:"" help:"When set, items are not archived."`
	CacheFile string   `optional:"" help:"Use a local cache file for testing"`
}

func main() {
	err := wrapped()
	if err != nil {
		fmt.Printf("err: %v\n", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func wrapped() error {
	var cli CLI
	_ = kong.Parse(&cli,
		kong.Name("status-reportr"),
		kong.Description("A status report generator and Github project manager."),
		kong.UsageOnError(),
	)

	gs, err := goschtalt.New(
		goschtalt.DefaultMarshalOptions(
			goschtalt.IncludeOrigins(),
			goschtalt.FormatAs("yml"),
		),
		goschtalt.DefaultUnmarshalOptions(
			goschtalt.WithValidator(validate.Validate),
			goschtalt.ErrorUnused(),
			goschtalt.WeaklyTypedInput(),
			goschtalt.TagName("yaml"),
			goschtalt.DecodeHook(
				mapstructure.StringToTimeDurationHookFunc(),
			),
		),
		goschtalt.AddBuffer("default.yml", []byte(defaultConfig), goschtalt.AsDefault()),
		goschtalt.AddJumbled(os.DirFS("/"), os.DirFS("."), cli.Files...),
		goschtalt.ExpandEnv(),
		goschtalt.AutoCompile(),
	)
	if err != nil {
		return err
	}

	if cli.Show {
		fmt.Fprintln(os.Stdout, gs.Explain())

		out, err := gs.Marshal()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Fprintln(os.Stdout, "---\n"+string(out))
		}
		return nil
	}

	cfg, err := goschtalt.Unmarshal[Config](gs, "")
	if err != nil {
		return err
	}

	cfg.Debug = cli.Debug

	var items Items
	if len(cli.CacheFile) > 0 && fileExist(cli.CacheFile) {
		buf, err := os.ReadFile(cli.CacheFile)
		if err == nil {
			err := json.Unmarshal(buf, &items)
			if err != nil {
				return err
			}
			fmt.Println("Read from disk.")
		}
	} else {
		fmt.Println("Fetching from GH")
		client := login(cfg)
		client = client.WithDebug(true)

		id, err := fetchProjectInfo(cfg.Owner, cfg.Project, client)
		if err != nil {
			return err
		}

		items, err = fetchIssues(id, client,
			cfg.Tuning.IssueCount,
			cfg.Tuning.LabelCount,
			cfg.Tuning.FieldValueCount)
		if err != nil {
			return err
		}
		if len(cli.CacheFile) > 0 {
			buf, err := json.MarshalIndent(items, "", "    ")
			if err != nil {
				return err
			}
			err = os.WriteFile(cli.CacheFile, buf, 0644)
			if err != nil {
				return err
			}
			fmt.Println("Cached to disk.")
		}
	}

	weeks := splitByWeeks(items.GetDone(), time.Now())

	_ = os.Mkdir(cfg.OutputDirectory, 0755)

	for _, week := range weeks {
		data := render(cfg, week)
		filename := fmt.Sprintf("%s-%s.md",
			week.Start.Format("2006.01.02"),
			week.End.AddDate(0, 0, -1).Format("2006.01.02"))

		err = os.WriteFile(filepath.Join(cfg.OutputDirectory, filename), []byte(data), 0644)
		if err != nil {
			return err
		}
	}

	if !cli.DryRun {
		client := login(cfg)
		client = client.WithDebug(true)

		id, err := fetchProjectInfo(cfg.Owner, cfg.Project, client)
		if err != nil {
			return err
		}

		err = archive(id, client, weeks)
		if err != nil {
			return err
		}
	}
	return nil
}

func fileExist(file string) bool {
	if _, err := os.Stat(file); err == nil {
		return true
	}
	return false
}

func login(cfg Config) *gql.Client {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.Token},
	)

	return gql.NewClient(cfg.Url, oauth2.NewClient(context.Background(), src))
}

func render(cfg Config, week WeeklyItems) string {
	sections := make(map[int]string, len(cfg.Sections))

	left := week.Items

	for _, section := range cfg.Sections {
		var buf strings.Builder
		left = section.ExtractAndRender(left, &buf)
		sections[section.RenderOrder] = buf.String()
	}

	if true {
		var buf strings.Builder
		Section{
			Name:        cfg.Unclassified.Name,
			RenderOrder: cfg.Unclassified.RenderOrder,
			OmitIfEmpty: cfg.Unclassified.OmitIfEmpty,
		}.Render(left, &buf)
		sections[cfg.Unclassified.RenderOrder] = buf.String()
	}

	if cfg.LabelSection.Enabled {
		var buf strings.Builder
		fmt.Fprintf(&buf, "\n## By Label\n\n")
		labels := week.Items.GetUniqLabels()
		keys := make([]string, 0, len(labels))
		for key := range labels {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			fmt.Fprintf(&buf, "- %s (%d)\n", key, labels[key])
		}

		sections[cfg.LabelSection.RenderOrder] = buf.String()
	}

	if cfg.Summary.Enabled {
		var buf strings.Builder
		fmt.Fprintf(&buf, "\n## %s\n\n", cfg.Summary.Name)
		fmt.Fprintf(&buf, "%s\n\n", cfg.Summary.Body)
		sections[cfg.Summary.RenderOrder] = buf.String()
	}

	var rv strings.Builder

	fmt.Fprintf(&rv, "# Status Report: %s ... %s\n\n## %s\n\n",
		week.Start.Format("Jan 2, 2006"),
		week.End.AddDate(0, 0, -1).Format("Jan 2, 2006"),
		cfg.Team,
	)

	keys := make([]int, 0, len(sections))
	for key := range sections {
		keys = append(keys, key)
	}
	sort.Ints(keys)

	for _, key := range keys {
		rv.WriteString(sections[key])
	}

	return rv.String()
}

func archive(projectId string, client *gql.Client, weeks []WeeklyItems) error {
	for _, week := range weeks {
		for _, item := range week.Items {
			if err := archiveItem(projectId, item.ID, client); err != nil {
				return err
			}
		}
	}

	return nil
}
