# QNAP File Station API SDK for Go - 项目概述

## 项目结构

```
qnap-filestation/
├── api/                          # 核心 API 客户端
│   ├── client.go                 # 主客户端实现
│   ├── client_test.go            # 客户端测试
│   ├── errors.go                 # 错误类型定义
│   ├── request.go                # HTTP 请求处理
│   └── response.go               # 响应处理
│
├── filestation/                  # File Station API 实现
│   ├── service.go                # 服务入口
│   ├── types.go                  # 数据类型定义
│   ├── file.go                   # 文件操作
│   ├── folder.go                 # 文件夹操作
│   ├── upload.go                 # 上传功能
│   ├── download.go               # 下载功能
│   ├── search.go                 # 搜索功能
│   ├── share.go                  # 分享链接管理
│   └── file_test.go              # 功能测试
│
├── internal/
│   └── testutil/
│       └── mockserver.go         # 测试 Mock 服务器
│
├── .github/workflows/
│   ├── ci.yml                    # CI/CD 工作流
│   └── release.yml               # 自动发布工作流
│
├── examples/
│   └── main.go                   # 使用示例
│
├── go.mod                        # Go 模块定义
├── go.sum
├── .golangci.yml                 # 代码检查配置
├── .goreleaser.yml               # 发布配置
├── README.md                     # 项目说明
├── LICENSE                       # MIT 许可证
└── CONTRIBUTING.md               # 贡献指南
```

## 已实现功能

### 核心 API (api/)
- ✅ 客户端初始化和配置
- ✅ SID-based 认证
- ✅ HTTP 请求/响应处理
- ✅ 错误处理和类型定义

### File Station 操作 (filestation/)
- ✅ 文件列表、删除、重命名、复制、移动
- ✅ 文件夹列表、创建、删除、重命名、复制、移动
- ✅ 文件上传（支持进度报告）
- ✅ 文件下载（支持进度报告）
- ✅ 文件搜索（按模式、类型、扩展名、大小）
- ✅ 分享链接管理（创建、列表、删除）

### CI/CD
- ✅ GitHub Actions CI 配置（多版本 Go 测试）
- ✅ 自动代码检查（golangci-lint）
- ✅ 自动发布流程（GoReleaser）
- ✅ Docker 构建支持

## 使用方法

```go
package main

import (
    "context"
    "log/slog"
    "github.com/fatelei/qnap-filestation/api"
    "github.com/fatelei/qnap-filestation/filestation"
)

func main() {
    // 创建客户端
    client, _ := api.NewClient(&api.Config{
        Host:     "192.168.1.100",
        Port:     8080,
        Username: "admin",
        Password: "password",
        Insecure: true,
        Logger:   slog.Default(),
    })

    // 登录
    ctx := context.Background()
    client.Login(ctx)
    defer client.Logout(ctx)

    // 使用 FileStation 服务
    fs := filestation.NewFileStationService(client)

    // 列出文件
    files, _ := fs.ListFiles(ctx, "/home", nil)

    // 上传文件
    fs.UploadFile(ctx, "/local/file.txt", "/home", nil)

    // 下载文件
    fs.DownloadFile(ctx, "/home/file.txt", "/local/download", nil)
}
```

## 下一步

1. 初始化 Git 仓库：
   ```bash
   cd qnap-filestation
   git init
   git add .
   git commit -m "Initial commit: QNAP File Station API SDK"
   ```

2. 推送到 GitHub：
   ```bash
   git remote add origin https://github.com/fatelei/qnap-filestation.git
   git push -u origin main
   ```

3. 发布第一个版本：
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

4. GitHub Actions 将自动构建和发布二进制文件

## 注意事项

- 目前测试还有一些 URL 解析问题需要修复（Mock Server）
- 实际使用时需要配置正确的 QNAP IP、端口和凭据
- 如果使用自签名证书，需要设置 `Insecure: true`

## 许可证

MIT License
