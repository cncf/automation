package main

// Label represents a label definition
type Label struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Description string `yaml:"description"`
	Previously  []struct {
		Name string `yaml:"name"`
	} `yaml:"previously"`
}

// ActionSpec represents action specifications
type ActionSpec struct {
	Match string `yaml:"match,omitempty"`
	Label string `yaml:"label,omitempty"`
}

// Action represents an action to take
type Action struct {
	Kind string     `yaml:"kind"`
	Spec ActionSpec `yaml:"spec,omitempty"`
}

// RuleSpec represents rule specifications
type RuleSpec struct {
	Command        string        `yaml:"command,omitempty"`
	Rules          []interface{} `yaml:"rules,omitempty"`
	Match          string        `yaml:"match,omitempty"`
	MatchCondition string        `yaml:"matchCondition,omitempty"`
	MatchPath      string        `yaml:"matchPath,omitempty"`
	MatchList      []string      `yaml:"matchList,omitempty"`
}

// Rule represents a labeling rule
type Rule struct {
	Name    string   `yaml:"name"`
	Kind    string   `yaml:"kind"`
	Spec    RuleSpec `yaml:"spec"`
	Actions []Action `yaml:"actions"`
}

// LabelsYAML represents the complete configuration
type LabelsYAML struct {
	Labels             []Label `yaml:"labels"`
	Ruleset            []Rule  `yaml:"ruleset"`
	AutoCreate         bool    `yaml:"autoCreateLabels"`
	AutoDelete         bool    `yaml:"autoDeleteLabels"`
	DefinitionRequired bool    `yaml:"definitionRequired"`
	Debug              bool    `yaml:"debug"`
}
