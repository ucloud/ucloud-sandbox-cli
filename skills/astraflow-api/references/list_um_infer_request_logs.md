# ListUMInferRequestLogs — 日志明细列表

分页查询指定时间范围内的模型推理请求日志明细列表，并返回该查询条件下的汇总统计信息（总请求数、失败请求数）。

## 请求

### 公共参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Action | string | 是 | 固定为 `ListUMInferRequestLogs` |
| PublicKey | string | 是 | 用户公钥 |
| Signature | string | 是 | 根据公钥、私钥及全部请求参数计算的签名，参见 [认证与调用指南](../SKILL.md) |

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Region | string | 是 | 业务地域，如 cn-wlcb。可先调用 ListUMInferRegions 获取可选地域 |
| ProjectId | string | 否 | 项目ID。不填写为默认项目，子帐号必须填写。请参考 GetProjectList 接口 |
| Zone | string | 是 | 可用区。参见可用区列表 |
| StartTime | int | 是 | 查询开始时间，Unix毫秒时间戳 |
| EndTime | int | 是 | 查询结束时间，Unix毫秒时间戳，必须大于等于 StartTime |
| ModelNames.N | string | 否 | 模型名称列表，数组类型，用于过滤 |
| ApiKeyIds.N | string | 否 | API Key ID 列表，数组类型，用于过滤 |
| RequestId | string | 否 | 请求ID，用于精确过滤 |
| Offset | int | 否 | 列表偏移量，默认0 |
| Limit | int | 否 | 返回数量，默认20 |

（已再次核对该接口页面：Region 与 Zone 均为必填参数，StartTime/EndTime 同样为必填。）

## 响应

### 响应字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| RetCode | int | 是 | 0表示成功，非0表示失败 |
| Action | string | 是 | 操作指令名称 |
| Message | string | 否 | RetCode非0时的错误说明 |
| Data | ListUMInferRequestLogsData | 是 | 日志明细列表返回数据 |

### ListUMInferRequestLogsData 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Summary | RequestLogSummary | 否 | 汇总信息 |
| Items | array[RequestLogItem] | 否 | 日志列表 |

### RequestLogSummary 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| TotalRequests | int | 否 | 查询条件命中的总请求数 |
| FailedRequests | int | 否 | 查询条件命中的失败请求数 |

### RequestLogItem 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| RequestId | string | 否 | 请求ID |
| StartTime | int | 否 | 请求开始时间，Unix毫秒时间戳 |
| StartTimeReadable | string | 否 | 请求开始时间，可读格式 |
| Region | string | 否 | 业务地域 |
| ModelName | string | 否 | 模型名称 |
| ApiKeyId | string | 否 | API Key ID |
| ApiKeyName | string | 否 | API Key 名称 |
| Latency | int | 否 | 请求总延迟，单位毫秒 |
| FirstTokenLatency | int | 否 | 首Token延迟，单位毫秒 |
| OutputTokenThroughput | float | 否 | 输出Token吞吐 |
| HttpStatusCode | int | 否 | HTTP状态码 |
| ErrorCode | string | 否 | 错误码 |
| IsSuccess | boolean | 否 | 请求是否成功 |
| TotalTokens | int | 否 | 总Token数 |
| PromptTokens | int | 否 | 输入Token数 |
| CompletionTokens | int | 否 | 输出Token数 |
| CacheHitTokens | int | 否 | 缓存命中Token数 |
| CacheCreationTokens | int | 否 | 缓存写入Token数 |
| CacheCreation5mTokens | int | 否 | 5分钟缓存写入Token数 |
| CacheCreation1hTokens | int | 否 | 1小时缓存写入Token数 |
| HasInferenceLog | boolean | 否 | 是否存在推理日志（原始日志详情，可用于判断能否调用 GetUMInferRequestLogDetail 获取详情） |

## 示例

请求示例：

```
https://api.ucloud.cn/?Action=ListUMInferRequestLogs
&Region=cn-wlcb
&Zone=cn-wlcb-01
&ProjectId=org-xxxx
&StartTime=1751299200000
&EndTime=1751385600000
&ModelNames.n=deepseek-r1
&ApiKeyIds.n=uminferapikey-xxxx
&RequestId=request-xxxx
&Offset=0
&Limit=20
```

响应示例：

```json
{
  "Action": "ListUMInferRequestLogsResponse",
  "Data": {
    "Items": [
      {
        "ApiKeyId": "uminferapikey-xxxx",
        "ApiKeyName": "default",
        "CacheCreation1hTokens": 0,
        "CacheCreation5mTokens": 0,
        "CacheCreationTokens": 0,
        "CacheHitTokens": 0,
        "CompletionTokens": 64,
        "ErrorCode": "",
        "FirstTokenLatency": 300,
        "HasInferenceLog": false,
        "HttpStatusCode": 200,
        "IsSuccess": true,
        "Latency": 1200,
        "ModelName": "deepseek-r1",
        "OutputTokenThroughput": 42.5,
        "PromptTokens": 64,
        "Region": "cn-wlcb",
        "RequestId": "request-xxxx",
        "StartTime": 1751299200000,
        "StartTimeReadable": "2025-07-01 00:00:00",
        "TotalTokens": 128
      }
    ],
    "Summary": {
      "FailedRequests": 0,
      "TotalRequests": 1
    }
  },
  "RetCode": 0,
  "TotalCount": 1
}
```

注：响应示例顶层出现了 `TotalCount` 字段，但该字段未在响应字段表格中列出，怀疑是文档示例的遗留字段（未与响应参数表同步），如实记录以供工程师核实。

## 文档来源

https://astraflow.ucloud.cn/reference/modelverse/list_um_infer_request_logs
