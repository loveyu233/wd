package wd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
)

// LogEntry 表示一个日志条目
type LogEntry struct {
	Level   zerolog.Level  `json:"level"`
	Key     string         `json:"-"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields"`
	Time    string         `json:"time"`
}

// RequestLogger 存储请求链路中的所有日志
type RequestLogger struct {
	entries    []LogEntry
	sqlEntries []SQLLogEntry
	mu         sync.Mutex
	logger     zerolog.Logger
	durationMs int64
	statusCode int
	flushed    bool
}

type SQLLogEntry struct {
	Level  zerolog.Level
	Fields map[string]any
}

// NewRequestLogger 用来创建单次请求期间使用的日志缓冲器。
func NewRequestLogger(logger zerolog.Logger) *RequestLogger {
	return &RequestLogger{
		entries:    make([]LogEntry, 0),
		sqlEntries: make([]SQLLogEntry, 0),
		logger:     logger,
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

// AddEntry 用来把一条日志事件写入缓冲区。
func (rl *RequestLogger) AddEntry(key string, level zerolog.Level, message string, fields map[string]any) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.flushed {
		return
	}

	entry := LogEntry{
		Level:   level,
		Key:     key,
		Message: message,
		Fields:  make(map[string]any),
		Time:    Now().Format(CSTLayout),
	}

	// 复制字段避免并发问题
	for k, v := range fields {
		entry.Fields[k] = v
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

	entryFields := make(map[string]any, len(fields))
	for k, v := range fields {
		entryFields[k] = v
	}
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
		case CtxKeyRespInfo, CtxKeyReqInfo:
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
	requestLogger *RequestLogger
}

// Info 用来创建记录 Info 级别日志的事件。
func (cl *ContextLogger) Info() *ContextLogEvent {
	return &ContextLogEvent{
		level:         zerolog.InfoLevel,
		requestLogger: cl.requestLogger,
		fields:        make(map[string]any),
	}
}

// Error 用来创建记录 Error 级别日志的事件。
func (cl *ContextLogger) Error() *ContextLogEvent {
	return &ContextLogEvent{
		level:         zerolog.ErrorLevel,
		requestLogger: cl.requestLogger,
		fields:        make(map[string]any),
	}
}

// Warn 用来创建记录 Warn 级别日志的事件。
func (cl *ContextLogger) Warn() *ContextLogEvent {
	return &ContextLogEvent{
		level:         zerolog.WarnLevel,
		requestLogger: cl.requestLogger,
		fields:        make(map[string]any),
	}
}

// Debug 用来创建记录 Debug 级别日志的事件。
func (cl *ContextLogger) Debug() *ContextLogEvent {
	return &ContextLogEvent{
		level:         zerolog.DebugLevel,
		requestLogger: cl.requestLogger,
		fields:        make(map[string]any),
	}
}

// ContextLogEvent 链路日志事件
type ContextLogEvent struct {
	level         zerolog.Level
	requestLogger *RequestLogger
	fields        map[string]any
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
	if e.requestLogger == nil {
		return
	}
	e.requestLogger.AddEntry(key, e.level, msg, e.fields)
}

// Msgf 用来以格式化文本写入请求日志。
func (e *ContextLogEvent) Msgf(key, format string, v ...any) {
	if e.requestLogger == nil {
		return
	}
	e.requestLogger.AddEntry(key, e.level, fmt.Sprintf(format, v...), e.fields)
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

// ResponseWriter 是对 gin.ResponseWriter 的包装，用于捕获写入的响应
type ResponseWriter struct {
	gin.ResponseWriter
	body *responseBodyCapture
}

// Write 重写 Write 方法以捕获响应内容
func (w *ResponseWriter) Write(b []byte) (int, error) {
	if w.body != nil && len(b) > 0 {
		_, _ = w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// WriteString 重写 WriteString 方法以捕获响应内容
func (w *ResponseWriter) WriteString(s string) (int, error) {
	if w.body != nil && len(s) > 0 {
		_, _ = w.body.Write([]byte(s))
	}
	return w.ResponseWriter.WriteString(s)
}

// ReadFrom 在 io.Copy 走 ReaderFrom 快路径时仍然进行限额捕获。
func (w *ResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	if rf, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		if w.body == nil || !w.body.Active() {
			return rf.ReadFrom(r)
		}
		return rf.ReadFrom(io.TeeReader(r, w.body))
	}
	if w.body == nil || !w.body.Active() {
		return io.Copy(w.ResponseWriter, r)
	}
	return io.Copy(w.ResponseWriter, io.TeeReader(r, w.body))
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
	ReqTime     time.Time         `json:"req_time"`
	Module      string            `json:"module,omitempty"`
	Option      string            `json:"option,omitempty"`
	Method      string            `json:"method,omitempty"`
	Path        string            `json:"path,omitempty"`
	URL         string            `json:"url,omitempty"`
	IP          string            `json:"ip,omitempty"`
	Content     map[string]any    `json:"content,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Params      map[string]any    `json:"params,omitempty"`
	Status      int               `json:"status,omitempty"`
	LatencyMs   int64             `json:"latency_ms,omitempty"`
	Body        any               `json:"body,omitempty"`
	RespStatus  int               `json:"resp_status"`  // 响应数据中的状态码
	RespMessage string            `json:"resp_message"` // 响应数据中的message
}

type MiddlewareLogConfig struct {
	HeaderKeys             []string
	ContentKeys            []string
	SaveLog                func(ReqLog)
	LogWriter              io.Writer
	RecordGETRequests      bool
	RecordRequestBody      bool
	RecordResponseBody     bool
	RequestBodyLimit       int64
	ResponseBodyLimit      int64
	BriefResponseBodyLimit int64
}

type FileInfo struct {
	Filename string               `json:"filename"`
	Size     int64                `json:"size"`
	Header   textproto.MIMEHeader `json:"header"`
}

const (
	defaultRequestBodyLogLimit       int64 = 32 << 10
	defaultResponseBodyLogLimit      int64 = 32 << 10
	defaultBriefResponseBodyLogLimit int64 = 8 << 10
)

type middlewareLogRuntimeConfig struct {
	headerKeys             []string
	contentKeys            []string
	saveLog                func(ReqLog)
	logWriter              io.Writer
	recordGETRequests      bool
	recordRequestBody      bool
	recordResponseBody     bool
	requestBodyLimit       int64
	responseBodyLimit      int64
	briefResponseBodyLimit int64
}

// MiddlewareLogger 用来在 gin 中记录请求与响应的详细日志。
func MiddlewareLogger(mc MiddlewareLogConfig) gin.HandlerFunc {
	cfg := buildMiddlewareLogRuntimeConfig(mc)
	baseLogger := zerolog.New(cfg.logWriter).With().
		Timestamp().
		Logger()
	return func(c *gin.Context) {
		if c.Request != nil && c.Request.Method == http.MethodGet && !cfg.recordGETRequests {
			c.Next()
			return
		}
		// 开始时间
		startTime := Now()
		requestContentType := parseMediaType(c.Request.Header.Get("Content-Type"))
		c.Set(requestContentTypeKey, requestContentType)
		c.Set(requestBodyRecordKey, cfg.recordRequestBody)

		requestBodyCapture := attachRequestBodyCapture(c, requestContentType, cfg.recordRequestBody, cfg.requestBodyLimit)
		if requestBodyCapture != nil {
			c.Set(requestBodyCaptureKey, requestBodyCapture)
		}
		// 创建自定义 ResponseWriter
		bodyBuffer := newResponseBodyCapture(cfg.responseBodyLimit, cfg.briefResponseBodyLimit)
		if cfg.recordResponseBody {
			bodyBuffer.EnableFull()
		}
		responseWriter := &ResponseWriter{
			ResponseWriter: c.Writer,
			body:           bodyBuffer,
		}
		c.Writer = responseWriter
		c.Set(responseCaptureKey, bodyBuffer)

		tracker := NewTracker()
		c.Set(trackerKey, tracker)
		// 创建请求日志器
		requestLogger := NewRequestLogger(baseLogger)
		c.Request = c.Request.WithContext(ContextWithRequestLogger(c.Request.Context(), requestLogger))
		c.Set(string(RequestLoggerKey), requestLogger)

		c.Next()
		if c.GetBool(CtxKeySkip) {
			return
		}
		requestLogger.AddEntry(HeaderTraceID, zerolog.InfoLevel, "response", map[string]any{HeaderTraceID: GetTraceID(c)})
		params := buildRequestParams(c, requestContentType, getRequestBodyCapture(c), isRequestBodyRecordingEnabled(c))

		headerMap := make(map[string]string)
		for _, item := range cfg.headerKeys {
			value := c.GetHeader(item)
			headerMap[item] = value
		}

		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		fullURL := scheme + "://" + c.Request.Host + c.Request.RequestURI

		var contentKV = make(map[string]any)
		for _, key := range cfg.contentKeys {
			value, exists := c.Get(key)
			if exists {
				contentKV[key] = value
			}
		}
		// 记录请求开始信息
		requestLogger.AddEntry(CtxKeyReqInfo, zerolog.InfoLevel, "request", map[string]any{
			"req_time":   startTime.Format(CSTLayout),
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"full_url":   fullURL,
			"req_body":   params,
			"user_agent": c.Request.UserAgent(),
			"client_ip":  c.ClientIP(),
			"header":     headerMap,
			"module":     c.GetString(CtxKeyModule),
			"option":     c.GetString(CtxKeyOption),
			"content_kv": contentKV,
		})
		for _, m := range tracker.Marks() {
			WriteGinInfoLog(c, m.key, "阶段[%s]耗时=%.2fms",
				m.Name,
				float64(m.Duration.Microseconds())/1000)
		}

		if c.GetBool(CtxKeyOnlyReq) {
			requestLogger.SetDurationMs(Now().Sub(startTime).Milliseconds())
			requestLogger.SetStatusCode(c.Writer.Status())
			// 输出所有收集的日志
			requestLogger.Flush()
			return
		}

		duration := Now().Sub(startTime)

		// 读取响应体（只读取一次）
		responseBody := buildResponseBodyPayload(c, bodyBuffer, cfg.recordResponseBody)
		bodyMap := normalizeLogFields(responseBody)

		requestLogger.AddEntry(CtxKeyRespInfo, zerolog.InfoLevel, "response", bodyMap)
		requestLogger.SetDurationMs(duration.Milliseconds())
		requestLogger.SetStatusCode(c.Writer.Status())

		// 输出所有收集的日志
		requestLogger.Flush()

		if cfg.saveLog != nil && !c.GetBool(CtxKeyNoRecord) {
			cfg.saveLog(ReqLog{
				ReqTime:     startTime,
				Module:      c.GetString(CtxKeyModule),
				Option:      c.GetString(CtxKeyOption),
				Method:      c.Request.Method,
				Path:        c.Request.URL.Path,
				URL:         fullURL,
				IP:          c.ClientIP(),
				Content:     contentKV,
				Headers:     headerMap,
				Params:      params,
				Status:      c.Writer.Status(),
				LatencyMs:   duration.Milliseconds(),
				Body:        responseBody,
				RespStatus:  c.GetInt(CtxKeyRespStatus),
				RespMessage: c.GetString(CtxKeyRespMsg),
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

// GinLogOnlyReqMsg 用来仅记录请求阶段日志。
func GinLogOnlyReqMsg() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(CtxKeyOnlyReq, true)
		c.Next()
	}
}

// GinLogBriefInformation 记录gjsonKeys指定的key的值，key的起始是整个响应的根节点，不是单个请求的返回
// 响应结构如下：如果需要获取data节点下的指定值，使用：data.xxx，具体用法参考：https://github.com/tidwall/gjson
//
//	{
//	  "code": 200,
//	  "message": "请求成功",
//	  "data": {
//	  }
//	}
func GinLogBriefInformation(gjsonKeys ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(CtxKeyBrief, true)
		c.Set(CtxKeyGjsonKeys, gjsonKeys)
		enableResponseBriefCapture(c)
		c.Next()
	}
}

// GinLogEnableRequestBody 用来为当前请求动态开启请求体记录。
func GinLogEnableRequestBody(limit ...int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		EnableGinLogRequestBody(c, limit...)
		c.Next()
	}
}

// GinLogEnableResponseBody 用来为当前请求动态开启响应体记录。
func GinLogEnableResponseBody(limit ...int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		EnableGinLogResponseBody(c, limit...)
		c.Next()
	}
}

// GinLogEnableBody 用来同时开启当前请求的请求体和响应体记录。
func GinLogEnableBody(requestLimit, responseLimit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		EnableGinLogRequestBody(c, requestLimit)
		EnableGinLogResponseBody(c, responseLimit)
		c.Next()
	}
}

// EnableGinLogRequestBody 用来在当前请求中手动开启请求体记录。
func EnableGinLogRequestBody(c *gin.Context, limit ...int64) {
	if c == nil {
		return
	}
	c.Set(requestBodyRecordKey, true)
	if getRequestBodyCapture(c) != nil {
		return
	}
	contentType, _ := c.Get(requestContentTypeKey)
	requestContentType, _ := contentType.(string)
	requestBodyLimit := defaultRequestBodyLogLimit
	if len(limit) > 0 && limit[0] > 0 {
		requestBodyLimit = limit[0]
	}
	capture := attachRequestBodyCapture(c, requestContentType, true, requestBodyLimit)
	if capture != nil {
		c.Set(requestBodyCaptureKey, capture)
	}
}

// EnableGinLogResponseBody 用来在当前请求中手动开启响应体记录。
func EnableGinLogResponseBody(c *gin.Context, limit ...int64) {
	if c == nil {
		return
	}
	v, ok := c.Get(responseCaptureKey)
	if !ok {
		return
	}
	capture, ok := v.(*responseBodyCapture)
	if !ok {
		return
	}
	if len(limit) > 0 && limit[0] > 0 {
		capture.fullLimit = limit[0]
	}
	capture.EnableFull()
}

func buildMiddlewareLogRuntimeConfig(mc MiddlewareLogConfig) middlewareLogRuntimeConfig {
	cfg := middlewareLogRuntimeConfig{
		headerKeys:             mc.HeaderKeys,
		contentKeys:            mc.ContentKeys,
		saveLog:                mc.SaveLog,
		logWriter:              mc.LogWriter,
		recordGETRequests:      mc.RecordGETRequests,
		recordRequestBody:      mc.RecordRequestBody,
		recordResponseBody:     mc.RecordResponseBody,
		requestBodyLimit:       mc.RequestBodyLimit,
		responseBodyLimit:      mc.ResponseBodyLimit,
		briefResponseBodyLimit: mc.BriefResponseBodyLimit,
	}
	if cfg.logWriter == nil {
		cfg.logWriter = os.Stdout
	}
	if cfg.requestBodyLimit <= 0 {
		cfg.requestBodyLimit = defaultRequestBodyLogLimit
	}
	if cfg.responseBodyLimit <= 0 {
		cfg.responseBodyLimit = defaultResponseBodyLogLimit
	}
	if cfg.briefResponseBodyLimit <= 0 {
		cfg.briefResponseBodyLimit = defaultBriefResponseBodyLogLimit
	}
	return cfg
}

// recordBodySkip 用来写入未记录请求体的原因。
func recordBodySkip(params map[string]any, reason string) {
	if reason == "" {
		return
	}
	if _, exists := params["body_skipped"]; !exists {
		params["body_skipped"] = reason
	}
}

func highestLogLevel(entries []LogEntry, sqlEntries []SQLLogEntry) zerolog.Level {
	level := zerolog.InfoLevel
	levelSet := false
	for _, entry := range entries {
		if entry.Key == CtxKeyReqInfo || entry.Key == CtxKeyRespInfo || entry.Key == HeaderTraceID {
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

type limitedBuffer struct {
	limit     int64
	truncated bool
	buf       bytes.Buffer
}

func newLimitedBuffer(limit int64) *limitedBuffer {
	if limit < 0 {
		limit = 0
	}
	return &limitedBuffer{limit: limit}
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b == nil || len(p) == 0 {
		return len(p), nil
	}
	remaining := b.limit - int64(b.buf.Len())
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if int64(len(p)) > remaining {
		_, _ = b.buf.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	_, _ = b.buf.Write(p)
	return len(p), nil
}

func (b *limitedBuffer) Bytes() []byte {
	if b == nil {
		return nil
	}
	return b.buf.Bytes()
}

func (b *limitedBuffer) Len() int {
	if b == nil {
		return 0
	}
	return b.buf.Len()
}

func (b *limitedBuffer) Limit() int64 {
	if b == nil {
		return 0
	}
	return b.limit
}

func (b *limitedBuffer) Truncated() bool {
	if b == nil {
		return false
	}
	return b.truncated
}

type responseCaptureMode uint8

const (
	responseCaptureDisabled responseCaptureMode = iota
	responseCaptureBrief
	responseCaptureFull
)

type responseBodyCapture struct {
	mode       responseCaptureMode
	fullLimit  int64
	briefLimit int64
	buf        *limitedBuffer
}

func newResponseBodyCapture(fullLimit, briefLimit int64) *responseBodyCapture {
	return &responseBodyCapture{
		mode:       responseCaptureDisabled,
		fullLimit:  fullLimit,
		briefLimit: briefLimit,
	}
}

func (c *responseBodyCapture) EnableFull() {
	if c == nil {
		return
	}
	if c.mode == responseCaptureFull {
		return
	}
	if c.mode == responseCaptureBrief && c.buf != nil {
		prevBytes := append([]byte(nil), c.buf.Bytes()...)
		upgraded := newLimitedBuffer(c.fullLimit)
		_, _ = upgraded.Write(prevBytes)
		if c.buf.Truncated() {
			upgraded.truncated = true
		}
		c.buf = upgraded
	}
	c.mode = responseCaptureFull
	if c.buf == nil {
		c.buf = newLimitedBuffer(c.fullLimit)
	}
}

func (c *responseBodyCapture) EnableBrief() {
	if c == nil || c.mode == responseCaptureFull || c.mode == responseCaptureBrief {
		return
	}
	c.mode = responseCaptureBrief
	c.buf = newLimitedBuffer(c.briefLimit)
}

func (c *responseBodyCapture) Active() bool {
	return c != nil && c.mode != responseCaptureDisabled
}

func (c *responseBodyCapture) Write(p []byte) (int, error) {
	if !c.Active() || len(p) == 0 {
		return len(p), nil
	}
	return c.buf.Write(p)
}

func (c *responseBodyCapture) Bytes() []byte {
	if c == nil || c.buf == nil {
		return nil
	}
	return c.buf.Bytes()
}

func (c *responseBodyCapture) Len() int {
	if c == nil || c.buf == nil {
		return 0
	}
	return c.buf.Len()
}

func (c *responseBodyCapture) Limit() int64 {
	if c == nil {
		return 0
	}
	if c.mode == responseCaptureBrief {
		return c.briefLimit
	}
	return c.fullLimit
}

func (c *responseBodyCapture) Truncated() bool {
	if c == nil || c.buf == nil {
		return false
	}
	return c.buf.Truncated()
}

type requestBodyCaptureReadCloser struct {
	io.ReadCloser
	capture *limitedBuffer
}

func (r *requestBodyCaptureReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 && r.capture != nil {
		_, _ = r.capture.Write(p[:n])
	}
	return n, err
}

func attachRequestBodyCapture(c *gin.Context, contentType string, enabled bool, limit int64) *limitedBuffer {
	if c.Request == nil || c.Request.Body == nil || c.Request.Body == http.NoBody {
		return nil
	}
	if !enabled {
		return nil
	}
	if !shouldCaptureRequestBody(contentType) {
		return nil
	}
	capture := newLimitedBuffer(limit)
	c.Request.Body = &requestBodyCaptureReadCloser{
		ReadCloser: c.Request.Body,
		capture:    capture,
	}
	return capture
}

func buildRequestParams(c *gin.Context, contentType string, bodyCapture *limitedBuffer, recordRequestBody bool) map[string]any {
	params := make(map[string]any)
	queryParams := make(map[string]any)
	for k, v := range c.Request.URL.Query() {
		if len(v) == 1 {
			queryParams[k] = v[0]
		} else {
			queryParams[k] = v
		}
	}
	if len(queryParams) > 0 {
		params["query"] = queryParams
	}

	pathParams := make(map[string]any)
	for _, param := range c.Params {
		pathParams[param.Key] = param.Value
	}
	if len(pathParams) > 0 {
		params["path"] = pathParams
	}
	if !recordRequestBody {
		if requestHasBody(c.Request, contentType) {
			recordBodySkip(params, "未开启请求体记录")
		}
		return params
	}

	switch contentType {
	case "multipart/form-data":
		appendMultipartRequestParams(c, params)
	case "application/x-www-form-urlencoded":
		appendFormRequestParams(c, params, bodyCapture)
	default:
		appendStructuredRequestBody(params, contentType, bodyCapture, c.Request.ContentLength)
	}
	return params
}

func appendMultipartRequestParams(c *gin.Context, params map[string]any) {
	form := c.Request.MultipartForm
	if form == nil {
		recordBodySkip(params, "multipart 请求体未被下游解析，已跳过记录")
		return
	}
	if len(form.Value) > 0 {
		params["form"] = valuesToMap(form.Value)
	}
	if len(form.File) == 0 {
		return
	}
	fileParams := make(map[string][]FileInfo, len(form.File))
	for key, files := range form.File {
		fileInfos := make([]FileInfo, 0, len(files))
		for _, file := range files {
			fileInfos = append(fileInfos, FileInfo{
				Filename: file.Filename,
				Size:     file.Size,
				Header:   file.Header,
			})
		}
		fileParams[key] = fileInfos
	}
	params["files"] = fileParams
}

func appendFormRequestParams(c *gin.Context, params map[string]any, bodyCapture *limitedBuffer) {
	if len(c.Request.PostForm) > 0 {
		params["form"] = valuesToMap(c.Request.PostForm)
		return
	}
	if bodyCapture == nil || bodyCapture.Len() == 0 {
		if c.Request.ContentLength > 0 {
			recordBodySkip(params, "表单请求体未被下游读取，已跳过记录")
		}
		return
	}
	if bodyCapture.Truncated() {
		recordBodySkip(params, fmt.Sprintf("表单请求体超过 %d 字节，已跳过记录", bodyCapture.Limit()))
		return
	}
	values, err := url.ParseQuery(string(bodyCapture.Bytes()))
	if err != nil {
		recordBodySkip(params, fmt.Sprintf("解析表单请求体失败: %v", err))
		return
	}
	params["form"] = valuesToMap(values)
}

func appendStructuredRequestBody(params map[string]any, contentType string, bodyCapture *limitedBuffer, contentLength int64) {
	if bodyCapture == nil || bodyCapture.Len() == 0 {
		if contentLength > 0 {
			if contentType != "" && !shouldCaptureRequestBody(contentType) {
				recordBodySkip(params, fmt.Sprintf("请求 Content-Type=%s，已跳过记录", emptyContentType(contentType)))
				return
			}
			recordBodySkip(params, "请求体未被下游读取，已跳过记录")
		}
		return
	}
	if bodyCapture.Truncated() {
		recordBodySkip(params, fmt.Sprintf("请求体超过 %d 字节，已跳过记录", bodyCapture.Limit()))
		return
	}
	bodyBytes := bodyCapture.Bytes()
	switch {
	case isJSONMediaType(contentType):
		var bodyParams any
		if err := json.Unmarshal(bodyBytes, &bodyParams); err == nil {
			params["json"] = bodyParams
		} else {
			params["json_raw"] = string(bodyBytes)
		}
	case isXMLMediaType(contentType):
		params["xml"] = string(bodyBytes)
	case contentType == "" && json.Valid(bodyBytes):
		var bodyParams any
		if err := json.Unmarshal(bodyBytes, &bodyParams); err == nil {
			params["json"] = bodyParams
		} else {
			params["json_raw"] = string(bodyBytes)
		}
	case contentType == "" && utf8.Valid(bodyBytes):
		params["raw"] = map[string]any{
			"content_type": contentType,
			"body":         string(bodyBytes),
			"size":         len(bodyBytes),
		}
	case isTextMediaType(contentType):
		params["raw"] = map[string]any{
			"content_type": contentType,
			"body":         string(bodyBytes),
			"size":         len(bodyBytes),
		}
	case len(bodyBytes) > 0:
		recordBodySkip(params, fmt.Sprintf("请求 Content-Type=%s，已跳过记录", emptyContentType(contentType)))
	}
}

func buildResponseBodyPayload(c *gin.Context, bodyCapture *responseBodyCapture, recordResponseBody bool) any {
	body, reason := parseResponseBody(c, bodyCapture, recordResponseBody)
	if reason == "" {
		return body
	}
	if body == nil {
		return map[string]any{"body_skipped": reason}
	}
	if bodyMap, ok := body.(map[string]any); ok {
		fields := make(map[string]any, len(bodyMap)+1)
		for k, v := range bodyMap {
			fields[k] = v
		}
		fields["body_skipped"] = reason
		return fields
	}
	return map[string]any{
		"body":         body,
		"body_skipped": reason,
	}
}

func parseResponseBody(c *gin.Context, bodyCapture *responseBodyCapture, recordResponseBody bool) (any, string) {
	if bodyCapture == nil || !bodyCapture.Active() {
		if c.GetBool(CtxKeyBrief) {
			return nil, "brief 响应捕获未启用"
		}
		if !recordResponseBody {
			return nil, "未开启响应体记录"
		}
		if c.Writer.Size() > 0 {
			return nil, "响应体未被记录"
		}
		return nil, ""
	}
	if bodyCapture.Len() == 0 {
		if c.Writer.Size() > 0 {
			return nil, "响应体未被记录"
		}
		return nil, ""
	}
	if bodyCapture.Truncated() {
		return nil, fmt.Sprintf("响应体超过 %d 字节，已跳过记录", bodyCapture.Limit())
	}
	if isAttachmentResponse(c.Writer.Header()) {
		return nil, "附件响应已跳过记录"
	}
	bodyBytes := bodyCapture.Bytes()
	contentType := parseMediaType(c.Writer.Header().Get("Content-Type"))
	if contentType == "" && json.Valid(bodyBytes) {
		contentType = MimeJSON
	}
	if c.GetBool(CtxKeyBrief) {
		if !isJSONMediaType(contentType) {
			return nil, fmt.Sprintf("响应 Content-Type=%s，不支持摘要提取", emptyContentType(contentType))
		}
		bodyMap := make(map[string]any)
		for _, ele := range c.GetStringSlice(CtxKeyGjsonKeys) {
			bodyMap[ele] = gjson.GetBytes(bodyBytes, ele).Value()
		}
		return bodyMap, ""
	}
	switch {
	case isJSONMediaType(contentType):
		var body any
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			return nil, fmt.Sprintf("解析响应体失败: %v", err)
		}
		return body, ""
	case isXMLMediaType(contentType), strings.HasPrefix(contentType, "text/"):
		return string(bodyBytes), ""
	case contentType == "" && utf8.Valid(bodyBytes):
		return string(bodyBytes), ""
	default:
		return nil, fmt.Sprintf("响应 Content-Type=%s，已跳过记录", emptyContentType(contentType))
	}
}

func normalizeLogFields(payload any) map[string]any {
	switch v := payload.(type) {
	case nil:
		return map[string]any{}
	case map[string]any:
		fields := make(map[string]any, len(v))
		for key, value := range v {
			fields[key] = value
		}
		return fields
	default:
		return map[string]any{"body": v}
	}
}

func valuesToMap(values map[string][]string) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		if len(value) == 1 {
			out[key] = value[0]
		} else {
			out[key] = value
		}
	}
	return out
}

func parseMediaType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return strings.ToLower(value)
	}
	return strings.ToLower(mediaType)
}

func shouldCaptureRequestBody(contentType string) bool {
	if contentType == "multipart/form-data" {
		return false
	}
	return contentType == "" || isTextMediaType(contentType)
}

func isJSONMediaType(contentType string) bool {
	return contentType == MimeJSON || strings.HasSuffix(contentType, "+json")
}

func isXMLMediaType(contentType string) bool {
	return contentType == "application/xml" || contentType == "text/xml" || strings.HasSuffix(contentType, "+xml")
}

func isTextMediaType(contentType string) bool {
	switch {
	case contentType == "application/x-www-form-urlencoded":
		return true
	case isJSONMediaType(contentType):
		return true
	case isXMLMediaType(contentType):
		return true
	case strings.HasPrefix(contentType, "text/"):
		return true
	default:
		return false
	}
}

func isAttachmentResponse(header http.Header) bool {
	contentDisposition := strings.ToLower(header.Get("Content-Disposition"))
	return strings.Contains(contentDisposition, "attachment")
}

func emptyContentType(contentType string) string {
	if contentType == "" {
		return "unknown"
	}
	return contentType
}

const (
	responseCaptureKey    = "middleware_log_response_capture"
	requestBodyCaptureKey = "middleware_log_request_capture"
	requestContentTypeKey = "middleware_log_request_content_type"
	requestBodyRecordKey  = "middleware_log_request_record_enabled"
)

func enableResponseBriefCapture(c *gin.Context) {
	v, ok := c.Get(responseCaptureKey)
	if !ok {
		return
	}
	capture, ok := v.(*responseBodyCapture)
	if !ok {
		return
	}
	capture.EnableBrief()
}

func requestHasBody(req *http.Request, contentType string) bool {
	if req == nil || req.Body == nil || req.Body == http.NoBody {
		return false
	}
	if req.ContentLength > 0 {
		return true
	}
	return req.ContentLength == -1 && contentType != ""
}

func getRequestBodyCapture(c *gin.Context) *limitedBuffer {
	if c == nil {
		return nil
	}
	v, ok := c.Get(requestBodyCaptureKey)
	if !ok {
		return nil
	}
	capture, ok := v.(*limitedBuffer)
	if !ok {
		return nil
	}
	return capture
}

func isRequestBodyRecordingEnabled(c *gin.Context) bool {
	if c == nil {
		return false
	}
	enabled, ok := c.Get(requestBodyRecordKey)
	if !ok {
		return false
	}
	flag, ok := enabled.(bool)
	return ok && flag
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
	if requestLogger, exists := c.Get(string(RequestLoggerKey)); exists {
		if rl, ok := requestLogger.(*RequestLogger); ok {
			return &ContextLogger{requestLogger: rl}
		}
	}

	// 如果获取失败，返回一个空的日志器避免 panic
	return &ContextLogger{
		requestLogger: NewRequestLogger(log.Logger),
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
