# Files-API 文件同步服务

这是一个基于 Go 实现的文件同步和分发服务，用于将 Git 仓库中的文件自动同步到 Minio 对象存储，并提供统一的文件访问接口。

## 功能特点

- 支持多 Git 仓库自动同步
- 增量更新，只同步变更文件
- 提供统一的文件访问端点
- 支持自定义访问路径映射
- SHA1 校验确保文件一致性
- 异步同步不影响访问性能
- 文件缓存提升访问速度

## 配置说明

服务启动时会自动从 `config.example.yaml` 创建 `config.yaml`。修改 `config.yaml` 以配置服务参数。

### 基本配置
```yaml
server:
  port: 8080          # 服务端口
  host: "0.0.0.0"     # 监听地址

minio:
  endpoint: "play.min.io"        # Minio服务器地址
  accessKey: "your-access-key"   # 访问密钥
  secretKey: "your-secret-key"   # 访问密钥
  useSSL: true                   # 是否使用SSL
  bucket: "files"                # 存储桶名称
```

### 仓库和路径配置
```yaml
git:
  cachePath: ".cache/repos"      # 本地缓存目录
  repositories:
    - url: "https://github.com/user/repo1"   # 仓库地址
      branch: "main"                         # 分支名称
      localPath: "repos/repo1"              # 本地缓存路径
      minioPath: "static"                   # 存储路径前缀

exposedPaths:
    - urlPath: "/assets"        # 访问URL路径
      minioPath: "static"       # 存储路径前缀
```

## 工作原理

1. 定期从 Git 仓库拉取最新文件
2. 使用 SHA1 校验检测文件变更
3. 只上传变更的文件到 Minio
4. 提供直接的文件访问服务

## 文件访问

### URL 格式
- `GET /{minioPath}/{filePath}`

### 示例
```
# 访问静态资源
GET /static/images/logo.png
GET /assets/css/main.css

# 访问其他文件
GET /public/files/document.pdf
```

## 同步机制

- 启动时执行初始同步
- 每10分钟自动检查更新
- 异步处理不阻塞访问
- 支持多worker并行同步

## 部署说明

1. 准备环境:
   ```bash
   go mod download
   ```

2. 修改配置:
   ```bash
   cp config.example.yaml config.yaml
   vim config.yaml
   ```

3. 启动服务:
   ```bash
   go run main.go
   ```

## 性能优化

- 文件缓存减少Git操作
- SHA1增量更新减少传输
- 异步同步避免阻塞
- 多worker并行处理

## 许可证

基于 AGPLv3 许可证开源
