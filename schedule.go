// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"sort"
	"time"
)

type WeeklyItems struct {
	Items Items
	Start time.Time
	End   time.Time
}

func splitByWeeks(list Items, now time.Time) []WeeklyItems {
	var weeks []WeeklyItems

	end := getClosestSunday(now)
	start := getPreviousSunday(end)

	sort.SliceStable(list,
		func(i, j int) bool {
			return list[i].DoneAt.Before(list[j].DoneAt)
		})

	list = list.GetOlder(end)
	for len(list) > 0 {
		issues := list.GetInRange(start, end)
		list = list.GetOlder(start)
		weeks = append(weeks, WeeklyItems{
			Items: issues,
			Start: start,
			End:   end,
		})

		end = start
		start = getPreviousSunday(end)
	}

	return weeks
}

func getClosestSunday(now time.Time) time.Time {
	day := now.Weekday()
	tmp := now.AddDate(0, 0, -1*int(day))
	y := tmp.Year()
	m := tmp.Month()
	d := tmp.Day()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func getPreviousSunday(when time.Time) time.Time {
	return when.AddDate(0, 0, -7)
}
