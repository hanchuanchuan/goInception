package session

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// MySQL encodes a server version as major*10000 + minor*100 + patch.
	// Keep these values in one place so version checks remain consistent.
	mysqlVersion84     = 80400
	mariaDBVersion1052 = 100502
)

// ParseMySQLVersion parses a MySQL/MariaDB version string into the numeric
// representation used by the session compatibility checks.
//
// It accepts the server version form (for example, "9.7.2" or
// "8.4.11-commercial") and ignores the distribution suffix.
func ParseMySQLVersion(value string) (int, error) {
	version := strings.TrimSpace(value)
	if i := strings.IndexByte(version, '-'); i >= 0 {
		version = version[:i]
	}
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return 0, fmt.Errorf("无法解析版本号:%s", value)
	}

	components := make([]int, len(parts))
	for i, part := range parts {
		if part == "" {
			return 0, fmt.Errorf("无法解析版本号:%s", value)
		}
		component, err := strconv.Atoi(part)
		if err != nil || component < 0 {
			return 0, fmt.Errorf("无法解析版本号:%s", value)
		}
		components[i] = component
	}
	// The compact representation reserves two decimal digits for minor and
	// patch. Rejecting larger components avoids ambiguous comparisons.
	if components[1] > 99 || components[2] > 99 {
		return 0, fmt.Errorf("无法解析版本号:%s", value)
	}
	return components[0]*10000 + components[1]*100 + components[2], nil
}

// binlogStatusQuery returns the server-specific query that exposes the
// current binary log file and position.
func binlogStatusQuery(dbType, dbVersion int) string {
	switch dbType {
	case DBTypeMysql:
		// SHOW MASTER STATUS was removed in MySQL 8.4.0.
		if dbVersion >= mysqlVersion84 {
			return "SHOW BINARY LOG STATUS;"
		}
	case DBTypeMariaDB:
		// MariaDB 10.5.2 introduced SHOW BINLOG STATUS.
		if dbVersion >= mariaDBVersion1052 {
			return "SHOW BINLOG STATUS;"
		}
	}
	return "SHOW MASTER STATUS;"
}
