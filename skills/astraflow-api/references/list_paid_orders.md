# ListPaidOrders — 查询已完成订单明细

分页查询已完成（已支付）的订单明细列表。取数范围是左闭右开区间 `[StartTime, EndTime)`，即取开始计费时间大于等于 StartTime 且小于 EndTime 的数据。

## 请求

### 公共参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Action | string | 是 | 固定为 `ListPaidOrders` |
| PublicKey | string | 是 | 用户公钥 |
| Signature | string | 是 | 根据公钥、私钥及全部请求参数计算的签名，参见 [认证与调用指南](../SKILL.md) |

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| StartTime | int | 是 | 查询开始时间（Unix 时间戳，秒级）。需与 EndTime 同时提供，用于启用自定义周期查询；EndTime 必须大于 StartTime |
| EndTime | int | 是 | 查询结束时间（Unix 时间戳，秒级）。需与 StartTime 同时提供 |
| Page | int | 是 | 页码，从1开始 |
| PageSize | int | 是 | 每页数量，最小10，最大100 |
| ResourceIds.N | string | 否 | 资源ID数组，支持多选 |
| ModelIds.N | string | 否 | 模型ID数组，支持多选 |
| PricingUnits.N | int | 否 | 计费单位数组，支持多选 |
| OrderTypes.N | int | 否 | 订单类型数组，支持多选 |
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
| Page | int | 是 | 当前页码 |
| PageSize | int | 否 | 每页数量 |
| Total | int | 否 | 总记录数 |
| Orders | array[OrderItemDetail] | 否 | 订单明细列表 |

### OrderItemDetail 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Region | string | 否 | 地域 |
| RegionDisplay | string | 否 | 地域显示名 |
| ProductCode | string | 否 | 产品类型 |
| ProductCodeDisplay | string | 否 | 产品类型显示名 |
| OrderNo | string | 否 | 订单号 |
| CompanyID | int | 否 | 公司ID |
| OrganizationID | int | 否 | 项目ID（原文档描述为"项目ID"，字段名为 OrganizationID） |
| OrganizationName | string | 否 | 项目名称 |
| UserEmail | string | 否 | 用户邮箱 |
| ChargeType | int | 否 | 计费类型 |
| ChargeTypeDisplay | string | 否 | 计费类型显示名 |
| Channel | int | 否 | 渠道 |
| Currency | string | 否 | 币种（如：人民币、USD） |
| CurrencyDisplay | string | 否 | 币种显示名 |
| ResourceID | string | 否 | 资源ID |
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
| StartTime | int | 否 | 开始计费时间（Unix 时间戳，秒级） |
| EndTime | int | 否 | 结束计费时间（Unix 时间戳，秒级） |
| PaidTime | int | 否 | 支付完成时间（Unix 时间戳，秒级） |
| CashAccount | string | 否 | 现金账户扣款金额 |
| BonusAccount | string | 否 | 赠金账户扣款金额 |
| Coupon | string | 否 | 代金券抵扣金额 |
| StarCardAccount | string | 否 | 星力卡抵扣金额 |
| UnpaidOrderNo | string | 否 | 欠费订单号 |

说明：响应示例 JSON 中还出现了字段说明表未记录的 `CreditAccount`、`PricingItemID`、`ResourceType`、`ResourceTypeDisplay` 四个字段（原文档表格未对这些字段给出类型/必填/描述说明），如实注明，字段含义文档未说明。

## 示例

### 请求示例

原文档给出的请求示例包含大量与本接口请求参数表不一致的字段（如 `Region`、`Zone`、`ProjectId` 等），判断为文档站点自动生成的通用占位示例，仅摘录与本接口参数相关的部分：

```
https://api.ucloud.cn/?Action=ListPaidOrders
&Page=6
&PageSize=9
&ResourceIds.N=AeFSrUzZ
&ModelIds.N=kfUPLHaN
&PricingUnits.N=4
&OrderTypes.N=1
&StartTime=4
&EndTime=5
&PricingSkus.N=ueOIxKoz
&ProductCodes.N=PcpOcCXZ
&Regions.N=IVigjwbh
&OrganizationIds.N=8
```

### 响应示例

```json
{
  "Action": "ListPaidOrdersResponse",
  "Data": {},
  "Orders": [
    {
      "BonusAccount": "AgXuWXUE",
      "CashAccount": "dvsJvKeb",
      "Channel": 5,
      "ChargeType": 8,
      "ChargeTypeDisplay": "BzeIdKsp",
      "CompanyID": 5,
      "Coupon": "bYZilSNV",
      "CreditAccount": "ddqZVbbL",
      "Currency": "OcSKWFHV",
      "CurrencyDisplay": "JfLOksHY",
      "DiscountPrice": "yqGjZsVq",
      "EndTime": 1,
      "ListPrice": "QoIypZbG",
      "ModelID": "KdJndPFe",
      "ModelName": "PuNopUfo",
      "OrderNo": "dPfZKSHW",
      "OrderTotalPrice": "dKQVVWHo",
      "OrderType": 6,
      "OrderTypeDisplay": "mhpQYHbO",
      "OrganizationID": 5,
      "OrganizationName": "TsIIjvmL",
      "OriginalPrice": "UElWQRCf",
      "PaidTime": 9,
      "PricingItemID": 5,
      "PricingSKU": "kGHeOOsB",
      "PricingUnit": 5,
      "PricingUnitDisplay": "mBfpOMCm",
      "Quantity": 9,
      "QuantityDisplay": "nnnhBEHV",
      "ResourceID": "nhULypVv",
      "ResourceType": 2,
      "ResourceTypeDisplay": "QGWVsrLb",
      "StartTime": 3,
      "Status": 4,
      "StatusDisplay": "AcQIUtuY",
      "UnpaidOrderNo": "qBHrxRBp",
      "UserEmail": "OOFVHQds"
    }
  ],
  "PageSize": 3,
  "RetCode": 0,
  "Total": 6
}
```

（该响应示例中的字段值均为文档自动生成的随机占位字符串，不代表真实数据格式）

## 文档来源

https://astraflow.ucloud.cn/reference/modelverse/list_paid_orders
</content>
