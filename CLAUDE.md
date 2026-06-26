# UCloud Sandbox CLI

## 项目概述

这是一个用Go编写的命令行工具，用于管理和操作UCloud Sandbox沙箱服务。UCloud Sandbox的API完全兼容E2B格式，CLI实现可以参考E2B的CLI设计。

## 核心依赖

1. **UCloud Sandbox SDK** (`github.com/ucloud/ucloud-sandbox-sdk-go`)
   - UCloud Sandbox的官方Go SDK
   - 用于调用UCloud的沙箱服务API
   - 源码位置：`submodules/ucloud-sandbox-sdk-go/`

2. **E2B CLI参考实现** (`github.com/e2b-dev/e2b`)
   - E2B原始CLI（JavaScript实现），UCloud Sandbox API与其完全兼容
   - 命令设计和用户体验均可参考此项目
   - 源码位置：`submodules/e2b/`

3. **Cobra** (`github.com/spf13/cobra`)
   - CLI框架，用于命令结构化管理

## 编码规范

- 代码注释使用**英文**编写
- 编写Plan时使用**中文**
- 命令只需要Short描述（英文），不需要Long描述

## 项目结构

### `cmd/`
所有CLI命令的实现都放在此目录。每个命令通过 `New<CommandName>Cmd()` 函数返回 `*cobra.Command`。

**新增命令时需要：**
1. 参考现有命令的实现方式（如 `login.go`、`logout.go`、`region.go`）
2. 遵循相同的代码风格和错误处理模式
3. 命令函数命名规范：`New<CommandName>Cmd()`

### `internal/`
存放CLI使用的公用包，包括：
- `config/` — 配置加载、保存和客户端创建
- `prompt/` — 命令行交互（使用 promptui）

**internal包要求：**
- 尽量为所有功能编写单元测试
- 单元测试使用 **testify** 库（`github.com/stretchr/testify`）
- 测试文件命名：`<package>_test.go`
