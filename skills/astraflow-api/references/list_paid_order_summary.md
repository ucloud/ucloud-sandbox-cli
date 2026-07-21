# ListPaidOrderSummary — 查询已完成订单汇总

按指定维度（资源、模型、计费单元、订单类型等）对已完成（已支付）的订单进行统计汇总，返回聚合后的用量与金额数据。

## 请求

### 公共参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Action | string | 是 | 固定为 `ListPaidOrderSummary` |
| PublicKey | string | 是 | 用户公钥 |
| Signature | string | 是 | 根据公钥、私钥及全部请求参数计算的签名，参见 [认证与调用指南](../SKILL.md) |

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| StartTime | int | 是 | 查询开始时间（Unix 时间戳，秒级） |
| EndTime | int | 是 | 查询结束时间（Unix 时间戳，秒级），必须大于 StartTime |
| ResourceIds.N | string | 否 | 资源ID数组，支持多选 |
| ModelIds.N | string | 否 | 模型ID数组，支持多选 |
| PricingUnits.N | int | 否 | 计费单位数组，支持多选 |
| OrderTypes.N | int | 否 | 订单类型数组，支持多选 |
| ChargeTypes.N | int | 否 | 计费类型数组，支持多选 |
| PricingSkus.N | string | 否 | 计费单元（SKU）列表，支持多选 |
| ProductCodes.N | string | 否 | 产品类型列表，支持多选；枚举值：`modelverse`、`sandbox` |
| Regions.N | string | 否 | 地域列表，支持多选 |
| OrganizationIds.N | int | 否 | 组织ID列表，支持多选 |

## 响应

### 响应字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| RetCode | int | 是 | 0表示成功，非0表示失败 |
| Action | string | 是 | 操作指令名称 |
| Message | string | 否 | RetCode非0时的错误说明 |
| Summaries | array[OrderSummaryItem] | 是 | 已完成订单汇总列表 |

### OrderSummaryItem 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| ResourceId | string | 否 | 资源ID |
| PricingSKU | string | 否 | 计费单元（SKU）名称 |
| ModelID | string | 否 | 模型ID |
| ModelName | string | 否 | 模型名称 |
| PricingUnit | int | 否 | 计费单位（计量单元） |
| PricingUnitName | string | 否 | 计费单位名称 |
| OrderType | int | 否 | 订单类型 |
| OrderTypeDisplay | string | 否 | 订单类型显示名 |
| ChargeType | int | 否 | 计费类型 |
| Status | int | 否 | 订单状态（2=已支付；3=已撤销） |
| StatusDisplay | string | 否 | 订单状态显示名 |
| ListPrice | string | 否 | 列表价（原单价） |
| DiscountPrice | string | 否 | 折后单价 |
| SumQuantity | int | 否 | 总用量（原始值） |
| SumQuantityDisplay | string | 否 | 总用量显示（格式化后的字符串，千token和百万token会进行单位转换） |
| SumOrderPrice | string | 否 | 总订单金额（格式化后的字符串） |
| SumOriginalPrice | string | 否 | 总原价（格式化后的字符串） |
| SumCashAccount | string | 否 | 总现金账户扣款（仅已完成订单返回） |
| SumStarCardAccount | string | 否 | 总星力卡抵扣金额（仅已完成订单返回） |
| SumBonusAccount | string | 否 | 总赠金账户扣款（仅已完成订单返回） |
| SumCoupon | string | 否 | 总代金券抵扣（仅已完成订单返回） |

## 示例

原文档给出的请求示例包含了大量与本接口请求参数表不一致的字段（如 `Region`、`Zone`、`ProjectId` 等），且参数重复出现，判断为文档站点自动生成的通用占位示例，不能作为真实调用参考，仅摘录如下供参考：

```
https://api.ucloud.cn/?Action=ListPaidOrderSummary
&StartTime=1&EndTime=rpsXDfTb
&ResourceIds.N=GtZdlexq&ModelIds.N=dufLMhYc&PricingUnits.N=3&OrderTypes.N=3&ChargeTypes.N=3
&PricingSkus.N=vqcXxipb&ProductCodes.N=TCBxiHlp&Regions.N=QwmZlhIn&OrganizationIds.N=dAhmKEcI
```

原文档给出的响应示例同样是占位内容，未包含真实字段：

```json
{
  "Action": "ListPaidOrderSummaryResponse",
  "Data": {},
  "RetCode": 0
}
```

## 文档来源

https://astraflow.ucloud.cn/reference/modelverse/list_paid_order_summary
</content>
