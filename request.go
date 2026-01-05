package wd

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm/schema"
)

type ReqKeyword struct {
	Keyword     string `json:"keyword" form:"keyword"`
	keywordLike string
	isParse     bool
}

func (req *ReqKeyword) parse() {
	req.keywordLike = fmt.Sprintf("%%%s%%", strings.TrimSpace(req.Keyword))
	req.isParse = true
}
func (req *ReqKeyword) KeywordLikeValue() string {
	if req.isParse {
		return req.keywordLike
	}
	req.parse()
	return req.keywordLike
}
func (req *ReqKeyword) GenWhereFilters(table schema.Tabler, column field.IColumnName) gen.Condition {
	if strings.TrimSpace(req.Keyword) == "" {
		return nil
	}
	if !req.isParse {
		req.parse()
	}

	return field.NewString(table.TableName(), column.ColumnName().String()).Like(req.keywordLike)
}

type ReqPageSize struct {
	Page int `json:"page" form:"page"`
	Size int `json:"size" form:"size"`

	offsetValue int
	isParse     bool
}

func (req *ReqPageSize) parse(defaultSize ...int) {
	if len(defaultSize) == 0 {
		defaultSize = []int{20}
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 || req.Size > 50 {
		req.Size = defaultSize[0]
	}
	req.isParse = true
	req.offsetValue = (req.Page - 1) * req.Size
}
func (req *ReqPageSize) SetPage(page int) {
	req.isParse = false
	req.Page = page
	req.parse()
}

func (req *ReqPageSize) SetSize(size int) {
	req.isParse = false
	req.Size = size
	req.parse()
}

func (req *ReqPageSize) GetLimit() int {
	if !req.isParse {
		req.parse()
	}
	return req.Size
}

func (req *ReqPageSize) GetOffset() int {
	if !req.isParse {
		req.parse()
	}
	return req.offsetValue
}

type ReqFiles struct {
	Files []*multipart.FileHeader `form:"files"`
}

func (r *ReqFiles) UploadFiles(ctx context.Context, uploadFileFunc func(file *multipart.FileHeader) (string, error), sem ...int64) (success map[string]string, err error) {
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
		weighted = semaphore.NewWeighted(sem[0])
		fileChan = make(chan successItem, len(files))
	)
	success = make(map[string]string)

	g := new(errgroup.Group)
	for _, fileHeader := range files {
		g.Go(func() error {
			if ctxErr := weighted.Acquire(ctx, 1); err != nil {
				return ctxErr
			}
			defer weighted.Release(1)
			fileFunc, err := uploadFileFunc(fileHeader)
			if err != nil {
				return err
			}
			fileChan <- successItem{
				Filename: fileHeader.Filename,
				Url:      fileFunc,
			}
			return nil
		})
	}
	err = g.Wait()
	close(fileChan)
	if err != nil {
		return
	}
	for ele := range fileChan {
		success[ele.Filename] = ele.Url
	}
	return
}

type ReqFile struct {
	File *multipart.FileHeader `json:"file" form:"file"`

	file      multipart.File
	fileBytes []byte
	isParse   bool
	isRead    bool
}

func (r *ReqFile) Enable() bool {
	return r.File != nil
}

func (r *ReqFile) parse() error {
	if !r.Enable() {
		return ErrNotFound.WithMessage("文件不存在")
	}
	var err error
	r.file, err = r.File.Open()
	if err != nil {
		return MsgErrDataExists(err.Error())
	}
	r.isParse = true
	return nil
}
func (r *ReqFile) FileContent() ([]byte, error) {
	if !r.isParse {
		err := r.parse()
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
		return nil, MsgErrDataExists(err.Error())
	}
	return r.fileBytes, nil
}

func (r *ReqFile) GetFileContentType() (string, error) {
	if !r.isParse {
		err := r.parse()
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
		return "", MsgErrDataExists(err.Error())
	}
	r.isRead = true
	return GetFileContentType(r.fileBytes), nil
}

func (r *ReqFile) GetFileNameType() (string, error) {
	if !r.Enable() {
		return "", MsgErrNotFound("文件不存在")
	}
	return GetFileNameType(r.File.Filename), nil
}

func (r *ReqFile) GetFileSize() (int64, error) {
	if !r.Enable() {
		return 0, MsgErrNotFound("文件不存在")
	}
	return r.File.Size, nil
}

func (r *ReqFile) UploadFile(ctx context.Context, uploadFileFunc func(file *multipart.FileHeader) (string, error)) (string, error) {
	urls, err := FilesUploadGoroutine(ctx, []*multipart.FileHeader{r.File}, uploadFileFunc)
	if err != nil {
		return "", err
	}
	if v, ok := urls[r.File.Filename]; ok {
		return v, nil
	}
	return "", MsgErrRequestExternalService("上传文件失败")
}
