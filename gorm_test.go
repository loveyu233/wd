package wd

import (
	"strings"
	"testing"

	"gorm.io/gorm/logger"
)

func TestNormalizedGormDriver(t *testing.T) {
	tests := map[GormDriver]GormDriver{
		"":           GormDriverMySQL,
		"mysql":      GormDriverMySQL,
		"pg":         GormDriverPostgres,
		"pgsql":      GormDriverPostgres,
		"postgres":   GormDriverPostgres,
		"postgresql": GormDriverPostgres,
		"PostgreSQL": GormDriverPostgres,
	}

	for input, want := range tests {
		if got := normalizedGormDriver(input); got != want {
			t.Fatalf("normalizedGormDriver(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestBuildMySQLDSN(t *testing.T) {
	dsn, err := buildMySQLDSN(GormConnConfig{
		Username: "root",
		Password: "secret",
		Host:     "127.0.0.1",
		Port:     3306,
		Database: "demo db",
		Params: map[string]interface{}{
			"charset":   "utf8mb4",
			"parseTime": false,
			"loc":       "Asia/Shanghai",
		},
	})
	if err != nil {
		t.Fatalf("buildMySQLDSN returned error: %v", err)
	}

	want := "root:secret@tcp(127.0.0.1:3306)/demo%20db?charset=utf8mb4&loc=Asia%2FShanghai&parseTime=false"
	if dsn != want {
		t.Fatalf("buildMySQLDSN() = %q, want %q", dsn, want)
	}
}

func TestBuildPostgresDSN(t *testing.T) {
	dsn, err := buildPostgresDSN(GormConnConfig{
		Username: "postgres",
		Password: "p@ss word",
		Host:     "127.0.0.1",
		Port:     5432,
		Database: "demo",
		Params: map[string]interface{}{
			"sslmode":          "require",
			"TimeZone":         "Asia/Tokyo",
			"application_name": "wd test",
		},
	})
	if err != nil {
		t.Fatalf("buildPostgresDSN returned error: %v", err)
	}

	wantParts := []string{
		"host=127.0.0.1",
		"port=5432",
		"user=postgres",
		"password='p@ss word'",
		"dbname=demo",
		"TimeZone=Asia/Tokyo",
		"application_name='wd test'",
		"sslmode=require",
	}
	want := strings.Join(wantParts, " ")
	if dsn != want {
		t.Fatalf("buildPostgresDSN() = %q, want %q", dsn, want)
	}
}

func TestPostgresDSNValue(t *testing.T) {
	if got := postgresDSNValue(`pa'ss\word`); got != `'pa\'ss\\word'` {
		t.Fatalf("postgresDSNValue() = %q", got)
	}
	if got := postgresDSNValue("Asia/Shanghai"); got != "Asia/Shanghai" {
		t.Fatalf("postgresDSNValue() = %q", got)
	}
}

func TestGormLogLevel(t *testing.T) {
	tests := map[GormLogLevel]logger.LogLevel{
		"":                     logger.Info,
		GormLogDebug:           logger.Info,
		GormLogInfo:            logger.Info,
		GormLogWarn:            logger.Warn,
		GormLogWarning:         logger.Warn,
		GormLogErr:             logger.Error,
		GormLogError:           logger.Error,
		GormLogSilent:          logger.Silent,
		GormLogLevel("waring"): logger.Warn,
	}

	for input, want := range tests {
		got, err := gormLogLevel(input)
		if err != nil {
			t.Fatalf("gormLogLevel(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("gormLogLevel(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestBuildGormDialectorRejectsUnsupportedDriver(t *testing.T) {
	_, err := buildGormDialector(GormConnConfig{Driver: "sqlite"})
	if err == nil {
		t.Fatal("buildGormDialector should reject unsupported driver")
	}
	if !strings.Contains(err.Error(), "不支持的 GORM 数据库驱动") {
		t.Fatalf("unexpected error: %v", err)
	}
}
