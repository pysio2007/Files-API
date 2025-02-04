![Files-API](https://socialify.git.ci/pysio2007/Files-API/image?custom_description=自动同步Github+Repo到Minio&description=1&font=Inter&forks=1&language=1&name=1&owner=1&pattern=Signal&pulls=1&stargazers=1&theme=Auto)

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

### 服务配置
```yaml
server:
    port: 8080          # 服务端口
    host: "0.0.0.0"     # 监听地址
    enableAPI: true     # 是否启用 API 接口
    apiOnly: false      # 是否仅使用 API（禁用文件直接访问）
```

### 服务模式说明

1. 完整模式 (默认)
   ```yaml
   enableAPI: true
   apiOnly: false
   ```
   - API 接口可用
   - 文件直接访问可用
   - 适合大多数场景

2. 仅 API 模式
   ```yaml
   enableAPI: true
   apiOnly: true
   ```
   - 只提供 API 接口
   - 禁用文件直接访问
   - 适合需要严格控制访问的场景

3. 仅文件服务模式
   ```yaml
   enableAPI: false
   apiOnly: false
   ```
   - 禁用 API 接口
   - 启用文件直接访问
   - 适合简单的静态文件服务

4. 无效配置
   ```yaml
   enableAPI: false
   apiOnly: true
   ```
   - 错误：两个服务都被禁用
   - 服务将拒绝启动

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

### 日志配置
```yaml
logs:
    accessLog: true     # 访问日志，记录所有文件请求
    processLog: false   # 处理详情，记录文件处理过程
    redirectLog: false  # 跳转详情，记录URL重定向信息
    presignLog: false   # 预签名URL详情，记录生成的临时访问链接
    saveToFile: true    # 是否保存日志到文件
    maxSize: 100        # 日志目录最大容量(MB)
    directory: "logs"   # 日志保存目录
```

日志管理功能：
1. 自动日志轮转
   - 按天切割日志文件
   - 自动在每日零点切换新文件
   - 文件名格式：Files-API-YYYY-MM-DD.log

2. 空间管理
   - 自动监控日志目录大小
   - 超过限制时删除最旧的日志
   - 默认限制 100MB 总容量
   - 可通过 maxSize 配置调整

3. 日志级别控制
   - accessLog：记录所有 HTTP 请求
   - processLog：记录文件处理详情
   - redirectLog：记录 URL 重定向
   - presignLog：记录预签名 URL 生成

4. 输出模式
   - saveToFile=true：同时输出到控制台和文件
   - saveToFile=false：仅输出到控制台
   - 默认开启文件保存

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

## 特殊启动参数

### 跳过首次同步 (--skip)

当使用 `--skip` 参数启动时，程序会：
1. 跳过启动时的初始同步
2. 等待各仓库配置的检查间隔后再进行首次同步
3. 适用于需要延迟同步的场景

示例：
```bash
# 正常启动（执行首次同步）
./Files-API

# 跳过首次同步
./Files-API --skip
```

使用场景：
- CI/CD 环境中避免重复同步
- 仓库内容暂时不可用时
- 需要等待外部服务就绪
- 控制同步时间窗口

## 命令行选项

### 基本命令
```bash
# 显示帮助信息
./Files-API -h
./Files-API --help

# 正常启动服务（含初始同步）
./Files-API
```

### 同步控制
```bash
# 跳过首次同步启动
./Files-API --skip

# 执行单次同步后退出
./Files-API --sync
```

### 日志管理
```bash
# 压缩所有日志文件
./Files-API --zip-logs

# 解压所有日志文件
./Files-API --unzip-logs

# 清除所有日志
./Files-API --clear-logs
./Files-API -cl
```

### 缓存清理
```bash
# 清除缓存目录
./Files-API --clear-cache
./Files-API -cc

# 清除所有日志和缓存
./Files-API --clear-all
```

命令详解：

1. 同步控制
   - `--skip`: 跳过首次同步，等待检查周期
   - `--sync`: 执行单次同步后退出

2. 日志管理
   - `--zip-logs`: 压缩所有日志为zip
   - `--unzip-logs`: 解压所有日志文件
   - `--clear-logs, -cl`: 清除所有日志

3. 缓存管理
   - `--clear-cache, -cc`: 清除缓存目录
   - `--clear-all`: 清除所有日志和缓存

清理操作说明：
- 所有清理命令都会显示释放的空间大小
- 日志清理包含 .log 和 .zip 文件
- 缓存清理会删除整个缓存目录
- 清理操作执行后自动退出

## API 接口

### 文件列表接口

获取指定目录下的文件和子目录列表。

```http
GET /api/files/{path}?page=1&pageSize=20
```

参数说明：
- `path`: 可选，目录路径
- `page`: 可选，页码，默认 1
- `pageSize`: 可选，每页条数，默认 20，最大 100

响应格式：
```json
{
    "code": 200,
    "message": "success",
    "data": [
        {
            "name": "images",
            "path": "static/images/",
            "isDirectory": true
        },
        {
            "name": "logo.png",
            "path": "static/logo.png",
            "size": 12345,
            "lastModified": "2024-02-05T12:34:56Z",
            "isDirectory": false,
            "url": "https://..."
        }
    ],
    "pagination": {
        "current": 1,
        "pageSize": 20,
        "total": 42
    }
}
```

响应字段说明：
1. 文件信息 (FileInfo)
   - `name`: 文件名或目录名
   - `path`: 完整路径
   - `size`: 文件大小（字节）
   - `lastModified`: 最后修改时间
   - `isDirectory`: 是否是目录
   - `url`: 文件访问链接（仅当配置 usePublicURL=true 时提供）

2. 分页信息 (pagination)
   - `current`: 当前页码
   - `pageSize`: 每页条数
   - `total`: 总条数

### 文件访问接口

直接访问文件内容。

```http
GET /{minioPath}/{filePath}
```

访问模式：
1. 重定向模式（usePublicURL=true）
   - 返回 302 重定向到预签名 URL
   - URL 有效期为 1 小时

2. 代理模式（usePublicURL=false）
   - 直接返回文件内容
   - 自动设置正确的 Content-Type
   - 支持大文件传输

示例：
```bash
# 访问文件
GET /static/images/logo.png

# 带 Accept 头获取 JSON 格式的文件信息
curl -H "Accept: application/json" http://localhost:8080/api/files/static/images/

# 分页查询
curl http://localhost:8080/api/files/static/?page=2&pageSize=50
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

- 启动时执行初始同步（可使用 --skip 跳过）
- 根据每个仓库的 checkInterval 定期检查
- 支持分钟/小时/天/年级别的同步间隔
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

## 调试指南

### 日志调试

1. 完整调试配置
```yaml
logs:
    accessLog: true     # 记录所有访问
    processLog: true    # 记录处理流程
    redirectLog: true   # 记录重定向
    presignLog: true    # 记录临时链接
    saveToFile: true    # 保存到文件
    maxSize: 100        # 限制总大小
    directory: "logs"   # 存储目录
```

2. 最小日志配置
```yaml
logs:
    accessLog: true     # 仅记录基本访问
    processLog: false
    redirectLog: false
    presignLog: false
    saveToFile: false   # 仅输出到控制台
```

3. 查看日志文件
```bash
# 查看今日日志
cat logs/Files-API-2025-02-05.log

# 监控实时日志
tail -f logs/Files-API-2025-02-05.log
```

## 许可证

基于 AGPLv3 许可证开源
