# DownloadListPaidOrders — 下载已完成订单明细

生成已完成（已支付）订单明细的 Excel 文件，并返回 US3 预签名下载链接；查询条件与 ListPaidOrders 完全一致，取数范围是左闭右开区间 `[StartTime, EndTime)`，即取开始计费时间大于等于 StartTime 且小于 EndTime 的数据。

## 请求

### 公共参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Action | string | 是 | 固定为 `DownloadListPaidOrders` |
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
| PricingSkus.N | string | 否 | 计费单元（SKU）列表，支持多选 |
| OrderTypes.N | int | 否 | 订单类型数组，支持多选 |
| OrganizationIds.N | int | 否 | 组织ID列表，支持多选 |
| Regions.N | string | 否 | 地域列表，支持多选 |
| ProductCodes.N | string | 否 | 产品类型列表，支持多选；枚举值：`modelverse`、`sandbox` |

文档未标注本接口任何参数为"已废弃"或"暂时无效"。本接口没有 Page/PageSize 分页参数（下载接口不分页，一次性导出全部符合条件的数据）。

## 响应

### 响应字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| RetCode | int | 是 | 0表示成功，非0表示失败 |
| Action | string | 是 | 操作指令名称 |
| Message | string | 否 | RetCode非0时的错误说明 |
| Data | DownloadFileData | 否 | 下载文件信息 |

### DownloadFileData 字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| DownloadURL | string | 否 | 文件下载链接（US3 预签名 URL，请在有效期内立即下载） |
| FileName | string | 否 | 文件名 |
| FileSize | int | 否 | 文件大小（字节） |

说明：本接口为同步返回下载链接的模式，响应中未出现 TaskId、TaskStatus 等异步任务相关字段；下载链接携带在 `Data.DownloadURL` 中。

## 示例

### 请求示例

原文档给出的请求示例包含大量与本接口请求参数表不一致的字段（如 `Region`、`Zone`、`ProjectId` 等，且参数重复出现两次），判断为文档站点自动生成的通用占位示例，仅摘录与本接口参数相关的部分：

```
https://api.ucloud.cn/?Action=DownloadListPaidOrders
&ResourceIds.N=ZaUxZUjD
&ModelIds.N=XJAeQtEB
&PricingUnits.N=8
&OrderTypes.N=2
&StartTime=6
&EndTime=5
&PricingSkus.N=mgqbideM
&OrganizationIds.N=1
&Regions.N=ubMVnzys
&ProductCodes.N=wHjCoFlq
```

### 响应示例

```json
{
  "Action": "DownloadListPaidOrdersResponse",
  "Data": {},
  "RetCode": 0
}
```

（该响应示例中 `Data` 以空对象呈现，未展示 DownloadFileData 的具体字段值，字段值均为文档自动生成的占位数据。）

## 文档来源

https://astraflow.ucloud.cn/reference/modelverse/download_list_paid_orders
