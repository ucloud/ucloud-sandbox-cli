# StartPayUnpaidOrders — 批量支付欠费订单

按订单号批量支付欠费订单，一次最多支付 50 个订单。这是一个写操作接口。

## 请求

### 公共参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Action | string | 是 | 固定为 `StartPayUnpaidOrders` |
| PublicKey | string | 是 | 用户公钥 |
| Signature | string | 是 | 根据公钥、私钥及全部请求参数计算的签名，参见 [认证与调用指南](../SKILL.md) |

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| OrderNos.N | string | 是 | 欠费订单号列表，数组参数，N 为从 0（或 1，文档未明确起始下标）开始的索引，每个订单号作为单独的键值对重复传递（如 `OrderNos.N=xxx`）；最多 50 个 |

## 响应

### 响应字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| RetCode | int | 是 | 0表示成功，非0表示失败 |
| Action | string | 是 | 操作指令名称 |
| Message | string | 否 | RetCode非0时的错误说明 |
| SuccessCount | int | 是 | 支付成功数量 |
| FailureCount | int | 是 | 支付失败数量 |
| Results | PayResult | 否 | 支付结果。原文档类型标注为 `PayResult`（单个对象引用，写作 `*PayResult`），并未标注为数组（如 `array[PayResult]`）；但接口语义是批量支付多个订单，理应对每个订单号都有一条支付结果，这里原文档的类型标注与"批量"语义存在不一致/表述不清的情况，如实注明，不做臆测修正 |

### PayResult 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| OrderNo | string | 否 | 订单号 |
| Success | boolean | 否 | 是否支付成功 |
| Reason | string | 否 | 失败原因（成功时为空） |

## 示例

### 请求示例

```
https://api.ucloud.cn/?Action=StartPayUnpaidOrders
&OrderNos.N=lbShwmqm
&OrderNos.N=grbzlKRi
&OrderNos.N=VBbOUETC
```

### 响应示例

```json
{
  "Action": "StartPayUnpaidOrdersResponse",
  "FailureCount": 9,
  "Results": {},
  "RetCode": 0,
  "SuccessCount": 7
}
```

说明：响应示例中 `Results` 以空对象 `{}` 呈现，未展示 PayResult 的具体字段值，也未展示是否为数组形式，示例本身信息有限，字段值均为文档自动生成的占位数据。

## 文档来源

https://astraflow.ucloud.cn/reference/modelverse/start_pay_unpaid_orders
</content>
