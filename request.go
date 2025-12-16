package wd

import (
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"sync"

	"gorm.io/gen"
	"gorm.io/gen/field"
)

type ReqKeyword struct {
	Keyword     string `json:"keyword" form:"keyword"`
	keywordLike string
}

func (req *ReqKeyword) parse() {
	req.keywordLike = fmt.Sprintf("%%%s%%", req.Keyword)
}
func (req *ReqKeyword) KeywordLikeValue() string {
	req.parse()
	return req.keywordLike
}
func (req *ReqKeyword) GenWhereFilters(columns field.String) gen.Condition {
	req.Keyword = strings.TrimSpace(req.Keyword)
	if req.Keyword == "" {
		return nil
	}
	req.parse()

	return columns.Like(req.keywordLike)
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

// ReqFileUploadGoroutine 用来并发上传多文件并收集结果。
func ReqFileUploadGoroutine(files []*multipart.FileHeader, uploadFileFunc func(file *multipart.FileHeader) (string, error)) (fileURLS []string, errs []error) {
	var (
		group sync.WaitGroup
		mu    sync.Mutex
	)

	for _, fileHeader := range files {
		fh := fileHeader
		group.Add(1)
		go func(header *multipart.FileHeader) {
			defer group.Done()
			url, err := uploadFileFunc(header)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			fileURLS = append(fileURLS, url)
		}(fh)
	}
	group.Wait()

	return fileURLS, errs
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

func (r *ReqFile) UploadFile(uploadFileFunc func(file *multipart.FileHeader) (string, error)) (string, error) {
	urls, errs := ReqFileUploadGoroutine([]*multipart.FileHeader{r.File}, uploadFileFunc)
	if len(errs) > 0 {
		return "", errs[0]
	}
	if len(urls) > 0 {
		return urls[0], nil
	}
	return "", MsgErrRequestExternalService("上传文件失败")
}
