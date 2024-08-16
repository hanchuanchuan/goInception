// Copyright 2013 The ql Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSES/QL-LICENSE file.

package session

import (
	"fmt"
	"github.com/hanchuanchuan/goInception/ast"
	"strings"

	"github.com/hanchuanchuan/goInception/model"
)

func (s *session) checkPartitionTruncate(t *TableInfo, parts []model.CIStr) {
	for _, part := range parts {
		found := false
		for _, oldPart := range t.Partitions {
			if strings.EqualFold(part.String(), oldPart.PartName) {
				found = true
				break
			}
		}
		if found && s.dbType == DBTypeOceanBase && s.inc.CheckOfflineDDL {
			s.appendErrorNo(ER_CANT_TRUNCATE_PARTITION, part.String())
			break
		}

		if !found {
			s.appendErrorNo(ErrPartitionNotExisted, part.String())
		}
	}
}

func (s *session) checkAlterPartitionRule(t *TableInfo, opts *ast.PartitionOptions) {
	if opts == nil {
		return
	}

	if s.dbType == DBTypeOceanBase && s.inc.CheckOfflineDDL {
		s.appendErrorMsg(fmt.Sprintf("Can't alter partition rule of table '%s'.", t.Name))
	}
}

func (s *session) checkCharsetChange(charset string, table string) bool {
	if s.inc.CheckOfflineDDL {
		s.appendErrorMsg(fmt.Sprintf("Can't change chartset of table '%s' to '%s'.", table, charset))
		return true
	}
	return false
}

func (s *session) checkAddPrimaryKey(t *TableInfo, c *ast.AlterTableSpec) {
	if s.inc.CheckOfflineDDL && s.dbType == DBTypeOceanBase {
		var primaryNames []string
		for _, key := range c.Constraint.Keys {
			primaryNames = append(primaryNames, key.Column.Name.O)
		}
		s.appendErrorNo(ER_CANT_ADD_PRIMARY_KEY,
			fmt.Sprintf("%s.%s", t.Name, primaryNames))
	}
}
