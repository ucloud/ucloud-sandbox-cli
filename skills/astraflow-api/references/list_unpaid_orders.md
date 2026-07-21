# ListUnpaidOrders — 查询欠费订单明细

分页查询当前欠费（未支付）的订单明细列表。

## 请求

### 公共参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Action | string | 是 | 固定为 `ListUnpaidOrders` |
| PublicKey | string | 是 | 用户公钥 |
| Signature | string | 是 | 根据公钥、私钥及全部请求参数计算的签名，参见 [认证与调用指南](../SKILL.md) |

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| StartTime | int | 是 | 与 EndTime 同时提供时启用自定义周期查询；EndTime 必须大于 StartTime |
| EndTime | int | 是 | 查询结束时间（Unix 时间戳，秒级）。需与 StartTime 同时提供 |
| Page | int | 是 | 页码，从1开始 |
| PageSize | int | 是 | 每页数量，最小10，最大100 |
| ResourceIds.N | string | 否 | 资源ID数组，支持多选（原文档描述为"key数组"） |
| ModelIds.N | string | 否 | 模型ID数组，支持多选 |
| PricingUnits.N | int | 否 | 计费单元数组，支持多选 |
| OrderTypes.N | int | 否 | 订单类型数组，支持多选 |
| Regions.N | string | 否 | 地域列表，支持多选 |
| PricingSkus.N | string | 否 | 计费单元（SKU）列表，支持多选 |
| ProductCodes.N | string | 否 | 产品类型列表，支持多选；枚举值：`modelverse`、`sandbox` |

说明：本接口的请求参数表中未出现 `ChargeTypes.N`、`OrganizationIds.N` 参数（与其他几个查询接口不同），如实按原文记录。

## 响应

### 响应字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| RetCode | int | 是 | 0表示成功，非0表示失败 |
| Action | string | 是 | 操作指令名称 |
| Message | string | 否 | RetCode非0时的错误说明 |
| Orders | array[UnpaidOrderItem] | 是 | 欠费订单明细列表 |

说明：本接口的顶层响应字段表中未出现 `Page`、`PageSize`、`Total` 等分页字段（与 ListPaidOrders 不同），如实按原文记录，未做推测或补充。

### UnpaidOrderItem 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Region | string | 否 | 地域代码 |
| RegionDisplay | string | 否 | 地域显示名 |
| ProductCode | string | 否 | 产品类型 |
| ProductCodeDisplay | string | 否 | 产品类型显示名 |
| OrderNo | string | 否 | 订单号 |
| SourceOrderNo | string | 否 | 来源订单号 |
| CompanyID | int | 否 | 公司ID |
| OrganizationID | int | 否 | 组织ID |
| OrganizationName | string | 否 | 组织名称 |
| UserEmail | string | 否 | 用户邮箱 |
| ChargeType | int | 否 | 计费类型 |
| ChargeTypeDisplay | string | 否 | 计价方式显示名 |
| Channel | int | 否 | 渠道 |
| Currency | string | 否 | 币种（如：人民币、USD） |
| CurrencyDisplay | string | 否 | 币种显示名 |
| ResourceID | string | 否 | 模型key（原文档字段描述为"模型key"） |
| ResourceType | int | 否 | 资源类型 |
| ResourceTypeDisplay | string | 否 | 资源类型显示名 |
| ModelID | string | 否 | 模型ID |
| ModelName | string | 否 | 模型名称 |
| OrderType | int | 否 | 订单类型 |
| OrderTypeDisplay | string | 否 | 订单类型显示名 |
| PricingSKU | string | 否 | 计费单元（SKU）名称 |
| Quantity | int | 否 | 用量 |
| QuantityDisplay | string | 否 | 用量显示（含单位） |
| PricingUnit | int | 否 | 计费单位（计量单元） |
| PricingUnitDisplay | string | 否 | 计费单位显示名（如：千Token、张、秒） |
| ListPrice | string | 否 | 列表价（原单价） |
| DiscountPrice | string | 否 | 折后价（折后单价） |
| OrderTotalPrice | string | 否 | 订单总额 |
| OriginalPrice | string | 否 | 原价 |
| Status | int | 否 | 订单状态 |
| StatusDisplay | string | 否 | 订单状态显示名 |
| CreateTime | string | 否 | 创建订单时间（Unix 时间戳，秒级）。注意：原文档将该字段类型标注为 string（而非 int），如实照录 |
| StartTime | int | 否 | 开始计费时间（Unix 时间戳，秒级） |
| EndTime | int | 否 | 结束计费时间（Unix 时间戳，秒级） |
| PaidTime | int | 否 | 订单支付时间（Unix 时间戳，秒级） |
| RevocationTime | string | 否 | 撤销时间（Unix 时间戳，秒级）。注意：原文档将该字段类型标注为 string（而非 int），如实照录 |

## 示例

原文档给出的请求示例包含大量与本接口请求参数表不一致的字段（如 `Region`、`Zone`、`ProjectId` 等），判断为文档站点自动生成的通用占位示例，仅摘录与本接口参数相关的部分：

```
https://api.ucloud.cn/?Action=ListUnpaidOrders
&Page=8
&PageSize=5
&ResourceIds.N=KrvxcKCz
&ModelIds.N=jYCYycUM
&PricingUnits.N=8
&OrderTypes.N=6
&StartTime=4
&EndTime=9
&Regions.N=hyVxsESW
&PricingSkus.N=smcMOmBi
&ProductCodes.N=zZgIeVcp
```

原文档给出的响应示例是占位内容，未包含真实字段：

```json
{
  "Action": "ListUnpaidOrdersResponse",
  "Data": {},
  "RetCode": 0
}
```

## 文档来源

https://astraflow.ucloud.cn/reference/modelverse/list_unpaid_orders
</content>
