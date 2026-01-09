package wd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	InsDB *GormClient
)

type GormClient struct {
	*gorm.DB
}

type GormConnConfig struct {
	Username string
	Password string
	Host     string
	Port     int
	Database string
	Params   map[string]interface{} // 连接参数,默认添加charset=utf8和parseTime=true以及loc=Asia%2FShanghai
}

// InitGormDB 用来根据配置初始化全局 GORM 连接。
func InitGormDB(gcc GormConnConfig, gormLogger logger.Interface, opt ...func(db *gorm.DB) error) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?", gcc.Username, gcc.Password, gcc.Host, gcc.Port, gcc.Database)
	if gcc.Params["charset"] == nil {
		dsn = fmt.Sprintf("%scharset=utf8", dsn)
	}
	if gcc.Params["parseTime"] == nil {
		dsn = fmt.Sprintf("%s&parseTime=true", dsn)
	}
	if gcc.Params["loc"] == nil {
		dsn = fmt.Sprintf("%s&loc=Asia%%2FShanghai", dsn)
	}
	for k, v := range gcc.Params {
		dsn = fmt.Sprintf("%s&%s=%v", dsn, k, v)
	}
	db, err := gorm.Open(
		mysql.Open(dsn),
		&gorm.Config{
			Logger:                 gormLogger,
			TranslateError:         true,
			SkipDefaultTransaction: true,
			PrepareStmt:            true,
		},
	)
	if err != nil {
		return err
	}

	for _, fn := range opt {
		if err := fn(db); err != nil {
			return err
		}
	}

	InsDB = new(GormClient)
	InsDB.DB = db

	return nil
}

// GormDefaultLogger 用来生成带默认阈值的 GORM 日志器。
type gormLoggerSettings struct {
	cfg    logger.Config
	writer io.Writer
	prefix string
	flag   int
}

// GormLoggerOption 用来修改默认 GORM logger 配置。
type GormLoggerOption func(*gormLoggerSettings)

// WithGormConfigLogLevel 设置日志级别（logger.Silent/Error/Warn/Info）。

func WithGormConfigLogLevel(level logger.LogLevel) GormLoggerOption {
	return func(settings *gormLoggerSettings) {
		settings.cfg.LogLevel = level
	}
}

// WithGormConfigSlowThreshold 设置慢查询阈值。
func WithGormConfigSlowThreshold(d time.Duration) GormLoggerOption {
	return func(settings *gormLoggerSettings) {
		settings.cfg.SlowThreshold = d
	}
}

// WithGormConfigIgnoreRecordNotFound 设置是否忽略 record not found 错误。
func WithGormConfigIgnoreRecordNotFound(ignore bool) GormLoggerOption {
	return func(settings *gormLoggerSettings) {
		settings.cfg.IgnoreRecordNotFoundError = ignore
	}
}

// WithGormConfigColorful 控制是否输出带颜色的日志。
func WithGormConfigColorful(colorful bool) GormLoggerOption {
	return func(settings *gormLoggerSettings) {
		settings.cfg.Colorful = colorful
	}
}

// WithGormConfigWriter 设置日志输出的 writer。
func WithGormConfigWriter(w io.Writer) GormLoggerOption {
	return func(settings *gormLoggerSettings) {
		if w != nil {
			settings.writer = w
		}
	}
}

// WithGormConfigDisableConsole 禁止输出到控制台，仅保留链路日志。
func WithGormConfigDisableConsole() GormLoggerOption {
	return func(settings *gormLoggerSettings) {
		settings.writer = io.Discard
	}
}

// WithGormConfigLogPrefix 设置底层 log.Logger 的前缀。
func WithGormConfigLogPrefix(prefix string) GormLoggerOption {
	return func(settings *gormLoggerSettings) {
		settings.prefix = prefix
	}
}

// WithGormConfigLogFlag 设置底层 log.Logger 的 flag。
func WithGormConfigLogFlag(flag int) GormLoggerOption {
	return func(settings *gormLoggerSettings) {
		settings.flag = flag
	}
}

// GormDefaultLogger 用来生成带默认阈值的 GORM 日志器。

func GormDefaultLogger(opts ...GormLoggerOption) logger.Interface {
	settings := &gormLoggerSettings{
		cfg: logger.Config{
			LogLevel: logger.Info,
		},
		writer: os.Stdout,
		prefix: "",
		flag:   log.LstdFlags,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(settings)
		}
	}
	base := logger.New(
		log.New(settings.writer, settings.prefix, settings.flag),
		settings.cfg,
	)
	return WrapGormLoggerWithRequestLogger(base)
}

// WrapGormLoggerWithRequestLogger 会让gorm日志透传到请求链路日志中。
func WrapGormLoggerWithRequestLogger(base logger.Interface) logger.Interface {
	return &requestAwareGormLogger{base: base}
}

type requestAwareGormLogger struct {
	base logger.Interface
}

func (l *requestAwareGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	if l.base == nil {
		return &requestAwareGormLogger{}
	}
	return &requestAwareGormLogger{base: l.base.LogMode(level)}
}

func (l *requestAwareGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.base != nil {
		l.base.Info(ctx, msg, data...)
	}
}

func (l *requestAwareGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.base != nil {
		l.base.Warn(ctx, msg, data...)
	}
}

func (l *requestAwareGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.base != nil {
		l.base.Error(ctx, msg, data...)
	}
}

func (l *requestAwareGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	capture := &traceSQL{}
	wrappedFC := func() (string, int64) {
		return capture.get(fc)
	}
	if l.base != nil {
		l.base.Trace(ctx, begin, wrappedFC, err)
	} else {
		wrappedFC()
	}
	if rl := RequestLoggerFromContext(ctx); rl != nil {
		sql, rows := capture.get(fc)
		level := zerolog.InfoLevel
		fields := map[string]any{
			"sql":         sql,
			"rows":        rows,
			"duration_ms": time.Since(begin).Milliseconds(),
			"level":       level.String(),
		}
		if err != nil {
			level = zerolog.ErrorLevel
			fields["error"] = err.Error()
			fields["level"] = level.String()
		}
		fields["timestamp"] = Now().Format(CSTLayout)
		rl.AddSQLEntry(fields)
	}
}

type traceSQL struct {
	once sync.Once
	sql  string
	rows int64
}

func (t *traceSQL) get(fn func() (string, int64)) (string, int64) {
	t.once.Do(func() {
		if fn != nil {
			t.sql, t.rows = fn()
		}
	})
	return t.sql, t.rows
}
