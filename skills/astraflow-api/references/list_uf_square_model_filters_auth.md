# ListUFSquareModelFiltersAuth — 查询模型广场过滤条件

登录状态下获取模型广场过滤器中可选的过滤条件内容。

## 请求

### 公共参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Action | string | 是 | 固定为 `ListUFSquareModelFiltersAuth` |
| PublicKey | string | 是 | 用户公钥 |
| Signature | string | 是 | 根据公钥、私钥及全部请求参数计算的签名，参见 [认证与调用指南](../SKILL.md) |

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Region | string | 是 | 地域。参见地域和可用区列表 |
| ProjectId | string | 否 | 项目ID。不填写为默认项目，子帐号必须填写。请参考 GetProjectList 接口 |
| Zone | string | 是 | 可用区。参见可用区列表 |

文档中该接口的请求参数表未列出数组类型（形如 Xxx.N）的参数。

## 响应

### 响应字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| RetCode | int | 是 | 0表示成功，非0表示失败 |
| Action | string | 是 | 操作指令名称 |
| Message | string | 否 | RetCode非0时的错误说明 |

文档未说明该接口除 RetCode / Action / Message 之外是否还返回具体的过滤条件数据字段（如制造商列表、模型类型列表、语言列表等）；已针对该接口页面反复抓取核实，文档的响应字段表格与响应示例中均只出现了上述三个通用字段，未提供任何嵌套 Data Model 或数组字段的说明，如实记录为文档未说明，请以实际调用结果为准。

## 示例

原文档给出的请求示例中 Region、Zone、ProjectId 参数重复出现且取值为随机占位字符串，判断为文档站点自动生成的通用占位示例，仅摘录如下供参考：

```
https://api.ucloud.cn/?Action=ListUFSquareModelFiltersAuth
&Region=cn-zj
&Zone=cn-zj-01
&ProjectId=wgyyjtpY
```

响应示例：

```json
{
  "Action": "ListUFSquareModelFiltersAuthResponse",
  "RetCode": 0
}
```

## 文档来源

https://astraflow.ucloud.cn/reference/modelverse/list_uf_square_model_filters_auth
