package wd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	InsDB                      *GormClient
	requestAwareGormLoggerFile = currentSourceFile()
	moduleRootCache            sync.Map
	requestAwareSkipFuncPrefix = []string{
		"gorm.io/gorm",
		"gorm.io/gen",
	}
	requestAwareSkipFileSuffix = []string{
		".gen.go",
	}
)

const (
	defaultMySQLCharset = "utf8"
)

type GormClient struct {
	*gorm.DB
}

type GormConnConfig struct {
	Username    string
	Password    string
	Host        string
	Port        int
	Database    string
	Params      map[string]interface{} // 连接参数,默认添加charset=utf8和parseTime=true以及loc=Asia/Shanghai
	PrepareStmt bool                   // 是否启用 PrepareStmt，默认 false
}

// InitGormDB 用来根据配置初始化全局 GORM 连接。
func InitGormDB(gcc GormConnConfig, gormLogger logger.Interface, opt ...func(db *gorm.DB) error) error {
	dsn, err := buildMySQLDSN(gcc)
	if err != nil {
		return err
	}

	db, err := gorm.Open(
		gormmysql.Open(dsn),
		&gorm.Config{
			Logger:                 gormLogger,
			TranslateError:         true,
			SkipDefaultTransaction: true,
			PrepareStmt:            gcc.PrepareStmt,
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

func buildMySQLDSN(gcc GormConnConfig) (string, error) {
	params := map[string]string{
		"charset":   defaultMySQLCharset,
		"parseTime": "true",
		"loc":       ShangHaiTimeLocation.String(),
	}

	for key, value := range gcc.Params {
		if value == nil {
			continue
		}

		rawValue := strings.TrimSpace(fmt.Sprint(value))
		if rawValue == "" {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(key)) {
		case "charset":
			params["charset"] = rawValue
		case "parsetime":
			parseTime, err := parseMySQLBoolParam(rawValue)
			if err != nil {
				return "", fmt.Errorf("mysql 参数 parseTime 无效: %w", err)
			}
			params["parseTime"] = strconv.FormatBool(parseTime)
		case "loc":
			loc, err := parseMySQLLocation(rawValue)
			if err != nil {
				return "", err
			}
			params["loc"] = loc.String()
		default:
			params[key] = rawValue
		}
	}

	paramPairs := make([]string, 0, len(params))
	for _, key := range sortedStringKeys(params) {
		paramPairs = append(paramPairs, fmt.Sprintf("%s=%s", key, url.QueryEscape(params[key])))
	}

	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?%s",
		gcc.Username,
		gcc.Password,
		gcc.Host,
		gcc.Port,
		url.PathEscape(gcc.Database),
		strings.Join(paramPairs, "&"),
	), nil
}

func parseMySQLBoolParam(value any) (bool, error) {
	switch typedValue := value.(type) {
	case bool:
		return typedValue, nil
	case string:
		return strconv.ParseBool(strings.TrimSpace(typedValue))
	default:
		raw := strings.TrimSpace(fmt.Sprint(value))
		switch strings.ToLower(raw) {
		case "1", "true", "yes", "on":
			return true, nil
		case "0", "false", "no", "off":
			return false, nil
		default:
			return false, fmt.Errorf("unsupported bool value %q", raw)
		}
	}
}

func parseMySQLLocation(value any) (*time.Location, error) {
	switch typedValue := value.(type) {
	case *time.Location:
		if typedValue == nil {
			return ShangHaiTimeLocation, nil
		}
		return typedValue, nil
	case string:
		locationName := strings.TrimSpace(typedValue)
		if locationName == "" {
			return ShangHaiTimeLocation, nil
		}
		loc, err := time.LoadLocation(locationName)
		if err != nil {
			return nil, fmt.Errorf("mysql 参数 loc 无效: %w", err)
		}
		return loc, nil
	default:
		locationName := strings.TrimSpace(fmt.Sprint(value))
		if locationName == "" {
			return ShangHaiTimeLocation, nil
		}
		loc, err := time.LoadLocation(locationName)
		if err != nil {
			return nil, fmt.Errorf("mysql 参数 loc 无效: %w", err)
		}
		return loc, nil
	}
}

func sortedStringKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// GormDefaultLogger 用来生成带默认阈值的 GORM 日志器。
type gormLoggerSettings struct {
	cfg            logger.Config
	writer         io.Writer
	prefix         string
	flag           int
	callerPathMode GormCallerPathMode
}

// GormLoggerOption 用来修改默认 GORM logger 配置。
type GormLoggerOption func(*gormLoggerSettings)

// GormCallerPathMode 用来控制 SQL 日志中的文件路径展示方式。
type GormCallerPathMode string

const (
	// GormCallerPathModeAbsolute 会输出绝对路径，适合在编辑器控制台中直接点击跳转。
	GormCallerPathModeAbsolute GormCallerPathMode = "absolute"
	// GormCallerPathModeModuleRelative 会输出相对 go.mod 根目录的路径，日志更干净。
	GormCallerPathModeModuleRelative GormCallerPathMode = "module-relative"
)

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

// WithGormConfigCallerPathMode 设置 SQL 日志中的调用文件路径输出模式。
func WithGormConfigCallerPathMode(mode GormCallerPathMode) GormLoggerOption {
	return func(settings *gormLoggerSettings) {
		switch mode {
		case GormCallerPathModeModuleRelative:
			settings.callerPathMode = GormCallerPathModeModuleRelative
		default:
			settings.callerPathMode = GormCallerPathModeAbsolute
		}
	}
}

// GormDefaultLogger 用来生成带默认阈值的 GORM 日志器。
func GormDefaultLogger(opts ...GormLoggerOption) logger.Interface {
	settings := &gormLoggerSettings{
		cfg: logger.Config{
			LogLevel: logger.Info,
		},
		writer:         os.Stdout,
		prefix:         "",
		flag:           log.LstdFlags,
		callerPathMode: GormCallerPathModeAbsolute,
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
	return wrapGormLoggerWithRequestLogger(base, settings.callerPathMode)
}

type requestAwareGormLogger struct {
	base                                logger.Interface
	writer                              logger.Writer
	cfg                                 logger.Config
	callerPathMode                      GormCallerPathMode
	delegateTraceToBase                 bool
	infoStr, warnStr, errStr            string
	traceStr, traceErrStr, traceWarnStr string
}

func (l *requestAwareGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	if l.delegateTraceToBase {
		if l.base == nil {
			return &requestAwareGormLogger{delegateTraceToBase: true}
		}
		return &requestAwareGormLogger{base: l.base.LogMode(level), delegateTraceToBase: true}
	}

	settings := l.cfg
	settings.LogLevel = level
	base := l.base
	if base != nil {
		base = base.LogMode(level)
	}
	return newRequestAwareGormLogger(base, l.writer, settings, l.callerPathMode)
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
	if l.delegateTraceToBase && l.base != nil {
		l.base.Trace(ctx, begin, wrappedFC, err)
	} else if l.delegateTraceToBase {
		wrappedFC()
	} else {
		l.trace(begin, wrappedFC, err)
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
		}
		fields["timestamp"] = Now().Format(CSTLayout)
		rl.AddSQLEntry(level, fields)
	}
}

// WrapGormLoggerWithRequestLogger 会在保留 SQL 输出的同时，把 SQL 明细写入请求链路日志。
// 当传入的是标准 GORM logger 时，这里会改为使用自定义 Trace，避免调用方文件被包装层截断成当前 gorm.go。
func WrapGormLoggerWithRequestLogger(base logger.Interface) logger.Interface {
	return wrapGormLoggerWithRequestLogger(base, GormCallerPathModeAbsolute)
}

// wrapGormLoggerWithRequestLogger 会根据路径模式包装 GORM logger。
func wrapGormLoggerWithRequestLogger(base logger.Interface, callerPathMode GormCallerPathMode) logger.Interface {
	if base == nil {
		return &requestAwareGormLogger{delegateTraceToBase: true}
	}
	writer, cfg, ok := extractGormLoggerOutput(base)
	if !ok {
		return &requestAwareGormLogger{base: base, delegateTraceToBase: true}
	}
	return newRequestAwareGormLogger(base, writer, cfg, callerPathMode)
}

// newRequestAwareGormLogger 构造一个兼容 GORM 默认输出格式的 SQL logger。
func newRequestAwareGormLogger(base logger.Interface, writer logger.Writer, cfg logger.Config, callerPathMode GormCallerPathMode) *requestAwareGormLogger {
	infoStr, warnStr, errStr, traceStr, traceWarnStr, traceErrStr := buildGormLoggerFormats(cfg)
	return &requestAwareGormLogger{
		base:                base,
		writer:              writer,
		cfg:                 cfg,
		callerPathMode:      normalizeGormCallerPathMode(callerPathMode),
		infoStr:             infoStr,
		warnStr:             warnStr,
		errStr:              errStr,
		traceStr:            traceStr,
		traceWarnStr:        traceWarnStr,
		traceErrStr:         traceErrStr,
		delegateTraceToBase: false,
	}
}

// extractGormLoggerOutput 从标准 GORM logger 中提取 writer 与配置，便于保持原有输出格式。
func extractGormLoggerOutput(base logger.Interface) (logger.Writer, logger.Config, bool) {
	v := reflect.ValueOf(base)
	for v.IsValid() && (v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer) {
		if v.IsNil() {
			return nil, logger.Config{}, false
		}
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return nil, logger.Config{}, false
	}

	writerField := v.FieldByName("Writer")
	configField := v.FieldByName("Config")
	if !writerField.IsValid() || !configField.IsValid() || !writerField.CanInterface() || !configField.CanInterface() {
		return nil, logger.Config{}, false
	}

	writer, ok := writerField.Interface().(logger.Writer)
	if !ok {
		return nil, logger.Config{}, false
	}
	cfg, ok := configField.Interface().(logger.Config)
	if !ok {
		return nil, logger.Config{}, false
	}
	return writer, cfg, true
}

// buildGormLoggerFormats 复用 GORM 默认日志格式，避免包装后控制台输出风格发生变化。
func buildGormLoggerFormats(cfg logger.Config) (string, string, string, string, string, string) {
	infoStr := "%s\n[info] "
	warnStr := "%s\n[warn] "
	errStr := "%s\n[error] "
	traceStr := "%s\n[%.3fms] [rows:%v] %s"
	traceWarnStr := "%s %s\n[%.3fms] [rows:%v] %s"
	traceErrStr := "%s %s\n[%.3fms] [rows:%v] %s"

	if cfg.Colorful {
		infoStr = logger.Green + "%s\n" + logger.Reset + logger.Green + "[info] " + logger.Reset
		warnStr = logger.BlueBold + "%s\n" + logger.Reset + logger.Magenta + "[warn] " + logger.Reset
		errStr = logger.Magenta + "%s\n" + logger.Reset + logger.Red + "[error] " + logger.Reset
		traceStr = logger.Green + "%s\n" + logger.Reset + logger.Yellow + "[%.3fms] " + logger.BlueBold + "[rows:%v]" + logger.Reset + " %s"
		traceWarnStr = logger.Green + "%s " + logger.Yellow + "%s\n" + logger.Reset + logger.RedBold + "[%.3fms] " + logger.Yellow + "[rows:%v]" + logger.Magenta + " %s" + logger.Reset
		traceErrStr = logger.RedBold + "%s " + logger.MagentaBold + "%s\n" + logger.Reset + logger.Yellow + "[%.3fms] " + logger.BlueBold + "[rows:%v]" + logger.Reset + " %s"
	}

	return infoStr, warnStr, errStr, traceStr, traceWarnStr, traceErrStr
}

// trace 复制了 GORM 默认 logger 的 Trace 判定逻辑，只替换了调用方文件的解析方式。
func (l *requestAwareGormLogger) trace(begin time.Time, fc func() (string, int64), err error) {
	if l.cfg.LogLevel <= logger.Silent || l.writer == nil {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.cfg.LogLevel >= logger.Error && (!errors.Is(err, logger.ErrRecordNotFound) || !l.cfg.IgnoreRecordNotFoundError):
		sql, rows := fc()
		caller := requestAwareCallerFileWithLineNum(l.callerPathMode)
		if rows == -1 {
			l.writer.Printf(l.traceErrStr, caller, err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.writer.Printf(l.traceErrStr, caller, err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > l.cfg.SlowThreshold && l.cfg.SlowThreshold != 0 && l.cfg.LogLevel >= logger.Warn:
		sql, rows := fc()
		caller := requestAwareCallerFileWithLineNum(l.callerPathMode)
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.cfg.SlowThreshold)
		if rows == -1 {
			l.writer.Printf(l.traceWarnStr, caller, slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.writer.Printf(l.traceWarnStr, caller, slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case l.cfg.LogLevel == logger.Info:
		sql, rows := fc()
		caller := requestAwareCallerFileWithLineNum(l.callerPathMode)
		if rows == -1 {
			l.writer.Printf(l.traceStr, caller, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.writer.Printf(l.traceStr, caller, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}

// requestAwareCallerFileWithLineNum 会像 GORM 一样向上查找调用栈，但会额外跳过当前包装文件。
func requestAwareCallerFileWithLineNum(mode GormCallerPathMode) string {
	var pcs [16]uintptr
	depth := runtime.Callers(3, pcs[:])
	frames := runtime.CallersFrames(pcs[:depth])
	for {
		frame, more := frames.Next()
		if frame.File == "" {
			if !more {
				break
			}
			continue
		}

		if !shouldSkipCallerFrame(frame) {
			return callerPathWithLineNum(frame.File, frame.Line, mode)
		}

		if !more {
			break
		}
	}
	return ""
}

// shouldSkipCallerFrame 统一维护需要跳过的 ORM 与包装层栈帧，避免把中间层误识别成业务调用方。
func shouldSkipCallerFrame(frame runtime.Frame) bool {
	normalizedFile := filepath.ToSlash(frame.File)
	if normalizedFile == requestAwareGormLoggerFile {
		return true
	}
	for _, prefix := range requestAwareSkipFuncPrefix {
		if strings.HasPrefix(frame.Function, prefix) {
			return true
		}
	}
	for _, suffix := range requestAwareSkipFileSuffix {
		if strings.HasSuffix(normalizedFile, suffix) {
			return true
		}
	}
	return false
}

// callerPathWithLineNum 会按配置把调用文件转换为绝对路径或模块相对路径。
func callerPathWithLineNum(file string, line int, mode GormCallerPathMode) string {
	displayFile := displayCallerPath(file, mode)
	return string(strconv.AppendInt(append([]byte(displayFile), ':'), int64(line), 10))
}

// displayCallerPath 会按配置输出绝对路径或模块相对路径。
func displayCallerPath(file string, mode GormCallerPathMode) string {
	switch normalizeGormCallerPathMode(mode) {
	case GormCallerPathModeModuleRelative:
		return moduleRelativePath(file)
	default:
		return filepath.ToSlash(file)
	}
}

// normalizeGormCallerPathMode 用来保证未知模式回退到绝对路径，避免出现不可点击的意外格式。
func normalizeGormCallerPathMode(mode GormCallerPathMode) GormCallerPathMode {
	if mode == GormCallerPathModeModuleRelative {
		return GormCallerPathModeModuleRelative
	}
	return GormCallerPathModeAbsolute
}

// moduleRelativePath 会基于调用文件所在模块的 go.mod 根目录生成相对路径。
func moduleRelativePath(file string) string {
	normalizedFile := filepath.ToSlash(file)
	moduleRoot := findModuleRoot(filepath.Dir(file))
	if moduleRoot == "" {
		return normalizedFile
	}

	relativePath, err := filepath.Rel(moduleRoot, file)
	if err != nil {
		return normalizedFile
	}
	return filepath.ToSlash(relativePath)
}

// findModuleRoot 会向上查找 go.mod，并用缓存减少频繁 SQL 日志下的文件系统探测开销。
func findModuleRoot(startDir string) string {
	normalizedDir := filepath.ToSlash(startDir)
	if cachedRoot, ok := moduleRootCache.Load(normalizedDir); ok {
		return cachedRoot.(string)
	}

	currentDir := startDir
	visitedDirs := make([]string, 0, 8)
	for {
		normalizedCurrentDir := filepath.ToSlash(currentDir)
		visitedDirs = append(visitedDirs, normalizedCurrentDir)
		if cachedRoot, ok := moduleRootCache.Load(normalizedCurrentDir); ok {
			root := cachedRoot.(string)
			for _, visitedDir := range visitedDirs {
				moduleRootCache.Store(visitedDir, root)
			}
			return root
		}

		if fileExists(filepath.Join(currentDir, "go.mod")) {
			root := normalizedCurrentDir
			for _, visitedDir := range visitedDirs {
				moduleRootCache.Store(visitedDir, root)
			}
			return root
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			for _, visitedDir := range visitedDirs {
				moduleRootCache.Store(visitedDir, "")
			}
			return ""
		}
		currentDir = parentDir
	}
}

// fileExists 用来判断模块根目录上的 go.mod 是否存在。
func fileExists(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !fileInfo.IsDir()
}

// currentSourceFile 返回当前源文件的绝对路径，供调用栈过滤时识别包装层使用。
func currentSourceFile() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.ToSlash(file)
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
