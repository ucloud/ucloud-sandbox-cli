# GetOrderAmount — 查询订单汇总统计

查询指定条件下订单的金额汇总及数量统计，用于展示订单总额、已支付/待支付金额及各类账户抵扣金额等汇总数据。

## 请求

### 公共参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Action | string | 是 | 固定为 `GetOrderAmount` |
| PublicKey | string | 是 | 用户公钥 |
| Signature | string | 是 | 根据公钥、私钥及全部请求参数计算的签名，参见 [认证与调用指南](../SKILL.md) |

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| StartTime | int | 是 | 查询开始时间（Unix 时间戳，秒级）。需与 EndTime 同时提供，最大查询跨度 366 天 |
| EndTime | int | 是 | 查询结束时间（Unix 时间戳，秒级）。需与 StartTime 同时提供 |
| ResourceIds.N | string | 否 | 资源ID数组，支持多选 |
| ModelIds.N | string | 否 | 模型ID数组，支持多选 |
| PricingUnits.N | int | 否 | 计费单位数组，支持多选 |
| PricingSkus.N | string | 否 | 计费单元（SKU）列表，支持多选 |
| ProductCodes.N | string | 否 | 产品类型列表，支持多选；枚举值：`modelverse`、`sandbox` |
| OrderTypes.N | int | 否 | 订单类型数组，支持多选 |
| Regions.N | string | 否 | 地域列表，支持多选 |
| OrganizationIds.N | string | 否 | 组织ID列表，支持多选（注意：本接口该字段类型文档标注为 string，与其他几个接口中同名字段的 int 类型不一致，如实按本接口页面记录） |

文档未标注本接口任何参数为"已废弃"或"暂时无效"。本接口没有 Page/PageSize 分页参数，返回的是聚合统计值而非明细列表。

## 响应

### 响应字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| RetCode | int | 是 | 0表示成功，非0表示失败 |
| Action | string | 是 | 操作指令名称 |
| Message | string | 否 | RetCode非0时的错误说明 |
| TotalOrderAmount | string | 是 | 订单总额（所有订单的总金额） |
| PaidAmount | string | 否 | 已支付金额 |
| UnpaidAmount | string | 否 | 待支付金额 |
| CashAmount | string | 否 | 现金账户总金额 |
| BonusAmount | string | 否 | 赠金账户总金额 |
| CouponAmount | string | 否 | 代金券抵扣总额 |
| StarCardAmount | string | 否 | 星力卡抵扣总金额 |
| OrderCount | int | 否 | 订单总数 |
| PaidCount | int | 否 | 已支付订单数 |
| UnpaidCount | int | 否 | 待支付订单数量 |

文档未提供任何嵌套 Data Model；本接口响应中未出现 TaskId、DownloadUrl 等异步任务/下载链接相关字段（本接口本身不涉及文件导出）。

说明：响应示例 JSON 中的顶层 key 为 `Action`、`BonusAmount`、`CashAmount`、`CouponAmount`、`Data`、`OrderCount`、`PaidAmount`、`PaidCount`、`RetCode`、`StarCardAmount`、`UnpaidAmount`、`UnpaidCount`——其中示例未包含字段表格中标注为"必填"的 `TotalOrderAmount` 字段，也未包含 `Message` 字段（后者为非必填字段，成功响应时不返回属正常现象，如实注明）；示例中出现了字段表格未记录的顶层 `Data` 字段，值为空对象，含义文档未说明。

## 示例

### 请求示例

原文档给出的请求示例包含大量与本接口请求参数表不一致的字段（如 `Region`、`Zone`、`ProjectId` 等），且 `Regions.N`、`OrganizationIds.N` 重复出现两次，判断为文档站点自动生成的通用占位示例，仅摘录与本接口参数相关的部分：

```
https://api.ucloud.cn/?Action=GetOrderAmount
&ResourceIds.N=YUhHIvbF
&ModelIds.N=YLZoVxpX
&PricingUnits.N=4
&OrderTypes.N=6
&StartTime=2
&EndTime=7
&PricingSkus.N=ZvSMEZfo
&ProductCodes.N=GQiNWIED
&Regions.N=dOnMCcfY
&OrganizationIds.N=HFhMPcol
&Regions.N=ainUEQqh
&OrganizationIds.N=BZLMCvrq
```

### 响应示例

```json
{
  "Action": "GetOrderAmountResponse",
  "BonusAmount": "wqQkjckV",
  "CashAmount": "Hsbcqacs",
  "CouponAmount": "ZRcRUuyv",
  "Data": {},
  "OrderCount": 7,
  "PaidAmount": "SiiWorjy",
  "PaidCount": 8,
  "RetCode": 0,
  "StarCardAmount": "fEuXwTFD",
  "UnpaidAmount": "TaOEmaIs",
  "UnpaidCount": 5
}
```

（该响应示例中的字段值均为文档自动生成的随机占位数据，不代表真实数据格式；如上所述，示例中缺少字段表格标注为必填的 `TotalOrderAmount` 字段。）

## 文档来源

https://astraflow.ucloud.cn/reference/modelverse/get_order_amount
