# GetUFSquareModelDetail — 获取广场模型详情

根据模型主键 Id 查询模型广场中某个模型的详细信息，包括模型的基础信息、能力标签及定价策略。

## 请求

### 公共参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Action | string | 是 | 固定为 `GetUFSquareModelDetail` |
| PublicKey | string | 是 | 用户公钥 |
| Signature | string | 是 | 根据公钥、私钥及全部请求参数计算的签名，参见 [认证与调用指南](../SKILL.md) |

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Region | string | 是 | 地域。参见地域和可用区列表 |
| ProjectId | string | 否 | 项目ID。不填写为默认项目，子帐号必须填写。请参考 GetProjectList 接口 |
| Zone | string | 是 | 可用区。参见可用区列表 |
| Id | string | 是 | 模型主键 |

## 响应

### 响应字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| RetCode | int | 是 | 0表示成功，非0表示失败 |
| Action | string | 是 | 操作指令名称 |
| Message | string | 否 | RetCode非0时的错误说明 |
| SquareModel | SquareModel | 是 | 模型详情 |

### SquareModel 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Manufacturer | string | 否 | 制造商 |
| Id | string | 否 | 主键 |
| Name | string | 否 | 名称 |
| SimpleDescribe | string | 否 | 简要描述 |
| Describe | string | 否 | 详细描述 |
| Language | array[string] | 否 | 语言 |
| MaxModelLen | int | 否 | 模型长度（上下文长度） |
| ModelType | string | 否 | 模型类型 |
| HfUpdateTime | int | 否 | HuggingFace 更新时间 |
| CreateAt | int | 否 | 创建时间 |
| UpdateAt | int | 否 | 更新时间 |
| SupportedCapabilities | array[string] | 否 | 模型能力 |
| Icon | string | 否 | 图标 |
| Pricing | Pricing | 否 | 定价策略 |
| Tiers | array[PriceTier] | 否 | 价格阶梯（有序数组） |

### Pricing 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Completion | float | 否 | 输出定价 |
| Prompt | float | 否 | 提示词定价 |
| Image | float | 否 | 生图定价 |
| Video | string | 否 | 生视频定价 |
| Currency | string | 否 | 币种 |
| Unit | string | 否 | 单位（中文），如"次""百万" |
| UnitEn | string | 否 | 单位（English），如"Time""Million" |

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

原文档给出的请求示例中参数（如 Region、Zone、ProjectId、Id）重复出现且取值为随机占位字符串，判断为文档站点自动生成的通用占位示例，仅摘录如下供参考：

```
https://api.ucloud.cn/?Action=GetUFSquareModelDetail
&Region=cn-zj
&Zone=cn-zj-01
&ProjectId=tPRsGRvy
&Id=lWvIsfSU
```

原文档给出的响应示例同样是占位内容，未包含真实字段：

```json
{
  "Action": "GetUFSquareModelDetailResponse",
  "RetCode": 0,
  "SquareModel": {}
}
```

## 文档来源

https://astraflow.ucloud.cn/reference/modelverse/get_uf_square_model_detail
