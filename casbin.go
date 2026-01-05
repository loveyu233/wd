package wd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/gin-gonic/gin"
)

var (
	InsCasbin *CachedEnforcer
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
	InsCasbin = &CachedEnforcer{e}
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

// CustomEnforce 校验权限是否存在
func (e *CachedEnforcer) CustomEnforce(sub, obj, act string) bool {
	enforce, _ := e.Enforce(sub, obj, act)
	return enforce
}

// CustomGinMiddleware gin的中间件，用于检查用户权限，请求的url path会过滤掉http配置中prefix前缀
func (e *CachedEnforcer) CustomGinMiddleware(getSubFunc func(c *gin.Context) (string, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		sub, err := getSubFunc(c)
		if err != nil {
			ResponseError(c, err)
			c.Abort()
		}
		if !InsCasbin.CustomEnforce(sub, strings.ReplaceAll(c.Request.URL.Path, globalApiPrefix, ""), c.Request.Method) {
			ResponseError(c, ErrForbiddenAuth)
			c.Abort()
		}
	}
}

type CasbinPolicies struct {
	Sub string
	Obj string
	Act []string
}

// CustomAddPoliciesEx 添加策略
func (e *CachedEnforcer) CustomAddPoliciesEx(cps ...CasbinPolicies) (bool, error) {
	var cpsList = make([][]string, len(cps))
	for i, ele := range cps {
		var acts []string
		for _, item := range ele.Act {
			acts = append(acts, fmt.Sprintf("(%s)", item))
		}
		cpsList[i] = []string{ele.Sub, ele.Obj, strings.Join(acts, "|")}
	}
	return e.AddPoliciesEx(cpsList)
}

// CustomRemovePoliciesEx 删除策略
func (e *CachedEnforcer) CustomRemovePoliciesEx(cps ...CasbinPolicies) (bool, error) {
	var cpsList = make([][]string, len(cps))
	for i, ele := range cps {
		var acts []string
		for _, item := range ele.Act {
			acts = append(acts, fmt.Sprintf("(%s)", item))
		}
		cpsList[i] = []string{ele.Sub, ele.Obj, strings.Join(acts, "|")}
	}
	return e.RemovePolicies(cpsList)
}

// CustomAddRolesForUser 给一个用户添加一个或者多个角色
func (e *CachedEnforcer) CustomAddRolesForUser(user string, roles ...string) (bool, error) {
	if len(roles) == 0 {
		return false, errors.New("roles is empty")
	}

	rulesMap, err := e.CustomHasRules(roles...)
	if err != nil {
		return false, err
	}
	for _, role := range roles {
		if v := rulesMap[role]; !v {
			return false, errors.New(fmt.Sprintf("角色%s不存在", role))
		}
	}
	return e.AddRolesForUser(user, roles)
}

// CustomDeleteRoleForUser 删除一个用户的角色
func (e *CachedEnforcer) CustomDeleteRoleForUser(user string, role string) (bool, error) {
	return e.DeleteRoleForUser(user, role)
}

// CustomDeleteAllRoleForUser 删除一个用户的全部角色
func (e *CachedEnforcer) CustomDeleteAllRoleForUser(user string) (bool, error) {
	return e.DeleteRolesForUser(user)
}

// CustomDeleteUser 删除用户
func (e *CachedEnforcer) CustomDeleteUser(user string) (bool, error) {
	return e.DeleteUser(user)
}

// CustomDeleteRole 删除角色
func (e *CachedEnforcer) CustomDeleteRole(role string) (bool, error) {
	return e.DeleteRole(role)
}

// CustomGetPermissionsForRole 获取角色的全部权限
func (e *CachedEnforcer) CustomGetPermissionsForRole(role string) ([]CasbinPolicies, error) {
	rolePermissions, err := e.GetPermissionsForUser(role)
	if err != nil {
		return nil, err
	}
	var cpsList = make([]CasbinPolicies, len(rolePermissions))
	for i, ele := range rolePermissions {
		if len(ele) != 3 {
			return nil, errors.New("casbin角色权限错误")
		}
		var acts []string
		for _, item := range strings.Split(ele[2], "|") {
			acts = append(acts, strings.ReplaceAll(strings.ReplaceAll(item, ")", ""), "(", ""))
		}
		cpsList[i] = CasbinPolicies{
			Sub: ele[0],
			Obj: ele[1],
			Act: acts,
		}
	}
	return cpsList, nil
}

// CustomGetRolesForUser 获取一个用户的全部角色
func (e *CachedEnforcer) CustomGetRolesForUser(user string) ([]string, error) {
	rolesForUser, err := e.GetRolesForUser(user)
	if err != nil {
		return nil, err
	}
	return rolesForUser, nil
}

// CustomGetUserAllInfo 获取一个用户的全部信息,key为角色，value为角色对应的权限
func (e *CachedEnforcer) CustomGetUserAllInfo(user string) (map[string][]CasbinPolicies, error) {
	roles, err := e.CustomGetRolesForUser(user)
	if err != nil {
		return nil, err
	}
	var rolePermissions = make(map[string][]CasbinPolicies)
	role, err := e.CustomGetPermissionsForRole(user)
	if err != nil {
		return nil, err
	}
	rolePermissions[user] = role

	for _, role := range roles {
		policies, err := e.CustomGetPermissionsForRole(role)
		if err != nil {
			return nil, err
		}
		rolePermissions[role] = policies
	}

	return rolePermissions, nil
}

// CustomHasRules 判断是否存在rules这些角色
func (e *CachedEnforcer) CustomHasRules(rules ...string) (rulesMap map[string]bool, err error) {
	rulesMap = make(map[string]bool)
	if len(rules) == 0 {
		return
	}

	type ruleCount struct {
		Role  string `gorm:"column:role"`
		Count int64  `gorm:"column:count"`
	}

	var ruleCountList []*ruleCount
	if err = InsDB.DB.
		Model(&gormadapter.CasbinRule{}).
		Where("ptype = 'p' and v0 in ?", rules).
		Select("v0 'role', count(1) 'count' ").
		Group("v0").
		Find(&ruleCountList).Error; err != nil {
		return
	}

	for _, ele := range ruleCountList {
		rulesMap[ele.Role] = ele.Count > 0
	}
	return
}
