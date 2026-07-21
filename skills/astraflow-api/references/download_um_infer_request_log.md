# DownloadUMInferRequestLog — 导出推理请求日志

导出指定时间范围内的模型推理请求日志。单次导出时间范围最长 30 天，最多导出 2000 万条日志；同一 TopOrganizationID 同一时间仅允许 1 个导出任务在执行，已有任务执行中时请稍后重试。

## 请求

### 公共参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Action | string | 是 | 固定为 `DownloadUMInferRequestLog` |
| PublicKey | string | 是 | 用户公钥 |
| Signature | string | 是 | 根据公钥、私钥及全部请求参数计算的签名，参见 [认证与调用指南](../SKILL.md) |

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Region | string | 是 | 业务地域，如 cn-wlcb。可先调用 ListUMInferRegions 获取可选地域 |
| ProjectId | string | 否 | 项目ID。不填写为默认项目，子帐号必须填写。请参考 GetProjectList 接口 |
| Zone | string | 是 | 可用区。参见可用区列表 |
| StartTime | int | 是 | 导出开始时间，Unix 毫秒时间戳 |
| EndTime | int | 是 | 导出结束时间，Unix 毫秒时间戳，最长支持 30 天范围 |
| Email | string | 是 | 接收导出结果的邮箱地址 |
| ModelNames.N | string | 否 | 模型名称列表，数组类型，用于过滤 |
| ApiKeyIds.N | string | 否 | API Key ID 列表，数组类型，用于过滤 |
| RequestId | string | 否 | 请求 ID，用于精确过滤 |

## 响应

### 响应字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| RetCode | int | 是 | 0表示成功，非0表示失败 |
| Action | string | 是 | 操作指令名称 |
| Message | string | 否 | RetCode非0时的错误说明 |
| TaskId | string | 否 | 导出任务ID |
| TotalCount | int | 否 | 本次导出查询命中的日志行数 |

文档未包含嵌套 Data Model 对象或数组类型的响应字段。

## 示例

请求示例：

```
https://api.ucloud.cn/?Action=DownloadUMInferRequestLog
&Region=cn-wlcb
&Zone=cn-wlcb-01
&ProjectId=org-xxx
&StartTime=1751299200000
&EndTime=1751385600000
&Email=ops@example.com
&ModelNames.n=deepseek-r1
&ApiKeyIds.n=uminferapikey-xxxx
&RequestId=request-xxxx
```

响应示例：

```json
{
  "Action": "DownloadUMInferRequestLogResponse",
  "RetCode": 0,
  "TaskId": "task-xxxx",
  "TotalCount": 1024
}
```

## 文档来源

https://astraflow.ucloud.cn/reference/modelverse/download_um_infer_request_log
