package wd

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ── 核心配置 ──────────────────────────────────────────────────

// WithJWTRealm 设置显示给用户的域名。
func WithJWTRealm(realm string) JWTOption {
	return func(mw *GinJWTMiddleware) { mw.Realm = realm }
}

// WithJWTKey 设置对称签名密钥。
func WithJWTKey(key []byte) JWTOption {
	return func(mw *GinJWTMiddleware) { mw.Key = key }
}

// WithJWTKeyFunc 设置自定义密钥获取回调，将绕过所有其他密钥设置。
func WithJWTKeyFunc(fn func(token *jwt.Token) (interface{}, error)) JWTOption {
	return func(mw *GinJWTMiddleware) { mw.KeyFunc = fn }
}

// WithJWTSigningAlgorithm 设置签名算法，默认 HS256。
func WithJWTSigningAlgorithm(alg string) JWTOption {
	return func(mw *GinJWTMiddleware) { mw.SigningAlgorithm = alg }
}

// WithJWTTimeout 设置令牌有效时长，默认一小时。
func WithJWTTimeout(d time.Duration) JWTOption {
	return func(mw *GinJWTMiddleware) { mw.Timeout = d }
}

// WithJWTMaxRefresh 设置令牌最大可刷新时长。
func WithJWTMaxRefresh(d time.Duration) JWTOption {
	return func(mw *GinJWTMiddleware) { mw.MaxRefresh = d }
}

// WithJWTIdentityKey 设置身份键名称。
func WithJWTIdentityKey(key string) JWTOption {
	return func(mw *GinJWTMiddleware) { mw.IdentityKey = key }
}

// ── 回调 ──────────────────────────────────────────────────────

// WithJWTIdentityHandler 设置身份处理函数：提取身份并判断是否放行。
// 返回 (identity, nil) 表示放行，返回 (nil, error) 表示拒绝。
func WithJWTIdentityHandler(fn func(c *gin.Context) (interface{}, error)) JWTOption {
	return func(mw *GinJWTMiddleware) { mw.IdentityHandler = fn }
}

// ── 响应回调 ──────────────────────────────────────────────────

// WithJWTUnauthorized 设置未授权响应回调。
func WithJWTUnauthorized(fn func(c *gin.Context, code int, message string)) JWTOption {
	return func(mw *GinJWTMiddleware) { mw.Unauthorized = fn }
}

// WithJWTLoginResponse 设置登录响应回调。
func WithJWTLoginResponse(fn func(c *gin.Context, code int, token string, expire time.Time)) JWTOption {
	return func(mw *GinJWTMiddleware) { mw.LoginResponse = fn }
}

// WithJWTLogoutResponse 设置注销响应回调。
func WithJWTLogoutResponse(fn func(c *gin.Context, code int)) JWTOption {
	return func(mw *GinJWTMiddleware) { mw.LogoutResponse = fn }
}

// WithJWTRefreshResponse 设置刷新响应回调。
func WithJWTRefreshResponse(fn func(c *gin.Context, code int, message string, expire time.Time)) JWTOption {
	return func(mw *GinJWTMiddleware) { mw.RefreshResponse = fn }
}

// ── Token ─────────────────────────────────────────────────────

// WithJWTTokenLookup 设置令牌来源，如 "header:Authorization,cookie:token"。
func WithJWTTokenLookup(lookup string) JWTOption {
	return func(mw *GinJWTMiddleware) { mw.TokenLookup = lookup }
}

// WithJWTTokenHeadName 设置头部中的令牌前缀，默认 "Bearer"。
func WithJWTTokenHeadName(name string) JWTOption {
	return func(mw *GinJWTMiddleware) { mw.TokenHeadName = name }
}

// WithJWTParseOptions 设置 jwt 解析选项。
func WithJWTParseOptions(opts ...jwt.ParserOption) JWTOption {
	return func(mw *GinJWTMiddleware) { mw.ParseOptions = opts }
}

// ── 子配置 ────────────────────────────────────────────────────

// WithJWTCookie 设置 Cookie 配置并启用 Cookie 发送。
func WithJWTCookie(cfg JWTCookieConfig) JWTOption {
	return func(mw *GinJWTMiddleware) { mw.Cookie = &cfg }
}

// WithJWTRSA 设置 RSA 非对称算法配置。
func WithJWTRSA(cfg JWTRSAConfig) JWTOption {
	return func(mw *GinJWTMiddleware) { mw.RSA = cfg }
}

// ── 逃生口 ────────────────────────────────────────────────────

// WithJWTCustom 提供直接修改 GinJWTMiddleware 的逃生口，用于覆盖选项函数未提供的冷门字段。
func WithJWTCustom(fn func(*GinJWTMiddleware)) JWTOption {
	return func(mw *GinJWTMiddleware) { fn(mw) }
}
