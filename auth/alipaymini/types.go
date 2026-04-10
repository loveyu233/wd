package alipaymini

// LoginRequest 表示支付宝小程序登录请求体。
type LoginRequest struct {
	Code          string `json:"code" binding:"required"`
	EncryptedData string `json:"encrypted_data"`
}

// MobilePhoneNumberDecryptionResp 用来承接支付宝手机号解密结果。
type MobilePhoneNumberDecryptionResp struct {
	Code    string `json:"code"`
	Msg     string `json:"msg"`
	SubCode string `json:"subCode"`
	SubMsg  string `json:"subMsg"`
	Mobile  string `json:"mobile"`
}
