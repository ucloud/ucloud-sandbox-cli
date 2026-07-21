# GetUFSquareModelPrices — 批量查询模型价格

按关键字模糊搜索模型，批量返回匹配到的多个模型的定价信息（价格阶梯及各档位收费明细），并支持分页。

## 请求

### 公共参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Action | string | 是 | 固定为 `GetUFSquareModelPrices` |
| PublicKey | string | 是 | 用户公钥 |
| Signature | string | 是 | 根据公钥、私钥及全部请求参数计算的签名，参见 [认证与调用指南](../SKILL.md) |

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Keyword | string | 否 | 模型名称模糊搜索（例：deepseek-r1） |
| Offset | int | 否 | 列表起始位置偏移量，默认为0 |
| Limit | int | 否 | 返回数据长度，默认为20 |

文档中该接口的请求参数表未列出数组类型（形如 Xxx.N）的参数。

## 响应

### 响应字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| RetCode | int | 是 | 0表示成功，非0表示失败 |
| Action | string | 是 | 操作指令名称 |
| Message | string | 否 | RetCode非0时的错误说明 |
| Models | array[ModelPriceGroup] | 是 | 匹配模型的价格信息 |
| TotalCount | int | 否 | 总条数，用于翻页 |

### ModelPriceGroup 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Manufacturer | string | 是 | 制造商 |
| ModelName | string | 否 | 模型名称 |
| ModelId | string | 否 | 模型ID |
| Tiers | array[PriceTier] | 否 | 价格阶梯（有序数组） |

### PriceTier 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Rates | array[PriceRate] | 是 | 该档位下的收费列表（有序数组） |
| DescriptionEn | string | 是 | 档位描述英文（例如"标准上下文 32k"） |
| Condition | string | 否 | 档位/条件（例如"32k"、"128k"） |
| Description | string | 否 | 档位描述（例如"标准上下文 32k"） |

### PriceRate 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| ChargeItemDescriptionEn | string | 是 | 收费项英文描述 |
| Currency | string | 是 | 货币单位 |
| Unit | string | 是 | 计价单位 |
| UnitEn | string | 是 | 计价单位英文 |
| ChargeItem | string | 否 | 收费项：input/output/thinking/tool... |
| ChargeItemDescription | string | 否 | 收费项描述 |
| Price | string | 否 | 价格 |

## 示例

原文档给出的请求示例中包含了本接口请求参数表中未出现的 Region、Zone 字段（判断为文档站点自动生成的通用占位示例），且 Keyword/Offset/Limit 取值也为随机占位字符串，仅摘录如下供参考：

```
https://api.ucloud.cn/?Action=GetUFSquareModelPrices
&Region=cn-zj
&Zone=cn-zj-01
&Keyword=tUpakpEx
&Offset=2
&Limit=2
```

响应示例：

```json
{
  "Action": "GetUFSquareModelPricesResponse",
  "Models": [
    {
      "ModelId": "OimToelj",
      "ModelName": "vNbcKKTy",
      "Tiers": [
        {
          "Condition": "eNlQgdAu",
          "Description": "sVuQZmlZ",
          "Rates": [
            {
              "ChargeItem": "qoFeTVec",
              "ChargeItemDescription": "jydMPQin",
              "Price": "xgXpKJwx"
            }
          ]
        }
      ]
    }
  ],
  "RetCode": 0,
  "TotalCount": 4
}
```

## 文档来源

https://astraflow.ucloud.cn/reference/modelverse/get_uf_square_model_prices
