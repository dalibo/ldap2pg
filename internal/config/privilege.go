package config

import (
	"fmt"
	"log/slog"

	"golang.org/x/exp/maps"
)

func (c Config) RegisterPrivileges() error {
	for name, acl := range c.ACLs {
		acl.Name = name
		slog.Debug("Registering ACL.", "name", acl.Name)
		err := acl.Register()
		if err != nil {
			return fmt.Errorf("ACL: %s: %w", acl.Name, err)
		}
	}
	for name, profile := range c.Privileges {
		err := profile.Register(name)
		if err != nil {
			return fmt.Errorf("privileges: %s: %w", name, err)
		}
	}
	return nil
}

func (c *Config) DropPrivileges() {
	slog.Debug("Dropping privilege configuration.")
	maps.Clear(c.Privileges)
	c.Rules = c.Rules.DropGrants()
}

func (c Config) ArePrivilegesManaged() bool {
	return 0 < len(c.Privileges)
}
