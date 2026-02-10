package wd

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// MapClaims 使用 map[string]interface{} 进行 JSON 解码的类型
// 如果你没有提供一个，这是默认的声明类型
type MapClaims map[string]interface{}

// JWTCookieConfig 聚合所有与 Cookie 相关的配置字段。
type JWTCookieConfig struct {
	// Cookie 名称，默认 "jwt"。
	Name string
	// Cookie 有效时长，默认等于 Timeout。
	MaxAge time.Duration
	// Cookie 域名。
	Domain string
	// 是否仅通过 HTTPS 发送。
	Secure bool
	// 是否禁止 JS 访问。
	HTTPOnly bool
	// SameSite 策略。
	SameSite http.SameSite
}

// JWTRSAConfig 聚合所有与 RSA 非对称算法相关的配置字段。
type JWTRSAConfig struct {
	// 私钥文件路径，优先于 PrivKeyBytes。
	PrivKeyFile string
	// 私钥 PEM 字节。
	PrivKeyBytes []byte
	// 公钥文件路径，优先于 PubKeyBytes。
	PubKeyFile string
	// 公钥 PEM 字节。
	PubKeyBytes []byte
	// 私钥密码短语。
	PrivateKeyPassphrase string
}

// GinJWTMiddleware 提供 Json-Web-Token 身份验证实现。失败时，返回 401 HTTP 响应
// 成功时，调用包装的中间件，并将 userID 设置为 c.Get("userID").(string)
// 用户可以通过向 LoginHandler 发送 json 请求来获取令牌。然后需要在 Authentication 头中传递令牌
// 示例：Authorization:Bearer XXX_TOKEN_XXX
type GinJWTMiddleware struct {
	// 显示给用户的域名。必需。
	Realm string

	// 签名算法 - 可能的值是 HS256, HS384, HS512, RS256, RS384 或 RS512
	// 可选，默认是 HS256。
	SigningAlgorithm string

	// 用于签名的密钥。必需。
	Key []byte

	// 回调函数检索用于签名的密钥。设置 KeyFunc 将绕过所有其他密钥设置
	KeyFunc func(token *jwt.Token) (interface{}, error)

	// jwt 令牌有效的持续时间。可选，默认为一小时。
	Timeout time.Duration
	// 回调函数，将覆盖默认的 Timeout 持续时间
	// 可选，默认返回一小时
	TimeoutFunc func(claims jwt.MapClaims) time.Duration

	// 此字段允许客户端刷新其令牌，直到 MaxRefresh 过期。
	// 请注意，客户端可以在 MaxRefresh 的最后一刻刷新其令牌。
	// 这意味着令牌的最大有效时间跨度是 TokenTime + MaxRefresh。
	// 可选，默认为 0 表示不可刷新。
	MaxRefresh time.Duration

	// 回调函数，应基于登录信息执行用户身份验证。
	// 必须返回用户数据作为用户标识符，它将存储在声明数组中。必需。
	// 检查错误 (e) 以确定适当的错误消息。
	Authenticator func(c *gin.Context) (interface{}, error)

	// 身份处理函数：从 JWT Claims 中提取身份并判断是否放行。
	// 返回 (identity, nil) 表示放行，返回 (nil, error) 表示拒绝。
	// 可选，默认从 claims 中按 IdentityKey 提取身份并放行。
	IdentityHandler func(c *gin.Context) (interface{}, error)

	// 在登录期间将被调用的回调函数。
	// 使用此函数可以向 webtoken 添加其他有效负载数据。
	// 然后，在请求期间通过 c.Get("JWT_PAYLOAD") 可以获得数据。
	// 请注意，有效负载未加密。
	// jwt.io 上提到的属性不能用作映射的键。
	// 可选，默认情况下不会设置其他数据。
	PayloadFunc func(data interface{}) MapClaims

	// 用户可以定义自己的未授权函数。
	Unauthorized func(c *gin.Context, code int, message string)

	// 用户可以定义自己的登录响应函数。
	LoginResponse func(c *gin.Context, code int, token string, time time.Time)

	// 用户可以定义自己的注销响应函数。
	LogoutResponse func(c *gin.Context, code int)

	// 用户可以定义自己的刷新响应函数。
	RefreshResponse func(c *gin.Context, code int, message string, time time.Time)

	// 设置身份键
	IdentityKey string

	// TokenLookup 是 "<source>:<name>" 形式的字符串，用于从请求中提取令牌。
	// 可选。默认值 "header:Authorization"。
	// 可能的值：
	// - "header:<name>"
	// - "query:<name>"
	// - "cookie:<name>"
	// - "param:<name>"
	// - "form:<name>"
	TokenLookup string

	// TokenHeadName 是头部中的字符串。默认值是 "Bearer"
	TokenHeadName string

	// WithoutDefaultTokenHeadName 允许设置空的 TokenHeadName
	WithoutDefaultTokenHeadName bool

	// TimeFunc 提供当前时间。你可以覆盖它来使用另一个时间值。这对于测试或如果你的服务器使用与令牌不同的时区很有用。
	TimeFunc func() time.Time

	// 当 JWT 中间件中的某些东西失败时的 HTTP 状态消息。
	// 检查错误 (e) 以确定适当的错误消息。
	HTTPStatusMessageFunc func(e error, c *gin.Context) string

	// RSA 非对称算法配置。
	RSA JWTRSAConfig

	// 私钥
	privKey *rsa.PrivateKey

	// 公钥
	pubKey *rsa.PublicKey

	// Cookie 配置，设置后自动启用 Cookie 发送。
	Cookie *JWTCookieConfig

	// SendAuthorization 允许为每个请求返回授权头
	SendAuthorization bool

	// 禁用上下文的 abort()。
	DisabledAbort bool

	// ParseOptions 允许修改 jwt 的解析方法
	ParseOptions []jwt.ParserOption
}

// JWTOption 是 GinJWTMiddleware 的函数选项类型。
type JWTOption func(*GinJWTMiddleware)

// NewGinJWTMiddleware 使用函数选项创建并初始化 GinJWTMiddleware。
func NewGinJWTMiddleware(opts ...JWTOption) (*GinJWTMiddleware, error) {
	mw := &GinJWTMiddleware{}
	for _, opt := range opts {
		opt(mw)
	}
	return mw, mw.init()
}

// readKeys 用来加载配置中的私钥和公钥文件。
func (mw *GinJWTMiddleware) readKeys() error {
	err := mw.privateKey()
	if err != nil {
		return err
	}
	err = mw.publicKey()
	if err != nil {
		return err
	}
	return nil
}

// privateKey 用来读取并解析 RSA 私钥。
func (mw *GinJWTMiddleware) privateKey() error {
	var keyData []byte
	if mw.RSA.PrivKeyFile == "" {
		keyData = mw.RSA.PrivKeyBytes
	} else {
		filecontent, err := os.ReadFile(mw.RSA.PrivKeyFile)
		if err != nil {
			return MsgErrTokenServerInvalid("登陆凭证生成失败", err)
		}
		keyData = filecontent
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(keyData)
	if err != nil {
		return MsgErrTokenServerInvalid("密钥解析失败", err)
	}
	mw.privKey = key
	return nil
}

// publicKey 用来读取并解析 RSA 公钥。
func (mw *GinJWTMiddleware) publicKey() error {
	var keyData []byte
	if mw.RSA.PubKeyFile == "" {
		keyData = mw.RSA.PubKeyBytes
	} else {
		filecontent, err := os.ReadFile(mw.RSA.PubKeyFile)
		if err != nil {
			return MsgErrTokenServerInvalid("登陆凭证生成失败", err)
		}
		keyData = filecontent
	}

	key, err := jwt.ParseRSAPublicKeyFromPEM(keyData)
	if err != nil {
		return MsgErrTokenServerInvalid("密钥解析失败", err)
	}
	mw.pubKey = key
	return nil
}

// usingPublicKeyAlgo 用来判断签名算法是否为公钥算法。
func (mw *GinJWTMiddleware) usingPublicKeyAlgo() bool {
	switch mw.SigningAlgorithm {
	case "RS256", "RS512", "RS384":
		return true
	}
	return false
}

// init 用来填充中间件的默认配置与依赖。
func (mw *GinJWTMiddleware) init() error {
	if mw.TokenLookup == "" {
		mw.TokenLookup = "header:Authorization"
	}

	if mw.SigningAlgorithm == "" {
		mw.SigningAlgorithm = "HS256"
	}

	if mw.Timeout == 0 {
		mw.Timeout = time.Hour
	}

	if mw.TimeoutFunc == nil {
		mw.TimeoutFunc = func(claims jwt.MapClaims) time.Duration {
			return mw.Timeout
		}
	}

	if mw.TimeFunc == nil {
		mw.TimeFunc = time.Now
	}

	mw.TokenHeadName = strings.TrimSpace(mw.TokenHeadName)
	if len(mw.TokenHeadName) == 0 && !mw.WithoutDefaultTokenHeadName {
		mw.TokenHeadName = "Bearer"
	}

	if mw.Unauthorized == nil {
		mw.Unauthorized = func(c *gin.Context, code int, message string) {
			ResponseError(c, MsgErrUnauthorized(message))
		}
	}

	if mw.LoginResponse == nil {
		mw.LoginResponse = func(c *gin.Context, code int, token string, expire time.Time) {
			if code == http.StatusOK {
				ResponseSuccessToken(c, token)
			} else {
				ResponseError(c, MsgErrBadRequest("登录失败"))
			}
		}
	}

	if mw.LogoutResponse == nil {
		mw.LogoutResponse = func(c *gin.Context, code int) {
			if code == http.StatusOK {
				ResponseSuccessMsg(c, "操作成功")
			} else {
				ResponseError(c, MsgErrServerBusy("操作失败"))
			}
		}
	}

	if mw.RefreshResponse == nil {
		mw.RefreshResponse = func(c *gin.Context, code int, token string, expire time.Time) {
			if code == http.StatusOK {
				ResponseSuccessToken(c, token)
			} else {
				ResponseError(c, MsgErrTokenServerInvalid("登陆凭证生成失败"))
			}
		}
	}

	if mw.IdentityKey == "" {
		mw.IdentityKey = "identity"
	}

	if mw.IdentityHandler == nil {
		mw.IdentityHandler = func(c *gin.Context) (interface{}, error) {
			claims := ExtractClaims(c)
			return claims[mw.IdentityKey], nil
		}
	}

	if mw.HTTPStatusMessageFunc == nil {
		mw.HTTPStatusMessageFunc = func(e error, c *gin.Context) string {
			return ConvertToAppError(e).Message
		}
	}

	if mw.Realm == "" {
		mw.Realm = "token"
	}

	// Cookie 默认值
	if mw.Cookie != nil {
		if mw.Cookie.MaxAge == 0 {
			mw.Cookie.MaxAge = mw.Timeout
		}
		if mw.Cookie.Name == "" {
			mw.Cookie.Name = "jwt"
		}
	}

	// 密钥校验
	if mw.KeyFunc != nil {
		return nil
	}

	if mw.usingPublicKeyAlgo() {
		return mw.readKeys()
	}

	if mw.Key == nil {
		return MsgErrTokenServerInvalid("密钥不能为空")
	}
	return nil
}

// MiddlewareFunc 用来返回执行 JWT 校验的 gin.HandlerFunc。
func (mw *GinJWTMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		mw.middlewareImpl(c)
	}
}

// middlewareImpl 用来解析令牌、校验权限并写入上下文。
func (mw *GinJWTMiddleware) middlewareImpl(c *gin.Context) {
	claims, err := mw.GetClaimsFromJWT(c)
	if err != nil {
		var appErr *AppError
		if errors.Is(err, jwt.ErrTokenExpired) {
			appErr = MsgErrTokenServerInvalid("登陆过期请重新登录", err)
		} else {
			appErr = MsgErrTokenServerInvalid("登陆凭证无效请重新登录", err)
		}
		mw.unauthorized(c, appErr.Code, mw.HTTPStatusMessageFunc(appErr, c))
		return
	}

	if err := mw.validateExpiration(claims); err != nil {
		mw.unauthorized(c, err.Code, mw.HTTPStatusMessageFunc(err, c))
		return
	}

	c.Set(CtxKeyJWTPayload, claims)
	identity, err := mw.IdentityHandler(c)
	if err != nil {
		mw.unauthorized(c, errForbiddenAuth.Code, mw.HTTPStatusMessageFunc(err, c))
		return
	}

	if identity != nil {
		c.Set(mw.IdentityKey, identity)
	}

	c.Next()
}

// validateExpiration 校验 claims 中的 exp 是否过期。
func (mw *GinJWTMiddleware) validateExpiration(claims MapClaims) *AppError {
	now := mw.TimeFunc().Unix()
	switch v := claims["exp"].(type) {
	case float64:
		if int64(v) < now {
			return MsgErrTokenServerInvalid("登陆过期请重新登录")
		}
	case json.Number:
		n, err := v.Int64()
		if err != nil || n < now {
			return MsgErrTokenServerInvalid("登陆过期请重新登录", err)
		}
	default:
		return MsgErrTokenServerInvalid("登陆凭证无效请重新登录")
	}
	return nil
}

// GetClaimsFromJWT 用来解析请求中的 JWT 并返回 Claims。
func (mw *GinJWTMiddleware) GetClaimsFromJWT(c *gin.Context) (MapClaims, error) {
	token, err := mw.ParseToken(c)
	if err != nil {
		return nil, err
	}

	if mw.SendAuthorization {
		if v, ok := c.Get(CtxKeyJWTToken); ok {
			c.Header("Authorization", mw.TokenHeadName+" "+v.(string))
		}
	}

	claims := MapClaims{}
	for key, value := range token.Claims.(jwt.MapClaims) {
		claims[key] = value
	}

	return claims, nil
}

// LoginHandler 用来处理登录请求并签发访问令牌。
func (mw *GinJWTMiddleware) LoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if mw.Authenticator == nil {
			appError := MsgErrTokenServerInvalid("缺少必要的函数定义")
			mw.unauthorized(c, appError.Code, mw.HTTPStatusMessageFunc(appError, c))
			return
		}

		data, apperr := mw.Authenticator(c)
		if apperr != nil {
			ResponseError(c, apperr)
			c.Abort()
			return
		}

		// 创建令牌
		token := jwt.New(jwt.GetSigningMethod(mw.SigningAlgorithm))
		claims := token.Claims.(jwt.MapClaims)

		if mw.PayloadFunc != nil {
			for key, value := range mw.PayloadFunc(data) {
				claims[key] = value
			}
		}

		copyClaims := make(jwt.MapClaims, len(claims))
		for k, v := range claims {
			copyClaims[k] = v
		}

		expire := mw.TimeFunc().Add(mw.TimeoutFunc(copyClaims))
		claims["exp"] = expire.Unix()
		claims["orig_iat"] = mw.TimeFunc().Unix()
		tokenString, err := mw.signedString(token)
		if err != nil {
			appError := MsgErrTokenServerInvalid("创建登陆凭证失败", err)
			mw.unauthorized(c, appError.Code, mw.HTTPStatusMessageFunc(appError, c))
			return
		}

		// 设置 cookie
		if mw.Cookie != nil {
			expireCookie := mw.TimeFunc().Add(mw.Cookie.MaxAge)
			maxage := int(expireCookie.Unix() - mw.TimeFunc().Unix())
			c.SetSameSite(mw.Cookie.SameSite)
			c.SetCookie(mw.Cookie.Name, tokenString, maxage, "/", mw.Cookie.Domain, mw.Cookie.Secure, mw.Cookie.HTTPOnly)
		}

		mw.LoginResponse(c, http.StatusOK, tokenString, expire)
	}
}

// LogoutHandler 用来清理客户端 cookie 并返回退出响应。
func (mw *GinJWTMiddleware) LogoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 删除认证 cookie
		if mw.Cookie != nil {
			c.SetSameSite(mw.Cookie.SameSite)
			c.SetCookie(mw.Cookie.Name, "", -1, "/", mw.Cookie.Domain, mw.Cookie.Secure, mw.Cookie.HTTPOnly)
		}

		mw.LogoutResponse(c, http.StatusOK)
	}
}

// signedString 用来根据配置的密钥对 token 进行签名。
func (mw *GinJWTMiddleware) signedString(token *jwt.Token) (string, error) {
	var tokenString string
	var err error
	if mw.usingPublicKeyAlgo() {
		tokenString, err = token.SignedString(mw.privKey)
	} else {
		tokenString, err = token.SignedString(mw.Key)
	}
	return tokenString, err
}

// RefreshHandler 用来响应刷新令牌的 HTTP 请求。
func (mw *GinJWTMiddleware) RefreshHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, expire, err := mw.RefreshToken(c)
		if err != nil {
			appErr := MsgErrTokenServerInvalid("登陆凭证刷新失败", err)
			mw.unauthorized(c, appErr.Code, mw.HTTPStatusMessageFunc(appErr, c))
			return
		}

		mw.RefreshResponse(c, http.StatusOK, tokenString, expire)
	}
}

// RefreshToken 用来校验旧令牌并生成新的 JWT。
func (mw *GinJWTMiddleware) RefreshToken(c *gin.Context) (string, time.Time, error) {
	claims, err := mw.CheckIfTokenExpire(c)
	if err != nil {
		return "", time.Now(), err
	}

	// 创建令牌
	newToken := jwt.New(jwt.GetSigningMethod(mw.SigningAlgorithm))
	newClaims := newToken.Claims.(jwt.MapClaims)
	copyClaims := make(jwt.MapClaims, len(claims))

	for k, v := range claims {
		newClaims[k] = claims[k]
		copyClaims[k] = v
	}

	expire := mw.TimeFunc().Add(mw.TimeoutFunc(copyClaims))
	newClaims["exp"] = expire.Unix()
	newClaims["orig_iat"] = mw.TimeFunc().Unix()
	tokenString, err := mw.signedString(newToken)
	if err != nil {
		return "", time.Now(), err
	}

	// 设置 cookie
	if mw.Cookie != nil {
		expireCookie := mw.TimeFunc().Add(mw.Cookie.MaxAge)
		maxage := int(expireCookie.Unix() - time.Now().Unix())
		c.SetSameSite(mw.Cookie.SameSite)
		c.SetCookie(mw.Cookie.Name, tokenString, maxage, "/", mw.Cookie.Domain, mw.Cookie.Secure, mw.Cookie.HTTPOnly)
	}

	return tokenString, expire, nil
}

// CheckIfTokenExpire 用来校验令牌是否在可刷新时间范围内。
func (mw *GinJWTMiddleware) CheckIfTokenExpire(c *gin.Context) (jwt.MapClaims, error) {
	token, err := mw.ParseToken(c)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, MsgErrTokenClientInvalid("登陆凭证无效请重新登录")
	}

	claims := token.Claims.(jwt.MapClaims)

	origIat := int64(claims["orig_iat"].(float64))

	if origIat < mw.TimeFunc().Add(-mw.MaxRefresh).Unix() {
		return nil, MsgErrTokenClientInvalid("登陆过期请重新登录")
	}

	return claims, nil
}

// TokenGenerator 用来根据自定义数据生成 JWT 及过期时间。
func (mw *GinJWTMiddleware) TokenGenerator(data interface{}) (string, time.Time, error) {
	token := jwt.New(jwt.GetSigningMethod(mw.SigningAlgorithm))
	claims := token.Claims.(jwt.MapClaims)

	if mw.PayloadFunc != nil {
		for key, value := range mw.PayloadFunc(data) {
			claims[key] = value
		}
	}

	copyClaims := make(jwt.MapClaims, len(claims))
	for k, v := range claims {
		copyClaims[k] = v
	}

	expire := mw.TimeFunc().UTC().Add(mw.TimeoutFunc(copyClaims))
	claims["exp"] = expire.Unix()
	claims["orig_iat"] = mw.TimeFunc().Unix()
	tokenString, err := mw.signedString(token)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expire, nil
}

// jwtFromHeader 用来从指定的请求头中提取 token。
func (mw *GinJWTMiddleware) jwtFromHeader(c *gin.Context, key string) (string, error) {
	authHeader := c.GetHeader(key)

	if authHeader == "" {
		return "", MsgErrTokenClientInvalid("请先登录")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if !((len(parts) == 1 && mw.WithoutDefaultTokenHeadName && mw.TokenHeadName == "") ||
		(len(parts) == 2 && parts[0] == mw.TokenHeadName)) {
		return "", MsgErrTokenClientInvalid("登陆凭证无效请重新登录")
	}

	return parts[len(parts)-1], nil
}

// jwtFromQuery 用来从查询参数中提取 token。
func (mw *GinJWTMiddleware) jwtFromQuery(c *gin.Context, key string) (string, error) {
	token := c.Query(key)

	if token == "" {
		return "", MsgErrTokenClientInvalid("请先登录")
	}

	return token, nil
}

// jwtFromCookie 用来从 Cookie 中读取 token。
func (mw *GinJWTMiddleware) jwtFromCookie(c *gin.Context, key string) (string, error) {
	cookie, err := c.Cookie(key)
	if err != nil || cookie == "" {
		return "", MsgErrTokenClientInvalid("请先登录")
	}

	return cookie, nil
}

// jwtFromParam 用来从路由参数中提取 token。
func (mw *GinJWTMiddleware) jwtFromParam(c *gin.Context, key string) (string, error) {
	token := c.Param(key)

	if token == "" {
		return "", MsgErrTokenClientInvalid("请先登录")
	}

	return token, nil
}

// jwtFromForm 用来从表单字段中提取 token。
func (mw *GinJWTMiddleware) jwtFromForm(c *gin.Context, key string) (string, error) {
	token := c.PostForm(key)

	if token == "" {
		return "", MsgErrTokenClientInvalid("请先登录")
	}

	return token, nil
}

// ParseToken 用来按照配置的来源顺序解析请求中的 JWT。
func (mw *GinJWTMiddleware) ParseToken(c *gin.Context) (*jwt.Token, error) {
	var token string
	var err error

	methods := strings.Split(mw.TokenLookup, ",")
	for _, method := range methods {
		if len(token) > 0 {
			break
		}
		parts := strings.Split(strings.TrimSpace(method), ":")
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		switch k {
		case "header":
			token, err = mw.jwtFromHeader(c, v)
		case "query":
			token, err = mw.jwtFromQuery(c, v)
		case "cookie":
			token, err = mw.jwtFromCookie(c, v)
		case "param":
			token, err = mw.jwtFromParam(c, v)
		case "form":
			token, err = mw.jwtFromForm(c, v)
		}
	}

	if err != nil {
		return nil, err
	}

	if mw.KeyFunc != nil {
		return jwt.Parse(token, mw.KeyFunc, mw.ParseOptions...)
	}

	return jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if jwt.GetSigningMethod(mw.SigningAlgorithm) != t.Method {
			return nil, MsgErrTokenClientInvalid("登陆凭证无效请重新登录")
		}
		if mw.usingPublicKeyAlgo() {
			return mw.pubKey, nil
		}

		// 如果有效，保存令牌字符串
		c.Set(CtxKeyJWTToken, token)

		return mw.Key, nil
	}, mw.ParseOptions...)
}

// ParseTokenString 用来解析给定的原始令牌字符串。
func (mw *GinJWTMiddleware) ParseTokenString(token string) (*jwt.Token, error) {
	if mw.KeyFunc != nil {
		return jwt.Parse(token, mw.KeyFunc, mw.ParseOptions...)
	}

	return jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if jwt.GetSigningMethod(mw.SigningAlgorithm) != t.Method {
			return nil, MsgErrTokenClientInvalid("登陆凭证无效请重新登录")
		}
		if mw.usingPublicKeyAlgo() {
			return mw.pubKey, nil
		}

		return mw.Key, nil
	}, mw.ParseOptions...)
}

// unauthorized 用来统一返回未授权响应并中断请求。
func (mw *GinJWTMiddleware) unauthorized(c *gin.Context, code int, message string) {
	c.Header("WWW-Authenticate", "JWT realm="+mw.Realm)
	if !mw.DisabledAbort {
		c.Abort()
	}

	mw.Unauthorized(c, code, message)
}

func (mw *GinJWTMiddleware) GetIdentity(c *gin.Context) any {
	return c.MustGet(mw.IdentityKey)
}

// GetIdentityAs 是泛型版本的 GetIdentity，避免调用方手动类型断言。
func GetIdentityAs[T any](mw *GinJWTMiddleware, c *gin.Context) (T, bool) {
	val := mw.GetIdentity(c)
	t, ok := val.(T)
	return t, ok
}

// ExtractClaims 是 GinJWTMiddleware 的实例方法，从 gin.Context 中取出 JWT Claims。
func (mw *GinJWTMiddleware) ExtractClaims(c *gin.Context) MapClaims {
	return ExtractClaims(c)
}

// ExtractClaims 用来从 gin.Context 中取出 JWT Claims（包级函数）。
func ExtractClaims(c *gin.Context) MapClaims {
	claims, exists := c.Get(CtxKeyJWTPayload)
	if !exists {
		return make(MapClaims)
	}

	return claims.(MapClaims)
}

// ExtractClaimsFromToken 用来从 jwt.Token 中复制 Claims。
func ExtractClaimsFromToken(token *jwt.Token) MapClaims {
	if token == nil {
		return make(MapClaims)
	}

	claims := MapClaims{}
	for key, value := range token.Claims.(jwt.MapClaims) {
		claims[key] = value
	}

	return claims
}

// GetToken 用来从上下文获取解析过的 token 字符串。
func GetToken(c *gin.Context) string {
	token, exists := c.Get(CtxKeyJWTToken)
	if !exists {
		return ""
	}

	return token.(string)
}
