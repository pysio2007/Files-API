# Files-API 文档管理服务

这是一个基于 Go 实现的文档管理服务，用于同步 Git 仓库中的文档到 Minio 对象存储，并提供统一的访问接口。

## 功能特点

- 支持从多个 Git 仓库同步文档
- 自动同步到 Minio 对象存储
- 提供统一的文档访问端点
- 支持配置公开访问路径
- 定时自动同步更新

## 配置说明

首次运行时会自动从 `config.example.yaml` 创建 `config.yaml`。
编辑 `config.yaml` 以配置你的服务。

示例配置文件包含所有可用选项的说明。配置文件应该保持在 `.gitignore` 中以避免泄露敏感信息。

配置文件位于 `config/config.yaml`，包含以下主要配置项：

### 服务器配置
```yaml
server:
  port: 8080  # 服务器端口
  host: "0.0.0.0"  # 监听地址
```

### Minio配置
```yaml
minio:
  endpoint: "play.min.io"  # Minio服务器地址
  accessKey: "your-access-key"  # 访问密钥
  secretKey: "your-secret-key"  # 访问密钥
  useSSL: true  # 是否使用SSL
  bucket: "documents"  # 存储桶名称
```

### Git仓库配置
```yaml
git:
  cachePath: ".cache/repos"  # Git仓库缓存目录
  repositories:
    - url: "https://github.com/user/repo1"  # 仓库地址
      branch: "main"  # 分支名称
      localPath: "repos/repo1"  # 相对于cachePath的路径
      minioPath: "repo1"  # Minio中的存储路径
```

### 公开路径配置
```yaml
exposedPaths:
    - urlPath: "/public-docs"  # 访问URL路径
      minioPath: "public"  # Minio中对应的路径
```

## 文件缓存

服务会将Git仓库同步到本地缓存目录(.cache/repos)，以提高性能和减少Git操作。缓存目录结构：

```
.cache/
  repos/
    repo1/    # 第一个仓库的缓存
    repo2/    # 第二个仓库的缓存
```

缓存目录已添加到 .gitignore，不会被版本控制。

## 使用方法

1. 配置 config.yaml
2. 运行服务:
```bash
go run main.go
```

## API接口

服务提供简单直接的文件访问接口：

### 文件访问

- 格式: `GET /{minioPath}/{filePath}`
- 示例:
  ```
  # 访问 repo1 目录下的图片
  GET /repo1/images/a.jpg
  
  # 访问 public 目录下的文档
  GET /public/docs/guide.pdf
  ```

### 访问规则

- URL路径直接映射到 Minio 存储路径
- 只能访问配置文件中定义的 minioPath 目录
- 示例配置:
  ```yaml
  git:
    repositories:
      - minioPath: "repo1"    # 可通过 /repo1/... 访问
      - minioPath: "docs"     # 可通过 /docs/... 访问
  exposedPaths:
    - minioPath: "public"    # 可通过 /public/... 访问
  ```

## 使用示例

1. 配置文件设置:
```yaml
git:
  repositories:
    - url: "https://github.com/user/repo1"
      minioPath: "repo1"
exposedPaths:
    - minioPath: "public"
```

2. 访问文件:
- `http://localhost:8080/repo1/image.jpg` -> 访问 repo1 仓库中的 image.jpg
- `http://localhost:8080/public/doc.pdf` -> 访问 public 目录中的 doc.pdf

## 许可证

基于 AGPLv3 许可证开源
