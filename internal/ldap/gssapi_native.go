package ldap

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	ldap3 "github.com/go-ldap/ldap/v3"
)

// decodeLDIFValue decodes the value part after "attr:" in LDIF format.
// If rest starts with ":" it is base64-encoded (attr:: value), otherwise plain text.
func decodeLDIFValue(rest string) (string, error) {
	if strings.HasPrefix(rest, ":") {
		b64 := strings.TrimSpace(strings.TrimPrefix(rest, ":"))
		decoded, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	}
	return strings.TrimSpace(rest), nil
}

// NativeGSSAPISearch executes ldapsearch command with GSSAPI authentication
// and parses the LDIF output into SearchResult
func (c *Client) NativeGSSAPISearch(base string, scope Scope, filter string, attributes []string) (*ldap3.SearchResult, error) {
	args := []string{
		"-Y", "GSSAPI",
		"-H", c.URI,
		"-b", base,
		"-s", scope.String(),
		"-LLL", // LDIF format without comments
		filter,
	}
	args = append(args, attributes...)

	slog.Debug("Executing native ldapsearch.", "base", base, "scope", scope.String(), "filter", filter)

	cmd := exec.Command("ldapsearch", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		slog.Debug("Native ldapsearch failed.", "err", err, "stderr", stderr.String())
		return nil, fmt.Errorf("ldapsearch failed: %w: %s", err, stderr.String())
	}

	// Parse LDIF output
	result, err := parseLDIF(stdout.String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse LDIF: %w", err)
	}

	slog.Debug("Native ldapsearch done.", "entries", len(result.Entries))
	return result, nil
}

// parseLDIF converts LDIF output to SearchResult
func parseLDIF(ldif string) (*ldap3.SearchResult, error) {
	result := &ldap3.SearchResult{
		Entries: []*ldap3.Entry{},
	}

	lines := strings.Split(ldif, "\n")
	var currentEntry *ldap3.Entry
	var currentAttr string
	var currentValues []string

	for _, line := range lines {
		// Skip empty lines and comments
		if strings.TrimSpace(line) == "" {
			// Empty line marks end of entry
			if currentEntry != nil {
				if currentAttr != "" && len(currentValues) > 0 {
					currentEntry.Attributes = append(currentEntry.Attributes, &ldap3.EntryAttribute{
						Name:   currentAttr,
						Values: currentValues,
					})
				}
				result.Entries = append(result.Entries, currentEntry)
				currentEntry = nil
				currentAttr = ""
				currentValues = nil
			}
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}

		// DN line starts a new entry
		if strings.HasPrefix(line, "dn:") {
			// Save previous entry if exists
			if currentEntry != nil {
				if currentAttr != "" && len(currentValues) > 0 {
					currentEntry.Attributes = append(currentEntry.Attributes, &ldap3.EntryAttribute{
						Name:   currentAttr,
						Values: currentValues,
					})
				}
				result.Entries = append(result.Entries, currentEntry)
			}

			dn, err := decodeLDIFValue(strings.TrimPrefix(line, "dn:"))
			if err != nil {
				slog.Debug("Failed to decode base64 DN.", "err", err)
				continue
			}
			currentEntry = &ldap3.Entry{
				DN:         dn,
				Attributes: []*ldap3.EntryAttribute{},
			}
			currentAttr = ""
			currentValues = nil
			continue
		}

		// Continuation line (starts with space)
		if strings.HasPrefix(line, " ") && currentAttr != "" {
			if len(currentValues) > 0 {
				currentValues[len(currentValues)-1] += strings.TrimPrefix(line, " ")
			}
			continue
		}

		// Attribute line
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			attrName := strings.TrimSpace(parts[0])
			attrValue, err := decodeLDIFValue(parts[1])
			if err != nil {
				slog.Debug("Failed to decode base64 attribute.", "attr", attrName, "err", err)
				continue
			}

			if attrName != currentAttr {
				if currentAttr != "" && len(currentValues) > 0 {
					currentEntry.Attributes = append(currentEntry.Attributes, &ldap3.EntryAttribute{
						Name:   currentAttr,
						Values: currentValues,
					})
				}
				currentAttr = attrName
				currentValues = []string{}
			}

			currentValues = append(currentValues, attrValue)
		}
	}

	// Don't forget the last entry
	if currentEntry != nil {
		if currentAttr != "" && len(currentValues) > 0 {
			currentEntry.Attributes = append(currentEntry.Attributes, &ldap3.EntryAttribute{
				Name:   currentAttr,
				Values: currentValues,
			})
		}
		result.Entries = append(result.Entries, currentEntry)
	}

	return result, nil
}
