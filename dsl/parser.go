package dsl

import (
	"github.com/spf13/viper"
)

type Workflow struct {
	Name   string       `mapstructure:"name"`
	Memory MemoryConfig `mapstructure:"memory"`
	Steps  []Step       `mapstructure:"steps"`
}

type MemoryConfig struct {
	Initial map[string]interface{} `mapstructure:"initial"`
}

type Step struct {
	ID       string                 `mapstructure:"id"`
	Type     string                 `mapstructure:"type"` // "task" or "parallel"
	Tool     string                 `mapstructure:"tool,omitempty"`
	Input    map[string]interface{} `mapstructure:"input,omitempty"`
	Output   string                 `mapstructure:"output,omitempty"`
	Messages map[string]string      `mapstructure:"messages,omitempty"` // New in v0.2
	Next     string                 `mapstructure:"next,omitempty"`     // New in v0.2
	Branches []Step                 `mapstructure:"branches,omitempty"` // For parallel type
}

func Parse(filename string) (*Workflow, error) {
	v := viper.New()
	v.SetConfigFile(filename)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var wf Workflow
	if err := v.UnmarshalKey("workflow", &wf); err != nil {
		return nil, err
	}

	return &wf, nil
}
