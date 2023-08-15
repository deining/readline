package completion

import (
	"strings"
)

// RawValues is a list of completion candidates.
type RawValues []Candidate

// Filter filters values.
func (c RawValues) Filter(values ...string) RawValues {
	toremove := make(map[string]bool)
	for _, v := range values {
		toremove[v] = true
	}

	filtered := make([]Candidate, 0)

	for _, rawValue := range c {
		if _, ok := toremove[rawValue.Value]; !ok {
			filtered = append(filtered, rawValue)
		}
	}

	return filtered
}

// Merge merges a set of values with the current ones,
// include usage/message strings, meta settings, etc.
func (c *Values) Merge(other Values) {
	if other.Usage != "" {
		c.Usage = other.Usage
	}

	c.NoSpace.Merge(other.NoSpace)
	c.Messages.Merge(other.Messages)

	for tag := range other.ListLong {
		if _, found := c.ListLong[tag]; !found {
			c.ListLong[tag] = true
		}
	}
}

// EachTag iterates over each tag and runs a function for each group.
func (c RawValues) EachTag(tagF func(tag string, values RawValues)) {
	tags := make([]string, 0)
	tagGroups := make(map[string]RawValues)

	for _, val := range c {
		if _, exists := tagGroups[val.Tag]; !exists {
			tagGroups[val.Tag] = make(RawValues, 0)

			tags = append(tags, val.Tag)
		}

		tagGroups[val.Tag] = append(tagGroups[val.Tag], val)
	}

	for _, tag := range tags {
		tagF(tag, tagGroups[tag])
	}
}

// FilterPrefix filters values with given prefix.
// If matchCase is false, the filtering is made case-insensitive.
// This function ensures that all spaces are correctly.
func (c RawValues) FilterPrefix(prefix string, matchCase bool) RawValues {
	if prefix == "" {
		return c
	}

	filtered := make(RawValues, 0)

	if !matchCase {
		prefix = strings.ToLower(prefix)
	}

	for _, raw := range c {
		val := raw.Value

		if !matchCase {
			val = strings.ToLower(val)
		}

		if strings.HasPrefix(val, prefix) {
			filtered = append(filtered, raw)
		}
	}

	return filtered
}

func (x RawValues) Len() int { return len(x) }

func (x RawValues) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

func (x RawValues) Less(i, j int) bool {
	return strings.ToLower(x[i].Value) < strings.ToLower(x[j].Display)
}
