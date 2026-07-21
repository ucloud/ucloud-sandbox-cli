# DeleteUMInferAPIKey — 删除apikey

删除用户在 UCloud AI 推理服务（Modelverse/星图平台）下的一个 API Key。

## 请求

### 公共参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Action | string | 是 | 固定为 `DeleteUMInferAPIKey` |
| PublicKey | string | 是 | 用户公钥 |
| Signature | string | 是 | 根据公钥、私钥及全部请求参数计算的签名，参见 [认证与调用指南](../SKILL.md) |

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| ProjectId | string | 是 | 项目ID。不填写为默认项目，子帐号必须填写。请参考 GetProjectList 接口 |
| KeyId | string | 是 | 要删除的apikey id |

## 响应

### 响应字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| RetCode | int | 是 | 0表示成功，非0表示失败 |
| Action | string | 是 | 操作指令名称 |
| Message | string | 否 | RetCode非0时的错误说明 |
| UminferID | string | 否 | apikey 的资源ID |

文档未提及任何嵌套的 Data Model 对象。

## 示例

请求示例：

```
https://api.ucloud.cn/?Action=DeleteUMInferAPIKey
&KeyId=JVbWEZfa
&ProjectId=rHOibxwm
```

响应示例：

```json
{
  "Action": "DeleteUMInferAPIKeyResponse",
  "RetCode": 0,
  "UminferID": "MsPfcvXw"
}
```

注：以上示例中的参数值均为文档站生成的占位符，非真实取值。

## 文档来源

https://astraflow.ucloud.cn/reference/modelverse/delete_um_infer_api_key
