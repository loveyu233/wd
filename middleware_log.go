package wd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"sync"
	"time"

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
	sqlEntries []map[string]any
	mu         sync.RWMutex
	ctx        context.Context
	logger     zerolog.Logger
	durationMs int64
	statusCode int
}

// NewRequestLogger 用来创建单次请求期间使用的日志缓冲器。
func NewRequestLogger(ctx context.Context, logger zerolog.Logger) *RequestLogger {
	return &RequestLogger{
		entries:    make([]LogEntry, 0),
		sqlEntries: make([]map[string]any, 0),
		ctx:        ctx,
		logger:     logger,
	}
}
func (r *RequestLogger) SetDurationMs(durationMs int64) {
	r.durationMs = durationMs
}
func (r *RequestLogger) SetStatusCode(statusCode int) {
	r.statusCode = statusCode
}

// AddEntry 用来把一条日志事件写入缓冲区。
func (rl *RequestLogger) AddEntry(key string, level zerolog.Level, message string, fields map[string]any) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

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
func (rl *RequestLogger) AddSQLEntry(fields map[string]any) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entryFields := make(map[string]any, len(fields))
	for k, v := range fields {
		entryFields[k] = v
	}
	rl.sqlEntries = append(rl.sqlEntries, entryFields)
}

// Flush 用来把收集到的日志一次性输出到底层日志器。
func (rl *RequestLogger) Flush() {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if len(rl.entries) == 0 {
		return
	}

	// 构建合并的日志事件
	event := rl.logger.Info()

	// 添加所有收集的日志条目
	var latencyMsInfoArr []string
	for _, entry := range rl.entries {
		if entry.Key == "resp_info" || entry.Key == "req_info" {
			event = event.Any(entry.Key, entry.Fields)
		} else if entry.Key == "latency_ms_info" {
			latencyMsInfoArr = append(latencyMsInfoArr, entry.Message)
		} else {
			event = event.Any(entry.Key, entry)
		}
	}
	if len(latencyMsInfoArr) > 0 {
		event = event.Strs("latency_ms_info", latencyMsInfoArr)
	}

	if len(rl.sqlEntries) > 0 {
		event = event.Any("sql", rl.sqlEntries)
	}

	event = event.Int64("duration_ms", rl.durationMs)
	event = event.Int("status_code", rl.statusCode)
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
	e.requestLogger.AddEntry(key, e.level, msg, e.fields)
}

// Msgf 用来以格式化文本写入请求日志。
func (e *ContextLogEvent) Msgf(key, format string, v ...any) {
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
			return requestLogger.(*RequestLogger)
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
	body *bytes.Buffer
}

// Write 重写 Write 方法以捕获响应内容
func (w ResponseWriter) Write(b []byte) (int, error) {
	// 写入到缓冲区
	w.body.Write(b)
	// 继续原始的写入操作
	return w.ResponseWriter.Write(b)
}

// WriteString 重写 WriteString 方法以捕获响应内容
func (w ResponseWriter) WriteString(s string) (int, error) {
	// 写入到缓冲区
	w.body.WriteString(s)
	// 继续原始的写入操作
	return w.ResponseWriter.WriteString(s)
}

var zlog zerolog.Logger

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
	Body        map[string]any    `json:"body,omitempty"`
	RespStatus  int               `json:"resp_status"`  // 响应数据中的状态码
	RespMessage string            `json:"resp_message"` // 响应数据中的message
}

type MiddlewareLogConfig struct {
	HeaderKeys  []string
	ContentKeys []string
	SaveLog     func(ReqLog)
	LogWriter   io.Writer
}

type FileInfo struct {
	Filename string               `json:"filename"`
	Size     int64                `json:"size"`
	Header   textproto.MIMEHeader `json:"header"`
}

// MiddlewareLogger 用来在 gin 中记录请求与响应的详细日志。
func MiddlewareLogger(mc MiddlewareLogConfig) gin.HandlerFunc {
	if mc.LogWriter == nil {
		mc.LogWriter = os.Stdout
	}
	zlog = zerolog.New(mc.LogWriter).With().
		Timestamp().
		Logger()
	return func(c *gin.Context) {
		// 开始时间
		startTime := Now()
		// 创建自定义 ResponseWriter
		bodyBuffer := &bytes.Buffer{}
		responseWriter := &ResponseWriter{
			ResponseWriter: c.Writer,
			body:           bodyBuffer,
		}
		c.Writer = responseWriter

		tracker := NewTracker()
		c.Set(trackerKey, tracker)
		// 创建请求日志器
		requestLogger := NewRequestLogger(c.Request.Context(), zlog)
		c.Set(string(RequestLoggerKey), requestLogger)

		c.Next()
		if c.GetBool("skip") {
			//for _, m := range tracker.Marks() {
			//	WriteGinInfoLog(c, m.Name, "[%s] 运行至此总耗时=%.2fms 当前阶段耗时=%.2fms",
			//		m.Name,
			//		float64(m.SinceStart.Microseconds())/1000,
			//		float64(m.SincePrev.Microseconds())/1000)
			//}
			//requestLogger.SetDurationMs(Now().Sub(startTime).Milliseconds())
			//requestLogger.Flush()
			c.Next()
			return
		}

		// 获取请求参数，分类存储
		params := make(map[string]any)

		// 1. 处理URL查询参数 (query parameters)
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

		// 2. 处理路径参数 (path parameters)
		pathParams := make(map[string]any)
		for _, param := range c.Params {
			pathParams[param.Key] = param.Value
		}
		if len(pathParams) > 0 {
			params["path"] = pathParams
		}

		// 3. 获取请求体和处理不同类型的参数
		contentType := c.ContentType()

		if strings.Contains(contentType, "multipart/form-data") {
			// 处理 multipart/form-data（包含文件上传）
			err := c.Request.ParseMultipartForm(32 << 20) // 32MB 最大内存
			if err == nil && c.Request.MultipartForm != nil {
				// 处理普通表单字段
				formData := make(map[string]any)
				for key, values := range c.Request.MultipartForm.Value {
					if len(values) == 1 {
						formData[key] = values[0]
					} else {
						formData[key] = values
					}
				}
				if len(formData) > 0 {
					params["form"] = formData
				}

				// 处理文件字段
				fileParams := make(map[string][]map[string]FileInfo)
				for key, files := range c.Request.MultipartForm.File {
					fileInfos := make([]map[string]FileInfo, len(files))
					for i, file := range files {
						fileInfos[i] = map[string]FileInfo{
							key: {
								Filename: file.Filename,
								Size:     file.Size,
								Header:   file.Header,
							},
						}
					}
					fileParams[key] = fileInfos
				}
				if len(fileParams) > 0 {
					params["files"] = fileParams
				}
			}
		} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
			// 处理表单编码数据
			err := c.Request.ParseForm()
			if err == nil {
				formData := make(map[string]any)
				for key, values := range c.Request.PostForm {
					if len(values) == 1 {
						formData[key] = values[0]
					} else {
						formData[key] = values
					}
				}
				if len(formData) > 0 {
					params["form"] = formData
				}
			}
		} else if strings.Contains(contentType, "application/json") {
			if ok, reason := shouldCaptureRequestBody(c.Request); ok {
				if requestBody, err := io.ReadAll(c.Request.Body); err == nil {
					c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
					if len(requestBody) > 0 {
						var bodyParams any
						if err := json.Unmarshal(requestBody, &bodyParams); err == nil {
							params["json"] = bodyParams
						} else {
							params["json_raw"] = string(requestBody)
						}
					}
				} else {
					recordBodySkip(params, fmt.Sprintf("failed to read request body: %v", err))
				}
			} else {
				recordBodySkip(params, reason)
			}
		} else if strings.Contains(contentType, "application/xml") || strings.Contains(contentType, "text/xml") {
			if ok, reason := shouldCaptureRequestBody(c.Request); ok {
				if requestBody, err := io.ReadAll(c.Request.Body); err == nil {
					c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
					if len(requestBody) > 0 {
						params["xml"] = string(requestBody)
					}
				} else {
					recordBodySkip(params, fmt.Sprintf("failed to read request body: %v", err))
				}
			} else {
				recordBodySkip(params, reason)
			}
		} else {
			if ok, reason := shouldCaptureRequestBody(c.Request); ok {
				if requestBody, err := io.ReadAll(c.Request.Body); err == nil {
					c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
					if len(requestBody) > 0 {
						params["raw"] = map[string]any{
							"content_type": contentType,
							"body":         string(requestBody),
							"size":         len(requestBody),
						}
					}
				} else {
					recordBodySkip(params, fmt.Sprintf("failed to read request body: %v", err))
				}
			} else {
				recordBodySkip(params, reason)
			}
		}

		headerMap := make(map[string]string)
		for _, item := range mc.HeaderKeys {
			value := c.GetHeader(item)
			headerMap[item] = value
		}

		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		fullURL := scheme + "://" + c.Request.Host + c.Request.RequestURI

		var contentKV = make(map[string]any)
		for _, key := range mc.ContentKeys {
			value, exists := c.Get(key)
			if exists {
				contentKV[key] = value
			}
		}
		// 记录请求开始信息
		requestLogger.AddEntry("req_info", zerolog.InfoLevel, "request", map[string]any{
			"req_time":   startTime.Format(CSTLayout),
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"full_url":   fullURL,
			"req_body":   params,
			"user_agent": c.Request.UserAgent(),
			"client_ip":  c.ClientIP(),
			"header":     headerMap,
			"module":     c.GetString("module"),
			"option":     c.GetString("option"),
			"content_kv": contentKV,
		})
		for _, m := range tracker.Marks() {
			WriteGinInfoLog(c, m.key, "运行至[%s]总耗时=%.2fms 上阶段运行到此耗时=%.2fms",
				m.Name,
				float64(m.SinceStart.Microseconds())/1000,
				float64(m.SincePrev.Microseconds())/1000)
		}

		if c.GetBool("only-req") {
			// 输出所有收集的日志
			requestLogger.Flush()
			requestLogger.SetDurationMs(Now().Sub(startTime).Milliseconds())
			requestLogger.SetStatusCode(c.Writer.Status())
			c.Next()
			return
		}

		duration := Now().Sub(startTime)

		bodyMap := make(map[string]any)
		if !c.GetBool("brief") {
			readAll, err := io.ReadAll(io.NopCloser(bodyBuffer))
			if err != nil {
				recordBodySkip(params, fmt.Sprintf("读取请求体失败: %v", err))
			} else {
				if err := json.Unmarshal(readAll, &bodyMap); err != nil {
					recordBodySkip(params, fmt.Sprintf("解析请求体失败: %v", err))
				}
			}
		} else {
			readAll, err := io.ReadAll(io.NopCloser(bodyBuffer))
			for _, ele := range c.GetStringSlice("gjsonkeys") {
				if err != nil {
					recordBodySkip(params, fmt.Sprintf("读取请求体失败: %v", err))
				} else {
					bodyMap[ele] = gjson.GetBytes(readAll, ele).Value()
				}
			}
		}

		requestLogger.AddEntry("resp_info", zerolog.InfoLevel, "response", bodyMap)
		requestLogger.SetDurationMs(duration.Milliseconds())
		requestLogger.SetStatusCode(c.Writer.Status())

		// 输出所有收集的日志
		requestLogger.Flush()

		if mc.SaveLog != nil && !c.GetBool("no_record") {
			mc.SaveLog(ReqLog{
				ReqTime:     startTime,
				Module:      c.GetString("module"),
				Option:      c.GetString("option"),
				Method:      c.Request.Method,
				Path:        c.Request.URL.Path,
				URL:         fullURL,
				IP:          c.ClientIP(),
				Content:     contentKV,
				Headers:     headerMap,
				Params:      params,
				Status:      c.Writer.Status(),
				LatencyMs:   duration.Milliseconds(),
				Body:        bodyMap,
				RespStatus:  c.GetInt("resp-status"),
				RespMessage: c.GetString("resp-msg"),
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
		c.Set("module", name)
		c.Next()
	}
}

// GinLogSetOptionName 用来记录操作名称并可选择不持久化日志。
func GinLogSetOptionName(name string, noRecord ...bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("option", name)
		if len(noRecord) > 0 && noRecord[0] {
			c.Set("no_record", true)
		}
		c.Next()
	}
}

// GinLogSetSkipLogFlag 用来标记当前请求跳过日志流程。
func GinLogSetSkipLogFlag() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("skip", true)
		c.Next()
	}
}

// GinLogOnlyReqMsg 用来仅记录请求阶段日志。
func GinLogOnlyReqMsg() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("only-req", true)
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
		c.Set("brief", true)
		c.Set("gjsonkeys", gjsonKeys)
		c.Next()
	}
}

// shouldCaptureRequestBody 用来判断请求体是否可被记录并给出原因。
func shouldCaptureRequestBody(r *http.Request) (bool, string) {
	if r == nil {
		return false, "request is nil"
	}
	if r.Body == nil || r.Body == http.NoBody {
		return false, "无请求数据"
	}
	if r.ContentLength == -1 {
		return false, "请求数据错误大小为-1"
	}
	return true, ""
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

var defaultMaskedHeaders = []string{"authorization"}

type Mark struct {
	key        string
	Name       string
	SinceStart time.Duration
	SincePrev  time.Duration
}

type Tracker struct {
	mu    sync.Mutex
	start time.Time
	last  time.Time
	marks []Mark
}

func NewTracker() *Tracker {
	now := time.Now()
	return &Tracker{start: now, last: now}
}

func (t *Tracker) Mark(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.marks = append(t.marks, Mark{
		key:        "latency_ms_info",
		Name:       name,
		SinceStart: now.Sub(t.start),
		SincePrev:  now.Sub(t.last),
	})
	t.last = now
}

func (t *Tracker) Marks() []Mark {
	t.mu.Lock()
	defer t.mu.Unlock()

	out := make([]Mark, len(t.marks))
	copy(out, t.marks)
	return out
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
		requestLogger: NewRequestLogger(context.Background(), log.Logger),
	}
}

// SetRecordTimeFlag 设置一个用于记录耗时的标志，会在后续的日志中输出耗时信息。
func SetRecordTimeFlag(c *gin.Context, flagName string) {
	v, ok := c.Get(trackerKey)
	if !ok {
		return
	}
	tracker, ok := v.(*Tracker)
	if ok {
		tracker.Mark(flagName)
	}
}
