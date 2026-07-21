# GetUMInferRequestLogDetail — 原始日志详情

根据 RequestId 查询单条模型推理请求的原始日志详情，包括请求延迟、状态码、错误信息及 usage 等原始数据。

## 请求

### 公共参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Action | string | 是 | 固定为 `GetUMInferRequestLogDetail` |
| PublicKey | string | 是 | 用户公钥 |
| Signature | string | 是 | 根据公钥、私钥及全部请求参数计算的签名，参见 [认证与调用指南](../SKILL.md) |

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Region | string | 是 | 业务地域，如 cn-wlcb。可先调用 ListUMInferRegions 获取可选地域 |
| ProjectId | string | 否 | 项目ID。不填写为默认项目，子帐号必须填写。请参考 GetProjectList 接口 |
| Zone | string | 是 | 可用区。参见可用区列表 |
| RequestId | string | 是 | 请求ID |

文档中该接口无数组类型参数，无嵌套参数。

## 响应

### 响应字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| RetCode | int | 是 | 0表示成功，非0表示失败 |
| Action | string | 是 | 操作指令名称 |
| Message | string | 否 | RetCode非0时的错误说明 |
| Data | RequestLogDetail | 是 | 请求日志详情 |

### RequestLogDetail 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| RequestId | string | 否 | 请求ID |
| TopOrganizationId | string | 否 | 顶级组织ID |
| OrganizationId | string | 否 | 组织ID |
| ClientIp | string | 否 | 客户端IP |
| Region | string | 否 | 业务地域 |
| StartTime | int | 否 | 请求开始时间，Unix毫秒时间戳 |
| StartTimeReadable | string | 否 | 请求开始时间，可读格式 |
| ModelName | string | 否 | 模型名称 |
| IsStream | boolean | 否 | 是否流式请求 |
| ApiKeyId | string | 否 | API Key ID |
| HttpStatusCode | int | 否 | HTTP状态码 |
| ErrorCode | string | 否 | 错误码 |
| ErrorMessage | string | 否 | 错误信息 |
| IsSuccess | boolean | 否 | 请求是否成功 |
| Latency | int | 否 | 请求总延迟，单位毫秒 |
| FirstTokenLatency | int | 否 | 首Token延迟，单位毫秒 |
| OutputTokenThroughput | float | 否 | 输出Token吞吐 |
| Usage | string | 否 | 模型返回的 usage 原文 JSON |
| Request | string | 否 | 请求原文，本期返回为空 |
| Response | string | 否 | 响应原文，本期返回为空 |
| Extras | string | 否 | 扩展信息，本期返回为空 |

注：Request、Response、Extras 三个字段文档明确标注"本期返回为空"，即当前版本尚未实际生效。

## 示例

请求示例：

```
https://api.ucloud.cn/?Action=GetUMInferRequestLogDetail
&Region=cn-wlcb
&Zone=cn-wlcb-01
&ProjectId=org-xxxx
&RequestId=request-xxxx
```

响应示例：

```json
{
  "Action": "GetUMInferRequestLogDetailResponse",
  "Data": {
    "ApiKeyId": "uminferapikey-xxxx",
    "ClientIp": "10.0.0.1",
    "ErrorCode": "",
    "ErrorMessage": "",
    "Extras": null,
    "FirstTokenLatency": 300,
    "HttpStatusCode": 200,
    "IsStream": true,
    "IsSuccess": true,
    "Latency": 1200,
    "ModelName": "deepseek-r1",
    "OrganizationId": "org-xxxx",
    "OutputTokenThroughput": 42.5,
    "Region": "cn-wlcb",
    "Request": null,
    "RequestId": "request-xxxx",
    "Response": null,
    "StartTime": 1751299200000,
    "StartTimeReadable": "2025-07-01 00:00:00",
    "TopOrganizationId": "org-xxxx",
    "Usage": {
      "completion_tokens": 64,
      "prompt_tokens": 64,
      "total_tokens": 128
    }
  },
  "RetCode": 0
}
```

## 文档来源

https://astraflow.ucloud.cn/reference/modelverse/get_um_infer_request_log_detail
