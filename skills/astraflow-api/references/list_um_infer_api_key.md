# ListUMInferAPIKey — 列表查询APIKey

分页查询当前项目下的 UCloud AI 推理服务（Modelverse/星图平台）API Key 列表，支持按 ModelverseDisabled 等条件过滤。

## 请求

### 公共参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Action | string | 是 | 固定为 `ListUMInferAPIKey` |
| PublicKey | string | 是 | 用户公钥 |
| Signature | string | 是 | 根据公钥、私钥及全部请求参数计算的签名，参见 [认证与调用指南](../SKILL.md) |

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| ProjectId | string | 是 | 项目ID。不填写为默认项目，子帐号必须填写 |
| Offset | int | 否 | 列表起始位置偏移量，默认为0 |
| Limit | int | 否 | 返回数据长度，默认为20，最大100 |
| ModelverseDisabled | int | 否 | 是否modelverse可用。0: 启用 1: 禁用 |
| SandBoxDisabled | int | 否 | 是否沙盒可用（文档注明沙盒控制暂未上线，该参数暂时无效） |

## 响应

### 响应字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| RetCode | int | 是 | 0表示成功，非0表示失败 |
| Action | string | 是 | 操作指令名称 |
| Message | string | 否 | RetCode非0时的错误说明 |
| Data | array[APIKey] | 是 | apikey列表 |

### APIKey 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| IPWhitelist | string | 是 | ip白名单，换行分割的多组ip。支持IPv4和网段，输入后回车生效，最多100个（原文档中该字段被标注为必填，其余字段均为非必填，如实记录） |
| KeyId | string | 否 | 资源ID |
| Name | string | 否 | 名称 |
| ChannelId | int | 否 | 渠道id |
| TopOrganizationId | int | 否 | 公司id |
| OrganizationId | int | 否 | 项目id |
| Status | int | 否 | 状态，1 表示正常 |
| CreateTime | int | 否 | 创建时间 |
| Key | string | 否 | 密钥值 |
| ExpireTime | int | 否 | 过期时间的unix时间戳，-1 表示不过期 |
| ModelverseDisabled | int | 否 | 是否modelverse可用。0: 启用 1: 禁用 |
| SandBoxDisabled | int | 否 | 是否沙盒可用。0: 启用 1: 禁用 |
| DailyLimitAmount | string | 否 | 日限额，单位随用户所在渠道。126渠道单位为美元 |
| DailyUsedAmount | string | 否 | 日已使用额，单位随用户所在渠道。126渠道单位为美元 |
| MonthlyLimitAmount | string | 否 | 月限额，单位随用户所在渠道。126渠道单位为美元 |
| MonthlyUsedAmount | string | 否 | 月已使用额，单位随用户所在渠道。126渠道单位为美元 |
| GrantAllModels | boolean | 否 | 全部模型访问开关，开启不受 GrantedModels 参数控制 |
| GrantedModels | array[string] | 否 | 授权的模型列表，all表示所有模型都有权限 |

## 示例

请求示例：

```
https://api.ucloud.cn/?Action=ListUMInferAPIKey
&Offset=7
&Limit=5
&ProjectId=BwpkilUS
&ModelverseDisabled=5
&SandBoxDisabled=5
```

响应示例：

```json
{
  "Action": "ListUMInferAPIKeyResponse",
  "Data": {},
  "RetCode": 0
}
```

注：以上示例中的参数值均为文档站生成的占位符，非真实取值；响应示例中的 `Data` 展示为空对象 `{}`，与响应字段表中 `Data` 为 APIKey 数组的说明不完全一致，怀疑是文档示例未与字段表同步生成，如实记录以供工程师核实。

## 文档来源

https://astraflow.ucloud.cn/reference/modelverse/list_um_infer_api_key
