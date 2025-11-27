package wd

import (
	"fmt"
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
	Page        int `json:"page" form:"page"`
	Size        int `json:"size" form:"size"`
	offsetValue int
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
	req.offsetValue = (req.Page - 1) * req.Size
}

//func (req *ReqPageSize) GenWhereFilters(do gen.Dao) gen.Dao {
//	req.parse()
//	return do.Limit(req.Size).Offset(req.offsetValue)
//}

func (req *ReqPageSize) Limit() int {
	req.parse()
	return req.Size
}

func (req *ReqPageSize) Offset() int {
	req.parse()
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
