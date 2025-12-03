package wd

import (
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/gin-gonic/gin"
)

var (
	InsCachedEnforcer *CachedEnforcer
)

type CachedEnforcer struct {
	*casbin.CachedEnforcer
}

func InitCasbin() error {
	if InsDB == nil {
		return gormClientNilErr()
	}
	adapter, err := gormadapter.NewAdapterByDB(InsDB.DB)
	if err != nil {
		return err
	}
	file, err := model.NewModelFromString(`[request_definition]
		r = sub, obj, act
		
		[policy_definition]
		p = sub, obj, act
		
		[role_definition]
		g = _, _
		
		[policy_effect]
		e = some(where (p.eft == allow))
		
		[matchers]
		m = g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && regexMatch(r.act, p.act)`)

	if err != nil {
		return err
	}
	e, err := casbin.NewCachedEnforcer(file, adapter)
	if err != nil {
		return err
	}
	InsCachedEnforcer = &CachedEnforcer{e}
	return nil
}

// InitCasbinRule 在数据库中创建casbin rule表，如果mandatory为true则会强制创建，否则则会先去检查是否存在，不存在则不创建
func (e *CachedEnforcer) InitCasbinRule(mandatory ...bool) error {
	if InsDB == nil {
		return gormClientNilErr()
	}
	if len(mandatory) == 0 || (len(mandatory) > 0 && !mandatory[0]) {
		if InsDB.DB.Migrator().HasTable(&gormadapter.CasbinRule{}) {
			return nil
		}
	}

	return InsDB.DB.AutoMigrate(gormadapter.CasbinRule{})
}

func (e *CachedEnforcer) CachedEnforce(sub, obj, act string) bool {
	enforce, _ := e.Enforce(sub, obj, act)
	return enforce
}

// GinMiddleware gin的中间件，用于检查用户权限，请求的url path会过滤掉http配置中prefix前缀
func (e *CachedEnforcer) GinMiddleware(getSubFunc func(c *gin.Context) (string, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		sub, err := getSubFunc(c)
		if err != nil {
			ResponseError(c, err)
			c.Abort()
		}
		if !InsCachedEnforcer.CachedEnforce(sub, strings.ReplaceAll(c.Request.URL.Path, globalApiPrefix, ""), c.Request.Method) {
			ResponseError(c, ErrForbiddenAuth)
			c.Abort()
		}
	}
}
