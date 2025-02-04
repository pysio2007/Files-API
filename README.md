# Files-API 文件同步服务

这是一个基于 Go 实现的文件同步和分发服务，用于将 Git 仓库中的文件自动同步到 Minio 对象存储，并提供统一的文件访问接口。

## 功能特点

- 支持多 Git 仓库自动同步
- 支持每个仓库独立配置同步间隔
- 支持自定义并发上传线程数
- 增量更新，只同步变更文件
- 提供统一的文件访问端点
- 支持自定义访问路径映射
- SHA1 校验确保文件一致性
- 异步同步不影响访问性能
- 文件缓存提升访问速度

## 配置说明

服务启动时会检查是否存在 config.yaml，如果不存在，程序将自动生成一个默认配置文件。请编辑 config.yaml 修改服务参数。

### 基本配置
```yaml
server:
  port: 8080          # 服务端口
  host: "0.0.0.0"     # 监听地址
```

### Minio配置
```yaml
minio:
    endpoint: "play.min.io"      # Minio服务器地址
    accessKey: "your-access-key" # 访问密钥
    secretKey: "your-secret-key" # 访问密钥
    useSSL: true                # 是否使用SSL
    bucket: "documents"         # 存储桶名称
    usePublicURL: true         # 是否使用Minio公共URL进行重定向
    maxWorkers: 16             # 最大并发上传线程数
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
      checkInterval: "1h"                   # 同步检查间隔 (支持 m/h/d/y)

exposedPaths:
    - urlPath: "/assets"        # 访问URL路径
      minioPath: "static"       # 存储路径前缀
```

### 时间间隔格式说明

支持的时间间隔格式：
- `m`: 分钟，例如：`"10m"` 表示10分钟
- `h`: 小时，例如：`"1h"` 表示1小时
- `d`: 天，例如：`"1d"` 表示1天
- `y`: 年，例如：`"1y"` 表示1年

未配置或配置无效时，默认使用 10 分钟作为检查间隔。

### 访问模式

服务支持两种访问模式：

1. 重定向模式（推荐）
   - 启用 `usePublicURL: true`
   - 自动使用 Minio 的预签名 URL
   - 支持直接从 Minio 服务器下载
   - 减轻应用服务器负载

2. 代理模式
   - 当 `usePublicURL: false` 或获取公共 URL 失败时
   - 通过应用服务器中转文件内容
   - 适用于内部网络或需要额外控制的场景

### 性能调优

1. 并发上传线程数
   - 通过 `minio.maxWorkers` 配置
   - 默认值为 16
   - 根据服务器性能和网络状况调整
   - 建议值范围：8-32

2. 仓库检查间隔
   - 每个仓库可独立配置
   - 支持分钟(m)、小时(h)、天(d)、年(y)
   - 未配置默认10分钟
   - 示例：
     ```yaml
     checkInterval: "30m"  # 30分钟
     checkInterval: "1h"   # 1小时
     checkInterval: "1d"   # 1天
     checkInterval: "1y"   # 1年
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
