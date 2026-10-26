package session

import "testing"

func TestParseMySQLVersion(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int
		wantErr bool
	}{
		{name: "mysql 8.4", value: "8.4.11", want: 80411},
		{name: "mysql 9.7 suffix", value: "9.7.2-commercial", want: 90702},
		{name: "mariadb", value: "10.11.13-MariaDB", want: 101113},
		{name: "invalid", value: "8.4", wantErr: true},
		{name: "invalid component", value: "8.x.1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMySQLVersion(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseMySQLVersion(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Fatalf("ParseMySQLVersion(%q) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestBinlogStatusQuery(t *testing.T) {
	tests := []struct {
		name     string
		dbType   int
		version  int
		expected string
	}{
		{name: "mysql 8.0", dbType: DBTypeMysql, version: 80036, expected: "SHOW MASTER STATUS;"},
		{name: "mysql 8.4", dbType: DBTypeMysql, version: 80400, expected: "SHOW BINARY LOG STATUS;"},
		{name: "mysql 9.7", dbType: DBTypeMysql, version: 90702, expected: "SHOW BINARY LOG STATUS;"},
		{name: "mariadb 10.4", dbType: DBTypeMariaDB, version: 100401, expected: "SHOW MASTER STATUS;"},
		{name: "mariadb 10.5", dbType: DBTypeMariaDB, version: 100502, expected: "SHOW BINLOG STATUS;"},
		{name: "tidb", dbType: DBTypeTiDB, version: 80000, expected: "SHOW MASTER STATUS;"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := binlogStatusQuery(tt.dbType, tt.version); got != tt.expected {
				t.Fatalf("binlogStatusQuery(%d, %d) = %q, want %q", tt.dbType, tt.version, got, tt.expected)
			}
		})
	}
}
