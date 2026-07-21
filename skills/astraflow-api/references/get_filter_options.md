# GetFilterOptions — 查询订单筛选选项

查询可用于订单筛选的资源、模型、地域等选项列表，供前端渲染订单查询页面的筛选下拉框使用。

## 请求

### 公共参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Action | string | 是 | 固定为 `GetFilterOptions` |
| PublicKey | string | 是 | 用户公钥 |
| Signature | string | 是 | 根据公钥、私钥及全部请求参数计算的签名，参见 [认证与调用指南](../SKILL.md) |

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| ProductCode | string | 否 | 产品类型（单选），枚举值：`modelverse`、`sandbox`；为空时返回所有产品下的选项 |

文档未标注本接口任何参数为"已废弃"或"暂时无效"，也未列出任何数组类型（形如 Xxx.N）的参数。

## 响应

### 响应字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| RetCode | int | 是 | 0表示成功，非0表示失败 |
| Action | string | 是 | 操作指令名称 |
| Message | string | 否 | RetCode非0时的错误说明 |
| ResourceIds | array[FilterOptionString] | 是 | 资源选项列表 |
| Models | array[FilterOptionString] | 否 | 模型选项列表 |
| Dimensions | array[FilterOptionString] | 否 | 账单维度选项列表 |
| PricingUnits | array[FilterOptionInteger] | 否 | 计费单位选项列表 |
| Regions | array[FilterOptionString] | 否 | 地域选项列表 |
| ProductCodes | array[FilterOptionString] | 否 | 产品类型选项列表 |
| Projects | array[FilterOptionInteger] | 否 | 项目选项列表 |
| PricingSKUs | array[FilterOptionString] | 否 | 计费 SKU 选项列表 |
| OrderTypes | array[FilterOptionInteger] | 否 | 订单类型选项列表 |

### FilterOptionString 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Name | string | 否 | 显示名称 |
| Value | string | 否 | 值 |

### FilterOptionInteger 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Name | string | 否 | 显示名称 |
| Value | int | 否 | 值 |

说明：`OrderTypes`、`PricingUnits`、`Projects` 使用 FilterOptionInteger（Value 为 int 类型），其余数组字段使用 FilterOptionString（Value 为 string 类型）。响应示例 JSON 中 `PricingSKUs`、`ProductCodes`、`Projects`、`Regions` 的示例值被渲染为普通字符串占位符而非数组，与字段表格中标注的 `array[...]` 类型不一致，判断为文档站点生成占位示例时的渲染问题，以字段表格中的类型说明为准。

## 示例

### 请求示例

原文档给出的请求示例包含与本接口请求参数表不一致的字段（如 `Region`、`Zone`、`ProjectId`），判断为文档站点自动生成的通用占位示例，仅摘录与本接口参数相关的部分：

```
https://api.ucloud.cn/?Action=GetFilterOptions
&ProductCode=rjAIcPrn
```

### 响应示例

```json
{
  "Action": "GetFilterOptionsResponse",
  "Data": {},
  "Dimensions": [
    {
      "Name": "XtBSHSoJ",
      "Value": "WgZmMQOo"
    }
  ],
  "Models": [
    {
      "Name": "kqJMJdxD",
      "Value": "gxiRtfIw"
    }
  ],
  "OrderTypes": [
    {
      "Name": "WftpPgnQ",
      "Value": {}
    }
  ],
  "PricingSKUs": "qgNjqply",
  "PricingUnits": [
    {
      "Name": "MoqIBNBK",
      "Value": "yJfochKY"
    }
  ],
  "ProductCodes": "nmkSUkVh",
  "Projects": "tEWWkBoj",
  "Regions": "nBOVDUOg",
  "RetCode": 0
}
```

（该响应示例中的字段值均为文档自动生成的随机占位数据，其中 `PricingSKUs`/`ProductCodes`/`Projects`/`Regions` 渲染为字符串、`OrderTypes[0].Value` 渲染为空对象，均与字段表格标注类型不一致，仅供参考不代表真实数据格式。示例中还出现了字段表格未记录的顶层 `Data` 字段，值为空对象。）

## 文档来源

https://astraflow.ucloud.cn/reference/modelverse/get_filter_options
