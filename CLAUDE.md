# UCloud Sandbox CLI

## 项目概述

这是一个用Go编写的命令行工具，用于管理和操作UCloud Sandbox沙箱服务。UCloud Sandbox的API完全兼容E2B格式，CLI实现可以参考E2B的CLI设计。

## 快速开始

### 编译项目

```bash
make build
```

编译后的二进制文件位于 `./bin/ucloud-sandbox-cli`

### 运行测试

```bash
make test
```

执行所有单元测试（包括 `internal/` 下的所有测试）

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
所有CLI命令的实现都放在此目录，每个命令组一个子包（如 `cmd/sandbox/`、`cmd/auth/`）。

**命令组：** 子包导出 `Command() *cobra.Command`，返回命令组并注册其下的子命令；顶层命令组在 `main.go` 中注册。

**子命令：** 每个子命令是一个实现了 `cmd.Operation` 接口的 `<name>Operation` 结构体（未导出），flag 绑定到结构体字段：
- `Command() *cobra.Command`：定义 `Use`、`Short`、`Args` 和 flags
- `Run(ctx cmd.OperationContext) error`：执行命令，通过 `ctx.Client` 调用 SDK

注册时用 `cmd.Build(&xxxOperation{})` 包装，它会加载配置并创建 SDK client；只读写本地配置、不需要 client 的命令（如 `login`、`update`）用 `cmd.BuildLocal`。

**新增命令时需要：**
1. 参考现有命令的实现方式（如 `cmd/sandbox/kill.go`、`cmd/auth/region.go`）
2. 遵循相同的代码风格和错误处理模式
3. 命名规范：命令组为 `Command()`，子命令为 `<name>Operation`

### `internal/`
存放CLI使用的公用包

**internal包要求：**
- 尽量为所有功能编写单元测试
- 单元测试使用 **testify** 库（`github.com/stretchr/testify`）
- 测试文件命名：`<package>_test.go`
