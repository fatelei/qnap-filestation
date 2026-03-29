# QNAP File Station API SDK - 项目结构

## 目录结构

```
qnap-filestation/
├── pkg/                          # 公共库代码
│   ├── api/                      # 核心 API 客户端
│   │   ├── client.go             # 主客户端实现
│   │   ├── client_test.go        # 客户端测试
│   │   ├── errors.go             # 错误类型定义
│   │   ├── request.go            # HTTP 请求处理
│   │   └── response.go           # 响应处理
│   │
│   └── filestation/              # File Station API 实现
│       ├── service.go            # 服务入口
│       ├── types.go              # 数据类型定义
│       ├── file.go               # 文件操作
│       ├── folder.go             # 文件夹操作
│       ├── upload.go             # 上传功能
│       ├── download.go           # 下载功能
│       ├── search.go             # 搜索功能
│       ├── share.go              # 分享链接管理
│       └── file_test.go          # 功能测试
│
├── internal/                     # 私有代码（外部不可导入）
│   └── testutil/
│       └── mockserver.go         # 测试 Mock 服务器
│
├── cmd/                          # CLI 工具
│   └── qnap-cli/
│
├── examples/                     # 使用示例
│   └── main.go
│
├── .github/workflows/            # GitHub Actions
│   ├── ci.yml                    # CI/CD 工作流
│   └── release.yml               # 自动发布工作流
│
├── docs/                         # 项目文档
│
├── go.mod                        # Go 模块定义
├── go.sum
├── .golangci.yml                 # 代码检查配置
├── .goreleaser.yml               # 发布配置
├── LICENSE                       # MIT 许可证
├── README.md                     # 项目说明
└── CONTRIBUTING.md               # 贡献指南
```

## 包说明

### pkg/api
核心 API 客户端，处理：
- 认证（SID-based session）
- HTTP 请求/响应
- 错误处理
- 配置管理

**导入路径**: `github.com/fatelei/qnap-filestation/pkg/api`

### pkg/filestation
File Station API 实现，提供：
- 文件操作（列表、删除、重命名、复制、移动）
- 文件夹操作（创建、列表、删除、重命名、复制、移动）
- 上传/下载（支持进度报告）
- 搜索功能
- 分享链接管理

**导入路径**: `github.com/fatelei/qnap-filestation/pkg/filestation`

### internal/testutil
测试工具，仅内部使用。

## 为什么使用 pkg/？

按照 [Standard Go Project Layout](https://github.com/golang-standards/project-layout)：

- **pkg/** - 可以被外部应用程序使用的库代码
- **internal/** - 私有应用程序和库代码
- **cmd/** - 主应用程序的入口点
- ****examples/** - 示例代码

这种结构的优势：
1. 清晰的公共 API 和私有实现分离
2. 符合 Go 社区的最佳实践
3. 易于维护和理解

## 使用示例

```go
import (
    "github.com/fatelei/qnap-filestation/pkg/api"
    "github.com/fatelei/qnap-filestation/pkg/filestation"
)

client, _ := api.NewClient(&api.Config{...})
fs := filestation.NewFileStationService(client)
```
