package config

import (
	"fmt"
	"log/slog"
	"strings"

	"golang.org/x/exp/maps"
)

func (c Config) RegisterPrivileges() error {
	for name, acl := range c.ACLs {
		// We have built-in handling for DEFAULT ACLs.
		// Don't trigger default code for custom ACL.
		if strings.HasSuffix(name, " DEFAULT") {
			return fmt.Errorf("ACL: %s: reserved name", name)
		}

		acl.Name = name
		slog.Debug("Registering ACL.", "name", acl.Name)
		err := acl.Register()
		if err != nil {
			return fmt.Errorf("ACL: %s: %w", acl.Name, err)
		}
	}
	for name, profile := range c.Privileges {
		profile.Register(name)
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
