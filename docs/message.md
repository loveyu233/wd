# 短信模块

当前业务能力相关模块只保留短信能力，对外统一通过根目录下的 `sms.go` 提供：

- `wd.NewSMS`
- `wd.NewSMSWithAccessKey`
- `wd.NewSMSWithClient`

## 一、初始化

### 1. 通过凭证配置初始化

```go
svc, err := wd.NewSMS(wd.SMSConfig{
    CredentialConfig: new(credential.Config).
        SetType("access_key").
        SetAccessKeyId("access-key-id").
        SetAccessKeySecret("access-key-secret"),
})
```

### 2. 通过 AccessKey 直接初始化

```go
svc, err := wd.NewSMSWithAccessKey("access-key-id", "access-key-secret")
```

### 3. 复用已有客户端

```go
svc, err := wd.NewSMSWithClient(dysmsClient)
```

## 二、发送短信

### 1. 发送单条短信

```go
err := svc.SendSimpleMsg(
    "13800138000",
    "测试签名",
    "SMS_123456789",
    `{"code":"9527"}`,
)
```

### 2. 发送批量短信

```go
err := svc.SendSimpleBatchMsg(
    `["13800138000","13900139000"]`,
    `["测试签名","测试签名"]`,
    "SMS_123456789",
    `[{"code":"9527"},{"code":"9528"}]`,
)
```

## 三、适用场景

适合：

- 登录验证码
- 通知类短信
- 批量业务提醒
