package wd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LogEntry 表示一个日志条目
type LogEntry struct {
	Level   zerolog.Level  `json:"level"`
	Key     string         `json:"-"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields"`
	Payload any            `json:"payload,omitempty"`
	Time    string         `json:"time"`
}

// RequestLogger 存储请求链路中的所有日志
type RequestLogger struct {
	entries    []LogEntry
	sqlEntries []SQLLogEntry
	mu         sync.Mutex
	logger     zerolog.Logger
	snapshot   logSnapshotMode
	durationMs int64
	statusCode int
	flushed    bool
}

type SQLLogEntry struct {
	Level  zerolog.Level
	Fields map[string]any
}

// LogSnapshotMode 用来控制请求日志在写入缓冲区时如何冻结字段和载荷。
type LogSnapshotMode string

const (
	// LogSnapshotModeShallow 默认只拷贝 map/slice/array 等容器，兼顾性能与隔离。
	LogSnapshotModeShallow LogSnapshotMode = "shallow"
	// LogSnapshotModeDeepJSON 使用 JSON 编解码做深拷贝，隔离最强但性能开销更高。
	LogSnapshotModeDeepJSON LogSnapshotMode = "deep_json"
	// LogSnapshotModeNone 不做快照拷贝，性能最好，但后续修改原对象可能污染日志。
	LogSnapshotModeNone LogSnapshotMode = "none"
)

type logSnapshotMode string

const (
	logSnapshotModeShallow  logSnapshotMode = logSnapshotMode(LogSnapshotModeShallow)
	logSnapshotModeDeepJSON logSnapshotMode = logSnapshotMode(LogSnapshotModeDeepJSON)
	logSnapshotModeNone     logSnapshotMode = logSnapshotMode(LogSnapshotModeNone)
)

// NewRequestLogger 用来创建单次请求期间使用的日志缓冲器。
func NewRequestLogger(logger zerolog.Logger) *RequestLogger {
	return NewRequestLoggerWithSnapshotMode(logger, LogSnapshotModeShallow)
}

// NewRequestLoggerWithSnapshotMode 用来创建带指定快照模式的请求日志缓冲器。
func NewRequestLoggerWithSnapshotMode(logger zerolog.Logger, mode LogSnapshotMode) *RequestLogger {
	return &RequestLogger{
		entries:    make([]LogEntry, 0),
		sqlEntries: make([]SQLLogEntry, 0),
		logger:     logger,
		snapshot:   normalizeLogSnapshotMode(mode),
	}
}

func (rl *RequestLogger) SetDurationMs(durationMs int64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.flushed {
		return
	}
	rl.durationMs = durationMs
}

func (rl *RequestLogger) SetStatusCode(statusCode int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.flushed {
		return
	}
	rl.statusCode = statusCode
}

// Entries 返回当前已收集日志条目的快照，供持久化等场景复用。
func (rl *RequestLogger) Entries() []LogEntry {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entries := make([]LogEntry, len(rl.entries))
	copy(entries, rl.entries)
	return entries
}

// AddEntry 用来把一条日志事件写入缓冲区。
func (rl *RequestLogger) AddEntry(key string, level zerolog.Level, message string, fields map[string]any) {
	rl.addEntry(key, level, message, fields, nil)
}

// AddEntryAny 用来把携带任意对象载荷的日志事件写入缓冲区。
func (rl *RequestLogger) AddEntryAny(key string, level zerolog.Level, payload any, fields map[string]any) {
	rl.addEntry(key, level, "", fields, payload)
}

func (rl *RequestLogger) addEntry(key string, level zerolog.Level, message string, fields map[string]any, payload any) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.flushed {
		return
	}

	entry := LogEntry{
		Level:   level,
		Key:     key,
		Message: message,
		Fields:  snapshotFields(fields, rl.snapshot),
		Payload: snapshotValue(payload, rl.snapshot),
		Time:    Now().Format(CSTLayout),
	}

	rl.entries = append(rl.entries, entry)
}

// AddSQLEntry 用来记录 SQL 相关的字段，会在 Flush 时作为一个数组输出。
func (rl *RequestLogger) AddSQLEntry(level zerolog.Level, fields map[string]any) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.flushed {
		return
	}

	entryFields := snapshotFields(fields, rl.snapshot)
	entryFields["level"] = level.String()
	rl.sqlEntries = append(rl.sqlEntries, SQLLogEntry{
		Level:  level,
		Fields: entryFields,
	})
}

// Flush 用来把收集到的日志一次性输出到底层日志器。
func (rl *RequestLogger) Flush() {
	rl.mu.Lock()
	if len(rl.entries) == 0 && len(rl.sqlEntries) == 0 {
		rl.flushed = true
		rl.mu.Unlock()
		return
	}
	if rl.flushed {
		rl.mu.Unlock()
		return
	}
	entries := append([]LogEntry(nil), rl.entries...)
	sqlEntries := append([]SQLLogEntry(nil), rl.sqlEntries...)
	durationMs := rl.durationMs
	statusCode := rl.statusCode
	rl.entries = nil
	rl.sqlEntries = nil
	rl.flushed = true
	rl.mu.Unlock()

	level := highestLogLevel(entries, sqlEntries)
	event := rl.logger.WithLevel(level)

	// 添加所有收集的日志条目
	var latencyMsInfoArr []string
	for _, entry := range entries {
		switch entry.Key {
		case CtxKeyReqInfo:
			event = event.Any(entry.Key, entry.Fields)
		case CtxKeyLatencyMsInfo:
			latencyMsInfoArr = append(latencyMsInfoArr, entry.Message)
		case HeaderTraceID:
			event = event.Any("trace_id", entry.Fields[HeaderTraceID])
		default:
			event = event.Any(entry.Key, entry)
		}
	}
	if len(latencyMsInfoArr) > 0 {
		event = event.Strs(CtxKeyLatencyMsInfo, latencyMsInfoArr)
	}

	if len(sqlEntries) > 0 {
		sqlFields := make([]map[string]any, 0, len(sqlEntries))
		for _, entry := range sqlEntries {
			sqlFields = append(sqlFields, entry.Fields)
		}
		event = event.Any("sql", sqlFields)
	}

	event = event.Int64(CtxKeyDurationMs, durationMs)
	event = event.Int(CtxKeyStatusCode, statusCode)
	event.Msg("")
}

// ContextLogger 提供链路日志记录功能
type ContextLogger struct {
	requestLogger  *RequestLogger
	fallbackLogger zerolog.Logger
	useFallback    bool
}

// Info 用来创建记录 Info 级别日志的事件。
func (cl *ContextLogger) Info() *ContextLogEvent {
	return &ContextLogEvent{
		level:          zerolog.InfoLevel,
		requestLogger:  cl.requestLogger,
		fallbackLogger: cl.fallbackLogger,
		useFallback:    cl.useFallback,
		fields:         make(map[string]any),
	}
}

// Error 用来创建记录 Error 级别日志的事件。
func (cl *ContextLogger) Error() *ContextLogEvent {
	return &ContextLogEvent{
		level:          zerolog.ErrorLevel,
		requestLogger:  cl.requestLogger,
		fallbackLogger: cl.fallbackLogger,
		useFallback:    cl.useFallback,
		fields:         make(map[string]any),
	}
}

// Warn 用来创建记录 Warn 级别日志的事件。
func (cl *ContextLogger) Warn() *ContextLogEvent {
	return &ContextLogEvent{
		level:          zerolog.WarnLevel,
		requestLogger:  cl.requestLogger,
		fallbackLogger: cl.fallbackLogger,
		useFallback:    cl.useFallback,
		fields:         make(map[string]any),
	}
}

// Debug 用来创建记录 Debug 级别日志的事件。
func (cl *ContextLogger) Debug() *ContextLogEvent {
	return &ContextLogEvent{
		level:          zerolog.DebugLevel,
		requestLogger:  cl.requestLogger,
		fallbackLogger: cl.fallbackLogger,
		useFallback:    cl.useFallback,
		fields:         make(map[string]any),
	}
}

// ContextLogEvent 链路日志事件
type ContextLogEvent struct {
	level          zerolog.Level
	requestLogger  *RequestLogger
	fallbackLogger zerolog.Logger
	useFallback    bool
	fields         map[string]any
}

// Str 用来为当前日志事件添加字符串字段。
func (e *ContextLogEvent) Str(key, val string) *ContextLogEvent {
	e.fields[key] = val
	return e
}

// Int 用来为日志事件添加整数字段。
func (e *ContextLogEvent) Int(key string, val int) *ContextLogEvent {
	e.fields[key] = val
	return e
}

// Float64 用来为日志事件添加浮点数字段。
func (e *ContextLogEvent) Float64(key string, val float64) *ContextLogEvent {
	e.fields[key] = val
	return e
}

// Bool 用来为日志事件添加布尔字段。
func (e *ContextLogEvent) Bool(key string, val bool) *ContextLogEvent {
	e.fields[key] = val
	return e
}

// Err 用来把错误详情附加到日志事件。
func (e *ContextLogEvent) Err(err error) *ContextLogEvent {
	if err != nil {
		e.fields["error"] = err.Error()
	}
	return e
}

// Interface 用来为日志事件添加任意类型字段。
func (e *ContextLogEvent) Interface(key string, val any) *ContextLogEvent {
	e.fields[key] = val
	return e
}

// Dur 用来为日志事件添加持续时间信息。
func (e *ContextLogEvent) Dur(key string, d time.Duration) *ContextLogEvent {
	e.fields[key] = d.String()
	return e
}

// Msg 用来将事件写入请求日志缓冲区。
func (e *ContextLogEvent) Msg(key, msg string) {
	if e.requestLogger != nil {
		e.requestLogger.AddEntry(key, e.level, msg, e.fields)
		return
	}
	if !e.useFallback {
		return
	}
	e.writeFallback(key, msg, nil)
}

// Msgf 用来以格式化文本写入请求日志。
func (e *ContextLogEvent) Msgf(key, format string, v ...any) {
	if e.requestLogger != nil {
		e.requestLogger.AddEntry(key, e.level, fmt.Sprintf(format, v...), e.fields)
		return
	}
	if !e.useFallback {
		return
	}
	e.writeFallback(key, fmt.Sprintf(format, v...), nil)
}

// MsgAny 用来把任意对象作为日志载荷写入请求日志。
func (e *ContextLogEvent) MsgAny(key string, payload any) {
	if e.requestLogger != nil {
		e.requestLogger.AddEntryAny(key, e.level, payload, e.fields)
		return
	}
	if !e.useFallback {
		return
	}
	e.writeFallback(key, "", payload)
}

func (e *ContextLogEvent) writeFallback(key, message string, payload any) {
	entry := LogEntry{
		Level:   e.level,
		Key:     key,
		Message: message,
		Fields:  snapshotFields(e.fields, logSnapshotModeShallow),
		Payload: snapshotValue(payload, logSnapshotModeShallow),
		Time:    Now().Format(CSTLayout),
	}
	e.fallbackLogger.WithLevel(e.level).Any(key, entry).Msg("")
}

// 上下文键
type contextKey string

const (
	RequestLoggerKey contextKey = "request_logger"
)

// ContextWithRequestLogger 将请求级日志器注入到 context.Context 中，方便链路外部（例如gorm）取用。
func ContextWithRequestLogger(ctx context.Context, rl *RequestLogger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, RequestLoggerKey, rl)
}

// RequestLoggerFromContext 用来从context.Context中提取请求级日志器。
func RequestLoggerFromContext(ctx context.Context) *RequestLogger {
	if ctx == nil {
		return nil
	}
	if gc, ok := ctx.(*gin.Context); ok {
		if requestLogger, exists := gc.Get(string(RequestLoggerKey)); exists {
			if rl, ok := requestLogger.(*RequestLogger); ok {
				return rl
			}
		}
	}
	if rl, ok := ctx.Value(RequestLoggerKey).(*RequestLogger); ok {
		return rl
	}
	return nil
}

// init 用来初始化 zerolog 配置并构建默认日志器。
func init() {
	//zerolog.TimeFieldFormat = CSTLayout
	zerolog.TimestampFunc = func() time.Time {
		return Now()
	}
	zerolog.TimeFieldFormat = CSTLayout
}

type ReqLog struct {
	ReqTime   time.Time         `json:"req_time"`
	Module    string            `json:"module,omitempty"`
	Option    string            `json:"option,omitempty"`
	Method    string            `json:"method,omitempty"`
	URL       string            `json:"url,omitempty"`
	IP        string            `json:"ip,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Status    int               `json:"status,omitempty"`
	LatencyMs int64             `json:"latency_ms,omitempty"`
	Logs      []LogEntry        `json:"logs,omitempty"`
}

type MiddlewareLogConfig struct {
	HeaderKeys   []string
	SaveLog      func(ReqLog)
	LogWriter    io.Writer
	SnapshotMode LogSnapshotMode
}

type middlewareLogRuntimeConfig struct {
	headerKeys   []string
	saveLog      func(ReqLog)
	logWriter    io.Writer
	snapshotMode logSnapshotMode
}

// MiddlewareLogger 用来记录请求摘要信息以及业务主动写入的链路日志。
func MiddlewareLogger(mc MiddlewareLogConfig) gin.HandlerFunc {
	cfg := buildMiddlewareLogRuntimeConfig(mc)
	baseLogger := zerolog.New(cfg.logWriter).With().
		Timestamp().
		Logger()
	return func(c *gin.Context) {
		startTime := Now()
		method := ""
		if c.Request != nil {
			method = c.Request.Method
		}
		url := buildRequestURL(c)
		ip := c.ClientIP()

		tracker := NewTracker()
		c.Set(trackerKey, tracker)
		requestLogger := NewRequestLoggerWithSnapshotMode(baseLogger, LogSnapshotMode(cfg.snapshotMode))
		if c.Request != nil {
			c.Request = c.Request.WithContext(ContextWithRequestLogger(c.Request.Context(), requestLogger))
		}
		c.Set(string(RequestLoggerKey), requestLogger)

		c.Next()
		if c.GetBool(CtxKeySkip) {
			return
		}
		headerMap := buildSelectedHeaders(c, cfg.headerKeys)
		module := c.GetString(CtxKeyModule)
		option := c.GetString(CtxKeyOption)
		traceID := GetTraceID(c)
		if traceID != "" {
			requestLogger.AddEntry(HeaderTraceID, zerolog.InfoLevel, "trace_id", map[string]any{
				HeaderTraceID: traceID,
			})
		}
		requestLogger.AddEntry(CtxKeyReqInfo, zerolog.InfoLevel, "request", map[string]any{
			"method":  method,
			"url":     url,
			"ip":      ip,
			"module":  module,
			"option":  option,
			"headers": headerMap,
		})
		for _, m := range tracker.Marks() {
			WriteGinInfoLog(c, m.key, "阶段[%s]耗时=%.2fms",
				m.Name,
				float64(m.Duration.Microseconds())/1000)
		}

		duration := Now().Sub(startTime)
		entrySnapshot := requestLogger.Entries()
		requestLogger.SetDurationMs(duration.Milliseconds())
		requestLogger.SetStatusCode(c.Writer.Status())
		requestLogger.Flush()

		if cfg.saveLog != nil && !c.GetBool(CtxKeyNoRecord) {
			cfg.saveLog(ReqLog{
				ReqTime:   startTime,
				Module:    module,
				Option:    option,
				Method:    method,
				URL:       url,
				IP:        ip,
				Headers:   headerMap,
				Status:    c.Writer.Status(),
				LatencyMs: duration.Milliseconds(),
				Logs:      filterPersistLogEntries(entrySnapshot),
			})
		}
	}
}

// WriteGinInfoLog 用来在当前请求记录 Info 级别日志。
func WriteGinInfoLog(c *gin.Context, key, format string, args ...any) {
	GetContextLogger(c).Info().Msgf(key, format, args...)
}

// WriteGinDebugLog 用来在当前请求记录 Debug 级别日志。
func WriteGinDebugLog(c *gin.Context, key, format string, args ...any) {
	GetContextLogger(c).Debug().Msgf(key, format, args...)
}

// WriteGinWarnLog 用来在当前请求记录 Warn 级别日志。
func WriteGinWarnLog(c *gin.Context, key, format string, args ...any) {
	GetContextLogger(c).Warn().Msgf(key, format, args...)
}

// WriteGinErrLog 用来在当前请求记录 Error 级别日志。
func WriteGinErrLog(c *gin.Context, key, format string, args ...any) {
	GetContextLogger(c).Error().Msgf(key, format, args...)
}

// WriteGinInfoAnyLog 用来在当前请求记录携带任意对象载荷的 Info 级别日志。
func WriteGinInfoAnyLog(c *gin.Context, key string, payload any) {
	GetContextLogger(c).Info().MsgAny(key, payload)
}

// WriteGinDebugAnyLog 用来在当前请求记录携带任意对象载荷的 Debug 级别日志。
func WriteGinDebugAnyLog(c *gin.Context, key string, payload any) {
	GetContextLogger(c).Debug().MsgAny(key, payload)
}

// WriteGinWarnAnyLog 用来在当前请求记录携带任意对象载荷的 Warn 级别日志。
func WriteGinWarnAnyLog(c *gin.Context, key string, payload any) {
	GetContextLogger(c).Warn().MsgAny(key, payload)
}

// WriteGinErrAnyLog 用来在当前请求记录携带任意对象载荷的 Error 级别日志。
func WriteGinErrAnyLog(c *gin.Context, key string, payload any) {
	GetContextLogger(c).Error().MsgAny(key, payload)
}

// GinLogSetModuleName 用来在上下文中标记模块名称。
func GinLogSetModuleName(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(CtxKeyModule, name)
		c.Next()
	}
}

// GinLogSetOptionName 用来记录操作名称并可选择不持久化日志。
func GinLogSetOptionName(name string, noRecord ...bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(CtxKeyOption, name)
		if len(noRecord) > 0 && noRecord[0] {
			c.Set(CtxKeyNoRecord, true)
		}
		c.Next()
	}
}

// GinLogSetSkipLogFlag 用来标记当前请求跳过日志流程。
func GinLogSetSkipLogFlag() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(CtxKeySkip, true)
		c.Next()
	}
}

func buildMiddlewareLogRuntimeConfig(mc MiddlewareLogConfig) middlewareLogRuntimeConfig {
	cfg := middlewareLogRuntimeConfig{
		headerKeys:   mc.HeaderKeys,
		saveLog:      mc.SaveLog,
		logWriter:    mc.LogWriter,
		snapshotMode: normalizeLogSnapshotMode(mc.SnapshotMode),
	}
	if cfg.logWriter == nil {
		cfg.logWriter = os.Stdout
	}
	return cfg
}

func buildSelectedHeaders(c *gin.Context, keys []string) map[string]string {
	if c == nil || len(keys) == 0 {
		return nil
	}
	headers := make(map[string]string, 0)
	for _, key := range keys {
		if key == "" {
			continue
		}
		headers[key] = c.GetHeader(key)
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}

// buildRequestURL 按优先级拼装请求 URL，避免日志中出现空字符串。
func buildRequestURL(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if c.Request.URL == nil {
		return c.Request.RequestURI
	}
	if c.Request.URL.IsAbs() {
		return c.Request.URL.String()
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if c.Request.Host != "" {
		return scheme + "://" + c.Request.Host + c.Request.URL.RequestURI()
	}
	if c.Request.URL.RequestURI() != "" {
		return c.Request.URL.RequestURI()
	}
	return c.Request.URL.String()
}

// filterPersistLogEntries 过滤掉中间件自动注入的基础字段，仅保留主动写入的请求日志条目。
func filterPersistLogEntries(entries []LogEntry) []LogEntry {
	if len(entries) == 0 {
		return nil
	}
	out := make([]LogEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Key == CtxKeyReqInfo || entry.Key == HeaderTraceID {
			continue
		}
		out = append(out, entry)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func highestLogLevel(entries []LogEntry, sqlEntries []SQLLogEntry) zerolog.Level {
	level := zerolog.InfoLevel
	levelSet := false
	for _, entry := range entries {
		if entry.Key == CtxKeyReqInfo || entry.Key == HeaderTraceID {
			continue
		}
		if !levelSet || entry.Level > level {
			level = entry.Level
			levelSet = true
		}
	}
	for _, entry := range sqlEntries {
		if !levelSet || entry.Level > level {
			level = entry.Level
			levelSet = true
		}
	}
	if !levelSet {
		return zerolog.InfoLevel
	}
	return level
}

type Mark struct {
	key      string
	Name     string
	Duration time.Duration
}

type Tracker struct {
	mu    sync.RWMutex
	marks []Mark
}

type StageTiming struct {
	tracker *Tracker
	name    string
	start   time.Time
	once    sync.Once
}

func NewTracker() *Tracker {
	return &Tracker{}
}

func (t *Tracker) Begin(name string) *StageTiming {
	if t == nil {
		return &StageTiming{name: name}
	}
	return &StageTiming{
		tracker: t,
		name:    name,
		start:   Now(),
	}
}

func (t *Tracker) Commit(name string, start time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := Now()
	t.marks = append(t.marks, Mark{
		key:      CtxKeyLatencyMsInfo,
		Name:     name,
		Duration: now.Sub(start),
	})
}

func (t *Tracker) Marks() []Mark {
	t.mu.RLock()
	defer t.mu.RUnlock()

	out := make([]Mark, len(t.marks))
	copy(out, t.marks)
	return out
}

func (f *StageTiming) Commit() {
	if f == nil || f.tracker == nil {
		return
	}
	f.once.Do(func() {
		f.tracker.Commit(f.name, f.start)
	})
}

const trackerKey = "record_time_flag"

// GetContextLogger 用来从 gin.Context 获取请求级日志器。
func GetContextLogger(c *gin.Context) *ContextLogger {
	if c != nil {
		if requestLogger, exists := c.Get(string(RequestLoggerKey)); exists {
			if rl, ok := requestLogger.(*RequestLogger); ok {
				return &ContextLogger{requestLogger: rl}
			}
		}
	}

	// 如果当前不在请求日志链路中，则退化为立即输出，避免日志被静默吞掉。
	return &ContextLogger{
		fallbackLogger: log.Logger,
		useFallback:    true,
	}
}

func snapshotFields(fields map[string]any, mode logSnapshotMode) map[string]any {
	if len(fields) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(fields))
	for key, value := range fields {
		cloned[key] = snapshotValue(value, mode)
	}
	return cloned
}

func snapshotValue(value any, mode logSnapshotMode) any {
	if value == nil {
		return nil
	}
	if errValue, ok := value.(error); ok {
		return errValue.Error()
	}
	if stringer, ok := value.(fmt.Stringer); ok {
		return stringer.String()
	}
	switch mode {
	case logSnapshotModeNone:
		return value
	case logSnapshotModeDeepJSON:
		return snapshotValueDeepJSON(value)
	default:
		return snapshotValueShallow(value)
	}
}

func snapshotValueDeepJSON(value any) any {
	data, err := json.Marshal(value)
	if err == nil {
		var cloned any
		if err = json.Unmarshal(data, &cloned); err == nil {
			return cloned
		}
	}
	return fmt.Sprintf("%+v", value)
}

// snapshotValueShallow 用来递归拷贝 map/slice/array 容器，避免日志记录后原容器继续被修改。
func snapshotValueShallow(value any) any {
	return cloneContainerValue(reflect.ValueOf(value), map[containerVisitKey]reflect.Value{})
}

type containerVisitKey struct {
	kind reflect.Kind
	typ  reflect.Type
	ptr  uintptr
}

func cloneContainerValue(value reflect.Value, visited map[containerVisitKey]reflect.Value) any {
	if !value.IsValid() {
		return nil
	}

	switch value.Kind() {
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type()).Interface()
		}
		visitKey := containerVisitKey{kind: value.Kind(), typ: value.Type(), ptr: value.Pointer()}
		if cloned, ok := visited[visitKey]; ok {
			return cloned.Interface()
		}
		cloned := reflect.MakeMapWithSize(value.Type(), value.Len())
		visited[visitKey] = cloned
		iter := value.MapRange()
		for iter.Next() {
			cloned.SetMapIndex(iter.Key(), cloneContainerTypedValue(iter.Value(), value.Type().Elem(), visited))
		}
		return cloned.Interface()
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type()).Interface()
		}
		visitKey := containerVisitKey{kind: value.Kind(), typ: value.Type(), ptr: value.Pointer()}
		if cloned, ok := visited[visitKey]; ok {
			return cloned.Interface()
		}
		cloned := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		visited[visitKey] = cloned
		for i := 0; i < value.Len(); i++ {
			cloned.Index(i).Set(cloneContainerTypedValue(value.Index(i), value.Type().Elem(), visited))
		}
		return cloned.Interface()
	case reflect.Array:
		cloned := reflect.New(value.Type()).Elem()
		for i := 0; i < value.Len(); i++ {
			cloned.Index(i).Set(cloneContainerTypedValue(value.Index(i), value.Type().Elem(), visited))
		}
		return cloned.Interface()
	default:
		return value.Interface()
	}
}

func cloneContainerTypedValue(value reflect.Value, targetType reflect.Type, visited map[containerVisitKey]reflect.Value) reflect.Value {
	if !value.IsValid() {
		return reflect.Zero(targetType)
	}

	clonedValue := cloneContainerValue(value, visited)
	if clonedValue == nil {
		return reflect.Zero(targetType)
	}

	clonedReflectValue := reflect.ValueOf(clonedValue)
	if clonedReflectValue.Type().AssignableTo(targetType) {
		return clonedReflectValue
	}
	if targetType.Kind() == reflect.Interface && clonedReflectValue.Type().Implements(targetType) {
		return clonedReflectValue
	}
	if clonedReflectValue.Type().ConvertibleTo(targetType) {
		return clonedReflectValue.Convert(targetType)
	}
	return value
}

func normalizeLogSnapshotMode(mode LogSnapshotMode) logSnapshotMode {
	switch mode {
	case LogSnapshotModeNone:
		return logSnapshotModeNone
	case LogSnapshotModeDeepJSON:
		return logSnapshotModeDeepJSON
	case LogSnapshotModeShallow, "":
		return logSnapshotModeShallow
	default:
		return logSnapshotModeShallow
	}
}

// BeginStageTiming 创建一个阶段耗时记录器，调用 Commit 后会把该阶段耗时写入请求日志。
func BeginStageTiming(c *gin.Context, stageName string) *StageTiming {
	if c == nil {
		return &StageTiming{name: stageName}
	}
	v, ok := c.Get(trackerKey)
	if !ok {
		return &StageTiming{name: stageName}
	}
	tracker, ok := v.(*Tracker)
	if !ok {
		return &StageTiming{name: stageName}
	}
	return tracker.Begin(stageName)
}
