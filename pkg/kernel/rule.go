/*
Copyright (c) 2025 OpenInfra Foundation Europe

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kernel

import (
	"context"
	"fmt"

	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/utils"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

type RuleMatchFunc func(*netlink.Rule) bool

// SkipBuiltInRules returns true if the given netlink.Rule's table is not one of the
// standard built-in or unspec routing tables (local, main, default).
func SkipBuiltInRules(rule *netlink.Rule) bool {
	switch rule.Table {
	case unix.RT_TABLE_UNSPEC:
		fallthrough
	case unix.RT_TABLE_COMPAT:
		fallthrough
	case unix.RT_TABLE_LOCAL:
		fallthrough
	case unix.RT_TABLE_MAIN:
		fallthrough
	case unix.RT_TABLE_DEFAULT:
		return false
	default:
		return true
	}
}

// FlushRules lists and deletes IP rules for the specified family that match the
// provided RuleMatchFunc.
// If matchFn is nil, all rules for the given family will be attempted to be deleted.
// Errors encountered during rule deletion are accumulated and returned.
func FlushRules(ctx context.Context, family int, matchFn RuleMatchFunc) error {
	rules, err := netlink.RuleList(family)
	if err != nil {
		return fmt.Errorf("failed to fetch ip rules: %w", err)
	}

	logger := log.FromContextOrGlobal(ctx).WithName("FlushRules")
	for _, rule := range rules {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:

		}

		if matchFn == nil || matchFn(&rule) {
			if rule.Family == 0 {
				rule.Family = family // explicitly set family, so that IPv6 "from any" rules can be also removed
			}
			logger.V(1).Info("Delete IP rule", "family", rule.Family, "rule", rule)
			delErr := netlink.RuleDel(&rule)
			if delErr != nil {
				err = utils.AppendErr(err, fmt.Errorf("failed to delete ip rule %v, %w", rule, delErr))
			}

		}
	}
	return err
}
