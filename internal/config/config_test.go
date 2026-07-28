package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestGitHubIssueNotificationsDefaultDisabled(t *testing.T) {
	v := viper.New()
	setDefaults(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal defaults: %v", err)
	}
	if cfg.GitHub.IssueNotificationsEnabled {
		t.Fatal("GitHub issue notifications must be disabled by default")
	}
}
