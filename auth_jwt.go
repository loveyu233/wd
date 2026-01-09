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

var InsGinJWTMiddleware *GinJWTMiddleware

// MapClaims 使用 map[string]interface{} 进行 JSON 解码的类型
// 如果你没有提供一个，这是默认的声明类型
type MapClaims map[string]interface{}

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

	// 回调函数，应执行已认证用户的授权。
	// 仅在身份验证成功后调用。成功时必须返回 true，失败时返回 false。
	// 可选，默认为成功。
	Authorizator func(data interface{}, c *gin.Context) bool

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

	// 设置身份处理函数，默认使用 IdentityKey
	IdentityHandler func(c *gin.Context) interface{}

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

	// 用于非对称算法的私钥文件
	PrivKeyFile string

	// 用于非对称算法的私钥字节
	//
	// 注意：如果两者都设置，PrivKeyFile 优先于 PrivKeyBytes
	PrivKeyBytes []byte

	// 用于非对称算法的公钥文件
	PubKeyFile string

	// 私钥密码短语
	PrivateKeyPassphrase string

	// 用于非对称算法的公钥字节。
	//
	// 注意：如果两者都设置，PubKeyFile 优先于 PubKeyBytes
	PubKeyBytes []byte

	// 私钥
	privKey *rsa.PrivateKey

	// 公钥
	pubKey *rsa.PublicKey

	// 可选地将令牌作为 cookie 返回
	SendCookie bool

	// cookie 有效的持续时间。可选，默认等于 Timeout 值。
	CookieMaxAge time.Duration

	// 允许不安全的 cookie 用于 http 开发
	SecureCookie bool

	// 允许客户端访问 cookie 用于开发
	CookieHTTPOnly bool

	// 允许 cookie 域更改用于开发
	CookieDomain string

	// SendAuthorization 允许为每个请求返回授权头
	SendAuthorization bool

	// 禁用上下文的 abort()。
	DisabledAbort bool

	// CookieName 允许更改 cookie 名称用于开发
	CookieName string

	// CookieSameSite 允许使用 http.SameSite cookie 参数
	CookieSameSite http.SameSite

	// ParseOptions 允许修改 jwt 的解析方法
	ParseOptions []jwt.ParserOption
}

// InitGinJWTMiddleware 用来初始化 GinJWTMiddleware 并执行必要检查。
func InitGinJWTMiddleware(m *GinJWTMiddleware) error {
	if err := m.MiddlewareInit(); err != nil {
		return err
	}
	InsGinJWTMiddleware = m
	return nil
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
	if mw.PrivKeyFile == "" {
		keyData = mw.PrivKeyBytes
	} else {
		filecontent, err := os.ReadFile(mw.PrivKeyFile)
		if err != nil {
			return MsgErrTokenServerInvalid("私钥文件读取失败")
		}
		keyData = filecontent
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(keyData)
	if err != nil {
		return MsgErrTokenServerInvalid("私钥无效")
	}
	mw.privKey = key
	return nil
}

// publicKey 用来读取并解析 RSA 公钥。
func (mw *GinJWTMiddleware) publicKey() error {
	var keyData []byte
	if mw.PubKeyFile == "" {
		keyData = mw.PubKeyBytes
	} else {
		filecontent, err := os.ReadFile(mw.PubKeyFile)
		if err != nil {
			return MsgErrTokenServerInvalid("公钥文件读取失败")
		}
		keyData = filecontent
	}

	key, err := jwt.ParseRSAPublicKeyFromPEM(keyData)
	if err != nil {
		return MsgErrTokenServerInvalid("公钥无效")
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

// MiddlewareInit 用来填充中间件的默认配置与依赖。
func (mw *GinJWTMiddleware) MiddlewareInit() error {
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

	if mw.Authorizator == nil {
		mw.Authorizator = func(data interface{}, c *gin.Context) bool {
			return true
		}
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
				ResponseError(c, MsgErrInvalidParam("登录失败"))
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
				ResponseError(c, ErrTokenServerInvalid)
			}
		}
	}

	if mw.IdentityKey == "" {
		mw.IdentityKey = "identity"
	}

	if mw.IdentityHandler == nil {
		mw.IdentityHandler = func(c *gin.Context) interface{} {
			claims := ExtractClaims(c)
			return claims[mw.IdentityKey]
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

	if mw.CookieMaxAge == 0 {
		mw.CookieMaxAge = mw.Timeout
	}

	if mw.CookieName == "" {
		mw.CookieName = "jwt"
	}

	// 如果设置了 KeyFunc，则绕过其他密钥设置
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
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			appErr = MsgErrTokenServerInvalid("令牌已过期")
		case errors.Is(err, jwt.ErrTokenMalformed):
			appErr = MsgErrTokenServerInvalid("令牌格式错误")
		default:
			appErr = MsgErrTokenServerInvalid("令牌异常")
		}
		mw.unauthorized(c, appErr.Code, mw.HTTPStatusMessageFunc(appErr, c))
		return
	}

	switch v := claims["exp"].(type) {
	case nil:
		appErr := MsgErrTokenServerInvalid("令牌异常")
		mw.unauthorized(c, appErr.Code, mw.HTTPStatusMessageFunc(appErr, c))
		return
	case float64:
		if int64(v) < mw.TimeFunc().Unix() {
			appErr := MsgErrTokenServerInvalid("令牌已过期")
			mw.unauthorized(c, appErr.Code, mw.HTTPStatusMessageFunc(appErr, c))
			return
		}
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			appErr := MsgErrTokenServerInvalid("令牌格式错误")
			mw.unauthorized(c, appErr.Code, mw.HTTPStatusMessageFunc(appErr, c))
			return
		}
		if n < mw.TimeFunc().Unix() {
			appErr := MsgErrTokenServerInvalid("令牌已过期")
			mw.unauthorized(c, appErr.Code, mw.HTTPStatusMessageFunc(appErr, c))
			return
		}
	default:
		appErr := MsgErrTokenServerInvalid("令牌格式错误")
		mw.unauthorized(c, appErr.Code, mw.HTTPStatusMessageFunc(appErr, c))
		return
	}

	c.Set("JWT_PAYLOAD", claims)
	identity := mw.IdentityHandler(c)

	if identity != nil {
		c.Set(mw.IdentityKey, identity)
	}

	if !mw.Authorizator(identity, c) {
		mw.unauthorized(c, ErrForbiddenAuth.Code, mw.HTTPStatusMessageFunc(ErrForbiddenAuth, c))
		return
	}

	c.Next()
}

// GetClaimsFromJWT 用来解析请求中的 JWT 并返回 Claims。
func (mw *GinJWTMiddleware) GetClaimsFromJWT(c *gin.Context) (MapClaims, error) {
	token, err := mw.ParseToken(c)
	if err != nil {
		return nil, err
	}

	if mw.SendAuthorization {
		if v, ok := c.Get("JWT_TOKEN"); ok {
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

		data, err := mw.Authenticator(c)
		if err != nil {
			ResponseError(c, err)
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
			appError := MsgErrTokenServerInvalid("创建令牌失败")
			mw.unauthorized(c, appError.Code, mw.HTTPStatusMessageFunc(appError, c))
			return
		}

		// 设置 cookie
		if mw.SendCookie {
			expireCookie := mw.TimeFunc().Add(mw.CookieMaxAge)
			maxage := int(expireCookie.Unix() - mw.TimeFunc().Unix())
			c.SetSameSite(mw.CookieSameSite)
			c.SetCookie(mw.CookieName, tokenString, maxage, "/", mw.CookieDomain, mw.SecureCookie, mw.CookieHTTPOnly)
		}

		mw.LoginResponse(c, http.StatusOK, tokenString, expire)
	}
}

// LogoutHandler 用来清理客户端 cookie 并返回退出响应。
func (mw *GinJWTMiddleware) LogoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 删除认证 cookie
		if mw.SendCookie {
			c.SetSameSite(mw.CookieSameSite)
			c.SetCookie(mw.CookieName, "", -1, "/", mw.CookieDomain, mw.SecureCookie, mw.CookieHTTPOnly)
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
			appErr := MsgErrTokenServerInvalid(err.Error())
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
	if mw.SendCookie {
		expireCookie := mw.TimeFunc().Add(mw.CookieMaxAge)
		maxage := int(expireCookie.Unix() - time.Now().Unix())
		c.SetSameSite(mw.CookieSameSite)
		c.SetCookie(mw.CookieName, tokenString, maxage, "/", mw.CookieDomain, mw.SecureCookie, mw.CookieHTTPOnly)
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
		return nil, MsgErrTokenClientInvalid("令牌无效")
	}

	claims := token.Claims.(jwt.MapClaims)

	origIat := int64(claims["orig_iat"].(float64))

	if origIat < mw.TimeFunc().Add(-mw.MaxRefresh).Unix() {
		return nil, MsgErrTokenClientInvalid("令牌已过期")
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
		return "", MsgErrTokenClientInvalid("认证头为空")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if !((len(parts) == 1 && mw.WithoutDefaultTokenHeadName && mw.TokenHeadName == "") ||
		(len(parts) == 2 && parts[0] == mw.TokenHeadName)) {
		return "", MsgErrTokenClientInvalid("认证头无效")
	}

	return parts[len(parts)-1], nil
}

// jwtFromQuery 用来从查询参数中提取 token。
func (mw *GinJWTMiddleware) jwtFromQuery(c *gin.Context, key string) (string, error) {
	token := c.Query(key)

	if token == "" {
		return "", MsgErrTokenClientInvalid("令牌为空")
	}

	return token, nil
}

// jwtFromCookie 用来从 Cookie 中读取 token。
func (mw *GinJWTMiddleware) jwtFromCookie(c *gin.Context, key string) (string, error) {
	cookie, err := c.Cookie(key)
	if err != nil {
		return "", MsgErrTokenClientInvalid("令牌为空")
	}

	if cookie == "" {
		return "", MsgErrTokenClientInvalid("令牌为空")
	}

	return cookie, nil
}

// jwtFromParam 用来从路由参数中提取 token。
func (mw *GinJWTMiddleware) jwtFromParam(c *gin.Context, key string) (string, error) {
	token := c.Param(key)

	if token == "" {
		return "", MsgErrTokenClientInvalid("令牌为空")
	}

	return token, nil
}

// jwtFromForm 用来从表单字段中提取 token。
func (mw *GinJWTMiddleware) jwtFromForm(c *gin.Context, key string) (string, error) {
	token := c.PostForm(key)

	if token == "" {
		return "", MsgErrTokenClientInvalid("令牌为空")
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
			return nil, MsgErrTokenClientInvalid("无效的签名算法")
		}
		if mw.usingPublicKeyAlgo() {
			return mw.pubKey, nil
		}

		// 如果有效，保存令牌字符串
		c.Set("JWT_TOKEN", token)

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
			return nil, MsgErrTokenClientInvalid("无效的签名算法")
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

// ExtractClaims 用来从 gin.Context 中取出 JWT Claims。
func ExtractClaims(c *gin.Context) MapClaims {
	claims, exists := c.Get("JWT_PAYLOAD")
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
	token, exists := c.Get("JWT_TOKEN")
	if !exists {
		return ""
	}

	return token.(string)
}
