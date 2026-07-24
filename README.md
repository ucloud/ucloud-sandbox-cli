# UCloud Sandbox CLI

强大的命令行工具，用于本地管理沙箱生命周期、构建模板及执行运维任务。

UCloud Sandbox CLI 是开发者最常用的工具之一。它不仅可以帮助您快速初始化和构建沙箱模板，还可以实时查看沙箱监控指标以及执行批量操作。

## 安装指南

### 卸载旧的基于npm的CLI（可选）

在`v1.0`及以前的版本中，我们的CLI是基于npm构建和分发的，在后续版本中，我们改为了手动执行安装脚本。

如果您安装过`v1.0.x`的`ucloud-sandbox-cli`，请先使用下面的命令进行卸载：

```bash
npm uninstall -g @ucloud-sdks/ucloud-sandbox-cli
```

如果您之前从未安装过`ucloud-sandbox-cli`，或者没有使用npm安装过，可以跳过这一步。

### 安装CLI

使用下面的命令安装CLI：

```bash
curl -sS https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh | sh
```

安装脚本会要求您确认安装路径（默认为`/usr/local/bin`），直接输入回车可以确认安装，或者您也可以手动输入安装路径。注意请确保安装路径在您的`$PATH`下面以可以直接使用命令行。

## 身份认证与配置

### 环境注入

CLI 会优先读取环境变量中的 API Key、地域和其他配置。

```bash
export UCLOUD_SANDBOX_API_KEY=your_api_key
export UCLOUD_SANDBOX_REGION=region
```

如需使用 HTTP 而不是 HTTPS 连接控制面和沙箱，可通过环境变量启用 `insecure`：

```bash
export UCLOUD_SANDBOX_INSECURE=true
```

也可以在 `~/.ucloud-sandbox-cli/config.json` 中持久化该配置：

```json
{
  "insecure": true
}
```

API key可以从星图平台的[密钥管理](https://astraflow.ucloud.cn/modelverse/api-keys)获取。

可用地域可以参考：[切换地域](https://astraflow.ucloud.cn/docs/agent-sandbox/product/region)。

### 持久化认证

配置持久化认证：

```bash
# 这个命令会要求您输入API key并选择默认地域
ucloud-sandbox-cli login
```

删除持久化认证：

```bash
ucloud-sandbox-cli logout
```

> 在持久化认证生效的情况下，仍然可以使用环境变量来替换API key和地域。

### 切换地域

快速选择并切换地域：

```bash
# 会列出当前可用的地域供您选择
ucloud-sandbox-cli region
```

## 沙箱运行管理

### 创建与连接

快速创建沙箱并进入交互式终端：

```bash
# 使用内置模板创建沙箱
ucloud-sandbox-cli sandbox create [template]
# 简写：ucloud-sandbox-cli sbx cr [template]
```

**内置模板：**

```bash
# 代码解释器 - 预装 Python 和数据科学库
ucloud-sandbox-cli sandbox create code-interpreter-v1
 
# 桌面环境 - 支持图形化应用和浏览器
ucloud-sandbox-cli sandbox create desktop
 
# 基础环境 - 轻量级 Linux 环境
ucloud-sandbox-cli sandbox create base
```

> 创建成功后，CLI 会自动连接终端，您可以像操作本地 Shell 一样执行命令。按`Ctrl+D`或输入`exit`退出连接(沙箱继续运行)。

### 连接现有沙箱

重新连接到已运行的沙箱实例：

```bash
ucloud-sandbox-cli sandbox connect <sandbox-id>
# 简写：ucloud-sandbox-cli sbx connect <sandbox-id>
```

### 列表查询

查看名下所有活跃（运行或暂停）的沙箱实例：

```bash
ucloud-sandbox-cli sandbox list
# 简写：ucloud-sandbox-cli sandbox ls
```

### 强制关停 (Kill)

立即释放沙箱资源：

```bash
# 关停特定 ID
ucloud-sandbox-cli sandbox kill <sandbox-id>
 
# 关停所有活跃沙箱
ucloud-sandbox-cli sandbox kill --all
```

### 监控

实时洞察沙箱运行状态：

```bash
# 查看资源占用指标 (CPU/RAM/Disk)
ucloud-sandbox-cli sandbox metrics <sandbox-id>

# 持续查看指标
ucloud-sandbox-cli sandbox metrics <sandbox-id> -w
```

## 模板构建管理

### 初始化模板项目

创建一个标准化的模板开发目录：

```bash
ucloud-sandbox-cli tpl init my-custom-env --cpu <cpu> --memory <memory>
cd my-custom-env
```

### 构建模板

在上面的`my-custom-env`里面，您可以看到`template.dockerfile`文件，您需要编辑这个文件，输入`RUN`命令以定义构建模板需要的命令。

构建模板：

```bash
ucloud-sandbox-cli tpl build my-custom-env
```

### 发布模板

默认情况下，模板只能由您当前的项目访问，如果您需要其他人也能使用模板，需要公开：

```bash
ucloud-sandbox-cli tpl publish <template-id>
```

取消公开：

```bash
ucloud-sandbox-cli tpl publish --unpublish <template-id>
```

## 典型工作流示例

1. 准备环境：`ucloud-sandbox-cli login`
2. 创建模板：`ucloud-sandbox-cli template init` -> 编写 `Dockerfile`
3. 业务接入：在 SDK 中使用 `Sandbox.create(template='my-agent-env')`
4. 资源回收：`ucloud-sandbox-cli sandbox kill --all`
