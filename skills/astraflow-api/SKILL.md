---
name: astraflow-api
description: 当用户需要调用星图平台（AstraFlow / UModelVerse）的管理API时使用，包括创建或管理推理API Key、查询账单与订单、查询模型广场数据、导出或查询推理请求日志等场景。提供UCloud通用签名认证的完整准备工作（如何获取PublicKey/PrivateKey、如何计算Signature、如何确定Region与ProjectId），并在 references/ 目录下收录了全部21个已知管理API的详细请求/响应参数规范。
---

# 星图平台管理API调用指南

星图（AstraFlow）是UCloud面向企业的专属AI开发平台，其下的模型服务子平台叫 UModelVerse。本技能只覆盖**管理类API**（管理API Key、账单订单、模型广场查询、推理日志查询等），也就是文档里位于 `https://astraflow.ucloud.cn/reference/modelverse/` 下的那一批接口。

## 先弄清楚：管理API ≠ 模型调用API

星图平台实际上有两套完全不同的接口体系，不要混淆：

1. **模型调用API**（OpenAI/Gemini兼容接口，例如 `POST https://api.modelverse.cn/v1/chat/completions`）：用来真正调用大模型做推理。鉴权方式很简单，只需要一个API Key放进请求头 `Authorization: Bearer {api_key}` 即可，不涉及签名计算。这个API Key可以在控制台 `https://console.ucloud.cn/modelverse/experience/api-keys` 创建。
2. **管理API**（本技能覆盖的对象，例如 `CreateUMInferAPIKey`、`ListUMInferRequestLogs`）：用来管理账号下的资源，比如创建/删除上面那种推理用的API Key、查订单账单、查模型广场数据、查推理日志等。这套接口走的是UCloud通用API体系的**PublicKey/PrivateKey签名认证**，和UCloud云主机、云硬盘等传统产品线的API是同一套认证机制，跟"模型调用API"的Bearer Token认证完全不是一回事。

后面所有内容都是针对第2种（管理API）。

## 调用前必须拿到的4个参数

调用任何一个星图管理API，下面4个参数都是必填的：

| 参数 | 说明 |
| --- | --- |
| `Region` | 业务地域，例如 `cn-wlcb`。决定请求路由到哪个地域的服务 |
| `ProjectId` | 项目ID。账号下有多个项目时用来区分资源归属；不填默认使用主账号的默认项目，但**子账号必须填写** |
| `PublicKey` | UCloud账号的公钥，是签名计算和请求本身都要用到的参数 |
| `PrivateKey` | UCloud账号的私钥，只参与本地签名计算，**不会**出现在请求参数里，也不会发送给服务端 |

`PublicKey`/`PrivateKey` 在UCloud控制台的"API密钥管理"页面获取（账号级别的密钥，不是上面提到的模型调用API Key）。

### 向用户索取这4个参数的方式

- 如果用户没有提供，直接询问用户要，或者告诉用户可以通过环境变量提供，推荐使用下面这组变量名（当前项目内没有既定约定时的建议命名，实际命名可以按用户/项目习惯调整）：

```bash
export ASTRAFLOW_PUBLIC_KEY="<public-key>"
export ASTRAFLOW_PRIVATE_KEY="<private-key>"
export ASTRAFLOW_REGION="cn-wlcb"
export ASTRAFLOW_PROJECT_ID="<project-id>"
```

- `PrivateKey` 是高度敏感信息：只用于本地计算签名，绝不能出现在回显给用户的内容、日志、生成的代码仓库文件或提交记录里；执行涉及它的脚本前确认没有开启 `set -x` 之类会打印变量值的调试开关。
- 如果用户不确定该选哪个 `Region`，参见下面"确定Region"一节。
- 如果用户不确定该用哪个 `ProjectId`，参见下面"确定ProjectId"一节。

## 请求的基本格式

- 请求地址：`https://api.ucloud.cn`（UCloud统一网关，具体由请求里的 `Action` 参数决定实际路由到哪个后端服务）
- 支持 `GET` 或 `POST`，`POST` 时使用 `Content-Type: application/json`
- 每个请求都必须带上下面3个"公共参数"，这是所有UCloud API的通用要求：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `Action` | string | 是 | API指令名称，例如 `CreateUMInferAPIKey`、`ListUMInferAPIKey` |
| `PublicKey` | string | 是 | 用户公钥 |
| `Signature` | string | 是 | 根据公钥、私钥及本次请求全部参数计算出的签名，见下一节 |

除了公共参数外，还要加上该 `Action` 自己的业务参数（比如 `Region`、`ProjectId`，以及各接口特有的参数）。

- 响应是JSON，公共响应字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `RetCode` | int | 0表示成功，非0表示失败 |
| `Action` | string | 对应请求的指令名称 |
| `Message` | string | `RetCode` 非0时，说明失败原因 |

调用失败时先看 `RetCode`/`Message`，常见问题：`170` 缺少签名、`171` 签名错误、`160`/`161` 缺少或不存在 `Action`、`292` 项目不存在、`294` 访问IP被拒绝。签名相关报错优先检查参数排序、是否遗漏了某个参数、`PrivateKey` 是否正确。

## Signature 计算方法

`Signature` 由 `PublicKey`、`PrivateKey` 以及本次请求的**全部参数**（不含 `Signature` 自身）计算得出，算法是SHA1，步骤如下：

1. 收集本次请求会发送的全部参数（公共参数 `Action`、`PublicKey`，以及所有业务参数，比如 `Region`、`ProjectId`、该接口特有的参数），但**不包括** `Signature` 本身。
2. 把这些参数按参数名做**升序排序**（ASCII顺序）。
3. 按排序后的顺序，把每个参数的"参数名"和"参数值"依次拼接成一个字符串，不做任何HTTP转义（不要做URL encode）。
4. 在拼接好的字符串**末尾**追加 `PrivateKey`。
5. 对整个字符串做SHA1哈希，得到的十六进制字符串（小写）就是 `Signature` 的值。

编码细节：

- 布尔值编码成字面量 `true` / `false`。
- 浮点数如果小数部分是0，只保留整数部分（例如 `42.0` 要写成 `42`）。
- 浮点数不能用科学计数法表示。
- 数组类型参数（例如 `ModelNames.N`）按其展开后的实际键名参与排序和拼接，比如 `ModelNames.0`、`ModelNames.1`。

示例（来自官方文档，用于校验实现是否正确）：

- `PublicKey`: `ucloudsomeone@example.com1296235120854146120`
- `PrivateKey`: `46f09bb9fab4f12dfc160dae12273d5332b5debe`
- 请求参数：`Action=DescribeUHostInstance`、`Region=cn-bj2`、`Limit=10`
- 拼接后的待签名字符串：

```
ActionDescribeUHostInstanceLimit10PublicKeyucloudsomeone@example.com1296235120854146120Regioncn-bj246f09bb9fab4f12dfc160dae12273d5332b5debe
```

- 对上面字符串做SHA1，得到 `Signature`：`cba5cf5ec4d4233d206b1b54951e3787350a642f`

用shell快速验证的写法（仅用于本地校验签名算法实现，实际调用时按参数升序拼接对应接口的真实参数）：

```bash
printf '%s' 'ActionDescribeUHostInstanceLimit10PublicKeyucloudsomeone@example.com1296235120854146120Regioncn-bj246f09bb9fab4f12dfc160dae12273d5332b5debe' | sha1sum
```

计算出Signature后，把它作为一个普通参数加进最终请求里，和其余参数一起发送。

### 请求示例

```bash
curl -X POST \
  https://api.ucloud.cn \
  -H 'Content-Type: application/json' \
  -d '{
      "Action"     : "DescribeUHostInstance",
      "Limit"      : 10,
      "PublicKey"  : "ucloudsomeone@example.com1296235120854146120",
      "Region"     : "cn-bj2",
      "Signature"  : "cba5cf5ec4d4233d206b1b54951e3787350a642f"
  }'
```

星图管理API的调用方式完全一样，只是把 `Action` 换成星图自己的指令（如 `CreateUMInferAPIKey`），把业务参数换成该指令要求的参数。

## 确定 Region

UCloud全平台的地域列表（共33个地域）：

| 地域短ID | 地域名称 |
| --- | --- |
| cn-bj1 | 华北（北京） |
| cn-bj2 | 华北（北京2） |
| cn-wlcb | 华北（乌兰察布） |
| cn-wlcb2 | 华北（乌兰察布2） |
| cn-sh2 | 华东（上海2） |
| cn-jx | 华东（嘉兴） |
| cn-sh | 金融云-华东（上海） |
| cn-gd2 | 华南（广州2） |
| cn-gd | 华南（广州） |
| cn-guiyang1 | 西南（贵阳） |
| hk | 香港 |
| tw-tp | 台湾（台北） |
| sg | 新加坡 |
| jpn-tky | 日本（东京） |
| kr-seoul | 韩国（首尔） |
| th-bkk | 泰国（曼谷） |
| idn-jakarta | 印度尼西亚（雅加达） |
| vn-sng | 越南（胡志明） |
| ph-mnl | 菲律宾（马尼拉） |
| ind-mumbai | 印度（孟买） |
| pk-khi | 巴基斯坦（卡拉奇） |
| us-den | 美国（丹佛） |
| us-ca | 美国（洛杉矶） |
| us-ws | 美国（华盛顿） |
| bra-saopaulo | 巴西（圣保罗） |
| rus-mosc | 俄罗斯（莫斯科） |
| ge-fra | 德国（法兰克福） |
| uk-london | 英国（伦敦） |
| uae-dubai | 阿联酋（迪拜） |
| afr-nigeria | 尼日利亚（拉各斯） |
| uz-tas | 乌兹别克斯坦（塔什干） |
| kz-ala | 哈萨克斯坦（阿拉木图） |
| mx-mex | 墨西哥（墨西哥城） |

**注意**：这是UCloud全平台的地域列表，星图/UModelVerse服务不一定在所有地域都开通。已知至少部分星图管理API（如 `ListUMInferRequestLogs`）文档明确提示"可先调用 `ListUMInferRegions` 获取可选地域"，说明星图自己维护了一份可用地域子集。如果用户不确定星图业务实际能用哪些地域，优先建议：

1. 先调用 `ListUMInferRegions`（星图管理API之一）拿到当前账号可用的地域列表，再让用户从中选择；
2. 如果暂时无法调用该接口，退而参考上表中的地域码，但要向用户说明这只是UCloud通用地域列表，不代表星图服务已在该地域开通。

常见的星图/UModelVerse相关地域包括 `cn-wlcb`（乌兰察布）和 `us-ca`（洛杉矶），不确定时应向用户确认，不要替用户擅自选择。

## 确定 ProjectId

如果用户不清楚自己的 `ProjectId`，调用管理API `GetProjectList` 获取账号下的项目列表。这个接口本身**不需要** `Region`/`ProjectId`，只需要标准的公共参数：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `Action` | string | 是 | 固定为 `GetProjectList` |
| `PublicKey` | string | 是 | 用户公钥 |
| `Signature` | string | 是 | 按上面的算法计算 |
| `IsFinance` | string | 否 | 是否财务账号（`Yes`/`No`），一般不用填 |

响应中关心的字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `RetCode` | int | 0表示成功 |
| `ProjectCount` | int | 项目总数 |
| `ProjectSet` | array | 项目列表，每项包含 `ProjectId`、`ProjectName`、`IsDefault`（是否默认项目）等字段 |

拿到列表后，把 `ProjectId` 和 `ProjectName` 都展示给用户，让用户确认要用哪个项目，不要替用户自动挑选（除非用户明确说"用默认项目"，这时可以选 `IsDefault=true` 的那一项）。

## 星图管理API详细规范

`references/` 目录下收录了目前已确认存在的21个星图管理API的详细请求/响应参数规范（每个接口一个独立文件，命名与Action对应的snake_case一致）。调用某个具体接口前，先打开对应文件确认它的业务参数、必填项和响应结构，不要凭印象或照抄别的接口的参数假设——即使是同一分组下的接口，参数也可能有细微差异（比如是否需要`Region`、数组参数是否带`.N`后缀等）。

**推理API Key管理：**
- [CreateUMInferAPIKey](references/create_um_infer_api_key.md) — 创建apikey
- [DeleteUMInferAPIKey](references/delete_um_infer_api_key.md) — 删除apikey
- [UpdateUMInferAPIKey](references/update_um_infer_api_key.md) — 更新apikey
- [ListUMInferAPIKey](references/list_um_infer_api_key.md) — 列表查询APIKey

**订单与账单管理：**
- [DownloadListPaidOrders](references/download_list_paid_orders.md) — 下载已完成订单明细
- [DownloadListUnpaidOrders](references/download_list_unpaid_orders.md) — 下载欠费订单明细
- [DownloadOrderSummary](references/download_order_summary.md) — 下载订单汇总
- [GetFilterOptions](references/get_filter_options.md) — 查询订单筛选选项
- [GetOrderAmount](references/get_order_amount.md) — 查询订单汇总统计
- [ListPaidOrderSummary](references/list_paid_order_summary.md) — 查询已完成订单汇总
- [ListPaidOrders](references/list_paid_orders.md) — 查询已完成订单明细
- [ListUnpaidOrderSummary](references/list_unpaid_order_summary.md) — 查询欠费订单汇总
- [ListUnpaidOrders](references/list_unpaid_orders.md) — 查询欠费订单明细
- [StartPayUnpaidOrders](references/start_pay_unpaid_orders.md) — 批量支付欠费订单（写操作，谨慎调用）

**模型广场查询：**
- [GetUFSquareModelDetail](references/get_uf_square_model_detail.md) — 获取广场模型详情
- [GetUFSquareModelPrices](references/get_uf_square_model_prices.md) — 批量查询模型价格
- [ListUFSquareModel](references/list_uf_square_model.md) — 查询模型广场数据
- [ListUFSquareModelFiltersAuth](references/list_uf_square_model_filters_auth.md) — 查询模型广场过滤条件

**推理请求日志：**
- [DownloadUMInferRequestLog](references/download_um_infer_request_log.md) — 导出推理请求日志
- [GetUMInferRequestLogDetail](references/get_um_infer_request_log_detail.md) — 原始日志详情
- [ListUMInferRequestLogs](references/list_um_infer_request_logs.md) — 日志明细列表

以上列表可能不完整（例如文档中提到的 `ListUMInferRegions` 就未在索引页出现，尚未收录）。索引页地址：`https://astraflow.ucloud.cn/reference/modelverse`，发现新接口时按同样的模板在 `references/` 下补充。

### 已知的文档自身存疑点

整理过程中发现官方文档存在几处需要工程师在真实调用时留意的不一致，均已在对应文件的"响应字段"或"示例"小节里逐条标注，这里列出汇总以便快速定位：

- `CreateUMInferAPIKey` / `ListUMInferAPIKey`：响应字段表中 `Data` 应为 `APIKey` 对象（或其数组），但文档给出的响应示例里 `Data` 被渲染成字符串/空对象，与字段类型不符。
- `ListUMInferRequestLogs`：响应示例顶层多出一个字段表未列出的 `TotalCount`。
- `ListUFSquareModelFiltersAuth`：响应字段文档本身只给出了 `RetCode`/`Action`/`Message` 三个通用字段，没有任何具体的过滤条件数据结构，怀疑文档遗漏。
- `GetFilterOptions`：响应示例中多个 `array[...]` 类型字段（如 `PricingSKUs`、`ProductCodes`、`Projects`、`Regions`）被渲染成裸字符串或空对象，与字段表类型不符。
- `GetOrderAmount`：响应示例缺少字段表标注为必填的 `TotalOrderAmount`，且多出一个字段表未记录的顶层 `Data`（空对象，含义未知）；另外该接口的 `OrganizationIds.N` 类型是 `string`，与其他几个订单接口里同名参数的 `int` 类型不一致。
- `ListUnpaidOrderSummary`：`OrderTypes` 参数不带 `.N` 后缀，与其余几个订单接口里同名参数的数组写法（`OrderTypes.N`）不一致。
- `StartPayUnpaidOrders`：响应字段 `Results` 文档标注为单个 `PayResult` 对象引用而非数组，但按"批量支付多个订单"的语义应为每个订单号返回一条结果，怀疑文档表述有误；响应示例中 `Results` 也只是空对象。

以上均为**原始文档本身的表述问题**，本技能只如实记录、不做主观修正；实际联调时应以真实返回结果为准，如发现文档确有错误，建议反馈给星图平台文档维护方。

## 调用一个具体星图管理API的操作流程

1. 确认已拿到 `PublicKey`、`PrivateKey`、`Region`（如该接口需要）、`ProjectId`（如该接口需要）。
2. 打开 `references/` 目录下目标 `Action` 对应的文件，确认它具体需要哪些业务参数（不是每个接口都要 `Region`，比如 `GetProjectList` 就不需要；具体以该文件为准，不要照抄别的接口的参数假设）。如果 `references/` 里还没有该接口，去索引页 `https://astraflow.ucloud.cn/reference/modelverse` 查找并按同样模板补充。
3. 组装完整参数集合（公共参数 + 业务参数），按上面的算法计算 `Signature`。
4. 发送请求，检查 `RetCode`，非0时按 `Message` 排查问题。
5. 如果怀疑是签名问题，优先检查：参数是否遗漏、排序是否正确、`PrivateKey` 是否放在拼接串末尾且没有被当作参数发送出去。
