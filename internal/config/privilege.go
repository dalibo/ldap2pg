package config

import (
	"log/slog"

	"golang.org/x/exp/maps"
)

func (c Config) RegisterPrivileges() {
	for name, profile := range c.Privileges {
		profile.Register(name)
	}
}

func (c *Config) DropPrivileges() {
	slog.Debug("Dropping privilege configuration.")
	maps.Clear(c.Privileges)
	c.Rules = c.Rules.DropGrants()
}

func (c Config) ArePrivilegesManaged() bool {
	return 0 < len(c.Privileges)
}
