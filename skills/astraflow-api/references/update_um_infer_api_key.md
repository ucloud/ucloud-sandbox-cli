# UpdateUMInferAPIKey — 更新apikey

更新用户在 UCloud AI 推理服务（Modelverse/星图平台）下已有 API Key 的名称、可用状态、限额、模型授权及 IP 白名单等配置。

## 请求

### 公共参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Action | string | 是 | 固定为 `UpdateUMInferAPIKey` |
| PublicKey | string | 是 | 用户公钥 |
| Signature | string | 是 | 根据公钥、私钥及全部请求参数计算的签名，参见 [认证与调用指南](../SKILL.md) |

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| ProjectId | string | 是 | 项目ID。不填写为默认项目，子帐号必须填写 |
| KeyId | string | 是 | apikey的id |
| Name | string | 否 | 更新的名称 |
| ModelverseDisabled | int | 否 | 是否modelverse可用。0: 启用 1: 禁用 |
| SandBoxDisabled | int | 否 | 是否沙盒可用。0: 启用 1: 禁用 |
| DailyLimitAmount | string | 否 | 日限额，单位随用户所在渠道。126渠道单位为美元 |
| MonthlyLimitAmount | string | 否 | 月限额，单位随用户所在渠道。126渠道单位为美元 |
| GrantAllModels | boolean | 否 | 全部模型访问开关，开启不受 GrantedModels 参数控制，关闭只能访问 GrantedModels 中添加的模型 |
| GrantedModels | string | 否 | 授权模型数组，例如 `["deepseek-ai/DeepSeek-V3.2-Think"]` |
| IPWhitelist | string | 否 | ip白名单，换行分割的多组ip。支持IPv4和网段 |

（该接口页面未如 CreateUMInferAPIKey 那样注明 SandBoxDisabled“暂时无效”，此处按原文档实际描述如实记录，未做推测性补充。）

## 响应

### 响应字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| RetCode | int | 是 | 0表示成功，非0表示失败 |
| Action | string | 是 | 操作指令名称 |
| Message | string | 否 | RetCode非0时的错误说明 |
| UminferID | string | 否 | apikey 的id |

文档未提供嵌套的 Data Model 对象。

## 示例

请求示例：

```
https://api.ucloud.cn/?Action=UpdateUMInferAPIKey
&KeyId=RpSGgoRp
&Name=BcePnSjI
&ProjectId=pAyypIxy
&ModelverseDisabled=XRkPwdKe
&SandBoxDisabled=asNBIzNq
&DailyLimitAmount=vWvOaWEK
&MonthlyLimitAmount=ldtrmGHL
&GrantAllModels=true
&GrantedModels=XtnFKsgy
&IPWhitelist=iBTshYVE
```

响应示例：

```json
{
  "Action": "UpdateUMInferAPIKeyResponse",
  "RetCode": 0,
  "UminferID": "ncvfYGNG"
}
```

注：以上示例中的参数值均为文档站生成的占位符，非真实取值。

## 文档来源

https://astraflow.ucloud.cn/reference/modelverse/update_um_infer_api_key
