package wd

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"sync"

	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm/schema"
)

type optionalParsedValue interface {
	IsZero() bool
}

func hasOptionalParsedValue[T optionalParsedValue](value *T) bool {
	return value != nil && !(*value).IsZero()
}

type ReqRange[T CustomTime] struct {
	Start *T `json:"start" form:"start"`
	End   *T `json:"end" form:"end"`
}

// HasRange 判断范围过滤参数是否已传入。
func (req *ReqRange[T]) HasRange() bool {
	return hasOptionalRangeValue(req.Start, req.End)
}

// Validate 校验范围参数是否合法。
func (req *ReqRange[T]) Validate() error {
	return validateOptionalRangeValue(req.Start, req.End)
}

// WhereExpr 生成范围过滤表达式。
func (req *ReqRange[T]) WhereExpr(table schema.Tabler, column field.IColumnName) (field.Expr, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return genOptionalRangeWhere(table, column, req.Start, req.End)
}

func hasOptionalRangeValue[T CustomTime](start, end *T) bool {
	return hasOptionalParsedValue(start) && hasOptionalParsedValue(end)
}

func normalizeKeyword(value string) string {
	return strings.TrimSpace(value)
}

func validateOptionalRangeValue[T CustomTime](start, end *T) error {
	if !hasOptionalRangeValue(start, end) {
		return nil
	}
	if After(*start, *end) {
		return MsgErrInvalidParam(fmt.Errorf("start 不能大于 end"))
	}
	return nil
}

func genOptionalRangeWhere[T CustomTime](table schema.Tabler, column field.IColumnName, start, end *T) (field.Expr, error) {
	if !hasOptionalRangeValue(start, end) {
		return nil, nil
	}
	return GenCustomTimeBetween(table, column, *start, *end), nil
}

type ReqKeyword struct {
	Keyword string `json:"keyword" form:"keyword"`
}

func (req ReqKeyword) HasKeyword() bool {
	return normalizeKeyword(req.Keyword) != ""
}

func (req ReqKeyword) LikePattern() string {
	if !req.HasKeyword() {
		return ""
	}
	return fmt.Sprintf("%%%s%%", normalizeKeyword(req.Keyword))
}

func (req ReqKeyword) Conditions(columns ...field.String) []gen.Condition {
	if !req.HasKeyword() || len(columns) == 0 {
		return nil
	}

	likeValue := req.LikePattern()
	exprs := make([]field.Expr, 0, len(columns))
	for _, column := range columns {
		exprs = append(exprs, column.Like(likeValue))
	}

	if len(exprs) == 1 {
		return []gen.Condition{exprs[0]}
	}
	return []gen.Condition{field.Or(exprs...)}
}

type ReqPageSize struct {
	Page int `json:"page" form:"page"`
	Size int `json:"size" form:"size"`
}

type pageRootQuery[R any] interface {
	Limit(int) R
	GetFieldByName(string) (field.OrderExpr, bool)
}

type pageOffsetQuery[R any] interface {
	Offset(int) R
}

const (
	defaultReqPageSize = 20
	maxReqPageSize     = 50
)

func (r ReqPageSize) PageNumber() int {
	if r.Page <= 0 {
		return 1
	}
	return r.Page
}

func (r ReqPageSize) PageSize() int {
	if r.Size <= 0 || r.Size > maxReqPageSize {
		return defaultReqPageSize
	}
	return r.Size
}

func (r ReqPageSize) Offset() int {
	return (r.PageNumber() - 1) * r.PageSize()
}

// ApplyPage 将分页参数应用到查询对象本身上，例如 query.User。
// 不支持传入 query.User.Where(...) 这类中间接口对象。
func ApplyPage[Q pageRootQuery[R], R pageOffsetQuery[R]](page ReqPageSize, query Q) R {
	return query.Limit(page.PageSize()).Offset(page.Offset())
}

type ReqFiles struct {
	Files []*multipart.FileHeader `form:"files"`
}

func (r *ReqFiles) UploadAll(ctx context.Context, uploadFileFunc func(file *multipart.FileHeader) (string, error), sem ...int64) (success map[string]string, err error) {
	return FilesUploadGoroutine(ctx, r.Files, uploadFileFunc, sem...)
}

// FilesUploadGoroutine 用来并发上传多文件并收集结果。
func FilesUploadGoroutine(ctx context.Context, files []*multipart.FileHeader, uploadFileFunc func(file *multipart.FileHeader) (string, error), sem ...int64) (success map[string]string, err error) {
	if len(sem) == 0 {
		sem = []int64{5}
	}

	type successItem struct {
		Filename string
		Url      string
	}

	var (
		limit    = int(sem[0])
		fileChan = make(chan successItem, len(files))
	)
	success = make(map[string]string)

	if limit <= 0 {
		limit = 1
	}

	workerChan := make(chan struct{}, limit)
	errChan := make(chan error, 1)
	var wg sync.WaitGroup
	for _, fileHeader := range files {
		fileHeader := fileHeader
		wg.Add(1)
		go func() {
			defer wg.Done()

			if ctxErr := ctx.Err(); ctxErr != nil {
				sendUploadError(errChan, ctxErr)
				return
			}

			select {
			case <-ctx.Done():
				sendUploadError(errChan, ctx.Err())
				return
			case workerChan <- struct{}{}:
			}
			defer func() { <-workerChan }()

			if ctxErr := ctx.Err(); ctxErr != nil {
				sendUploadError(errChan, ctxErr)
				return
			}

			fileFunc, uploadErr := uploadFileFunc(fileHeader)
			if uploadErr != nil {
				sendUploadError(errChan, uploadErr)
				return
			}

			select {
			case <-ctx.Done():
				sendUploadError(errChan, ctx.Err())
			case fileChan <- successItem{
				Filename: fileHeader.Filename,
				Url:      fileFunc,
			}:
			}
		}()
	}
	wg.Wait()
	close(fileChan)
	close(errChan)
	if err = firstUploadError(errChan); err != nil {
		return
	}
	for ele := range fileChan {
		success[ele.Filename] = ele.Url
	}
	return
}

func sendUploadError(errChan chan error, err error) {
	if err == nil {
		return
	}
	select {
	case errChan <- err:
	default:
	}
}

func firstUploadError(errChan chan error) error {
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

type ReqFile struct {
	File *multipart.FileHeader `json:"file" form:"file"`

	file      multipart.File
	fileBytes []byte
	isParse   bool
	isRead    bool
}

func (r *ReqFile) HasFile() bool {
	return r.File != nil
}

func (r *ReqFile) open() error {
	if !r.HasFile() {
		return MsgErrNotFound("文件不存在")
	}
	var err error
	r.file, err = r.File.Open()
	if err != nil {
		return MsgErrServerBusy("文件读取失败", err)
	}
	r.isParse = true
	return nil
}
func (r *ReqFile) Content() ([]byte, error) {
	if !r.isParse {
		err := r.open()
		if err != nil {
			return nil, err
		}
	}
	if r.isRead {
		return r.fileBytes, nil
	}
	var err error
	r.fileBytes, err = io.ReadAll(r.file)
	if err != nil {
		return nil, MsgErrServerBusy("文件读取失败", err)
	}
	return r.fileBytes, nil
}

func (r *ReqFile) ContentType() (string, error) {
	if !r.isParse {
		err := r.open()
		if err != nil {
			return "", err
		}
	}
	if r.isRead {
		return GetFileContentType(r.fileBytes), nil
	}
	var err error
	r.fileBytes, err = io.ReadAll(r.file)
	if err != nil {
		return "", MsgErrServerBusy("文件读取失败", err)
	}
	r.isRead = true
	return GetFileContentType(r.fileBytes), nil
}

func (r *ReqFile) FilenameExt() (string, error) {
	if !r.HasFile() {
		return "", MsgErrNotFound("文件不存在")
	}
	return GetFileNameType(r.File.Filename), nil
}

func (r *ReqFile) Size() (int64, error) {
	if !r.HasFile() {
		return 0, MsgErrNotFound("文件不存在")
	}
	return r.File.Size, nil
}

func (r *ReqFile) Upload(ctx context.Context, uploadFileFunc func(file *multipart.FileHeader) (string, error)) (string, error) {
	urls, err := FilesUploadGoroutine(ctx, []*multipart.FileHeader{r.File}, uploadFileFunc)
	if err != nil {
		return "", err
	}
	if v, ok := urls[r.File.Filename]; ok {
		return v, nil
	}
	return "", MsgErrRequestExternalService("上传文件失败")
}
