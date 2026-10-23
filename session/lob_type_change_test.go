package session

import "testing"

// TestLobTypeChangeAllowed 覆盖 text/blob 家族的类型变更方向判断。
// 该函数不依赖数据库, 因此在没有测试 MySQL 的环境里也能直接跑。
func TestLobTypeChangeAllowed(t *testing.T) {
	cases := []struct {
		oldType, newType string
		want             bool
	}{
		// 同家族扩容: 放行
		{"text", "mediumtext", true},
		{"text", "longtext", true},
		{"tinytext", "longtext", true},
		{"mediumtext", "mediumtext", true},
		{"blob", "longblob", true},
		{"tinyblob", "blob", true},
		{"mediumblob", "longblob", true},

		// 同家族缩容: 告警
		{"mediumtext", "text", false},
		{"longtext", "mediumtext", false},
		{"longblob", "blob", false},
		{"blob", "tinyblob", false},

		// 跨家族: text 带字符集, blob 是二进制, 不等价
		{"text", "blob", false},
		{"blob", "text", false},
		{"mediumtext", "longblob", false},

		// 与非 text/blob 类型互转: 交给原有规则
		{"text", "json", false},
		{"json", "longtext", false},
		{"varchar(100)", "varchar(200)", false},
		{"int(11)", "bigint(20)", false},
	}

	for _, c := range cases {
		if got := lobTypeChangeAllowed(c.oldType, c.newType); got != c.want {
			t.Errorf("lobTypeChangeAllowed(%q, %q) = %v, want %v", c.oldType, c.newType, got, c.want)
		}
	}
}
