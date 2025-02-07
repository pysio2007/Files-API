<div align="center">

# Files-API
![Files-API](https://socialify.git.ci/pysio2007/Files-API/image?custom_description=自动同步Github+Repo到Minio&description=1&font=Inter&forks=1&language=1&name=1&owner=1&pattern=Signal&pulls=1&stargazers=1&theme=Auto)

[![GitHub issues](https://img.shields.io/github/issues/pysio2007/Files-API)](https://github.com/pysio2007/Files-API/issues)
[![GitHub license](https://img.shields.io/github/license/pysio2007/Files-API)](https://github.com/pysio2007/Files-API/blob/main/LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/pysio2007/Files-API)](https://github.com/pysio2007/Files-API/stargazers)
[![Go Report Card](https://goreportcard.com/badge/github.com/pysio2007/Files-API)](https://goreportcard.com/report/github.com/pysio2007/Files-API)
[![Go Version](https://img.shields.io/github/go-mod/go-version/pysio2007/Files-API)](https://github.com/pysio2007/Files-API)

🚀 一个高性能的 Git 仓库文件同步和分发服务，支持自动同步到 Minio 对象存储。

[English](./README_EN.md) | 简体中文

</div>

## 📚 目录

- [✨ 特性](#-特性)
- [🚀 快速开始](#-快速开始)
- [📝 配置说明](#-配置说明)
- [🛠️ API 接口](#️-api-接口)
- [📈 性能优化](#-性能优化)
- [🔧 调试指南](#-调试指南)
- [🤝 参与贡献](#-参与贡献)
- [📄 开源协议](#-开源协议)

## ✨ 特性

- 🔄 支持多 Git 仓库自动同步
- ⏱️ 每个仓库可独立配置同步间隔
- 🚀 自定义并发上传线程数
- 📝 增量更新，只同步变更文件
- 🔗 统一的文件访问 API
- 🎯 自定义访问路径映射
- 🔒 SHA1 校验确保文件一致性
- 💫 异步同步不影响访问性能
- 📦 本地缓存提升访问速度

## 🚀 快速开始

### 环境要求

- Go 1.16+
- Minio 服务器 (或其他 S3 兼容存储)
- Git

### 安装

```bash
# 克隆仓库
git clone https://github.com/pysio2007/Files-API.git
cd Files-API

# 安装依赖
go mod download

# 运行服务
go run main.go
```

首次运行时会自动生成默认配置文件 `config.yaml`。

## 📝 配置说明

### 主要配置项

```yaml
server:
    port: 8080
    host: "0.0.0.0"
    enableAPI: true      # 是否启用 API 服务
    apiOnly: false       # 是否仅使用 API（禁用文件直接访问）
    legacyAPI: true      # 启用旧版API支持

minio:
    endpoint: "play.min.io"
    accessKey: "your-access-key"
    secretKey: "your-secret-key"
    useSSL: true
    bucket: "documents"
    usePublicURL: true   # 使用预签名URL直接访问
    maxWorkers: 16       # 并发上传线程数

cache:
    enabled: true
    directory: ".cache/files"
    maxSize: 1000        # 缓存容量(MB)
    ttl: "7d"            # 缓存有效期
    cacheControl: "30d"  # CDN缓存时间
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

### 缓存配置
```yaml
cache:
    enabled: true               # 启用文件缓存
    directory: ".cache/files"   # 缓存目录
    maxSize: 1000              # 缓存最大容量(MB)
    ttl: "7d"                  # 缓存有效期（支持 m/h/d/y）
    cacheControl: "30d"        # 静态文件CDN缓存时间
    enableAPICache: true       # 启用API缓存控制
    apiCacheControl: "5m"      # API响应的缓存时间
    cacheLog: true             # 记录缓存操作日志
    hitLog: true               # 记录缓存命中日志
```

### 多桶配置

支持配置多个存储桶以满足不同数据存储需求。每个桶配置项说明：
- Name: 桶在配置中的标识，用于路由匹配。
- Endpoint: 存储服务器地址。
- AccessKey & SecretKey: 认证凭证。
- UseSSL: 是否启用SSL。
- BucketName: 实际桶名称。
- BasePath: 桶中文件的基础路径（可留空）。
- ReadOnly: 是否只读（只读存储桶不允许写入操作）。

示例配置：
```yaml
buckets:
  - name: "Images"            # 路由标识，如通过 /Images/ 访问
    endpoint: "minioapi.example.com"
    accessKey: "your-access-key"
    secretKey: "your-secret-key"
    useSSL: true
    bucketName: "pysioimages"   # 实际的桶名称
    basePath: ""                # 根目录
    readOnly: true
```

### 缓存配置进阶说明

1. API 缓存控制
```yaml
cache:
    enableAPICache: true       # 启用API缓存功能
    apiCacheControl: "5m"      # API缓存时间
    apiExcludePaths:          # 不缓存的API路径
        - "/api/files/sync/status"  # 同步状态接口不缓存
```

2. 缓存排除规则
- 支持精确路径匹配
- 支持路径前缀匹配
- 在 enableAPICache 为 true 时仍然生效
- 优先级高于全局缓存设置

3. 性能建议
- 动态内容的API应加入排除列表
- 静态内容建议启用较长的缓存时间
- 监控类接口建议禁用缓存

### 缓存机制说明

1. 本地缓存
   - 缓存文件内容和元数据到本地磁盘
   - 自动清理过期的缓存文件
   - 支持配置缓存目录大小限制
   - 缓存有效期可独立配置

2. 分离控制
   - API和静态文件分别配置缓存时间
   - 可单独开关API响应的缓存控制
   - 静态文件默认较长缓存时间(30天)
   - API响应默认较短缓存时间(5分钟)

3. CDN支持
   - 通过Cache-Control响应头控制CDN缓存
   - 支持配置不同资源类型的缓存时间
   - 适配各类CDN服务

4. 缓存监控
   - 可选启用缓存操作日志
   - 可选启用缓存命中日志
   - 记录缓存清理和过期情况
   - 监控缓存空间使用情况

5. 清理机制
   - 自动清理过期缓存
   - 定期检查缓存大小限制
   - 支持手动清理缓存
   ```bash
   ./Files-API --clear-cache   # 清理缓存
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
- `s`: 秒，例如：`"60s"` 表示60秒
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

### 旧版 API 支持

服务支持自动重定向旧版 API 路径到新版格式：

1. 配置启用：
```yaml
server:
    legacyAPI: true     # 启用旧版API支持
```

2. 路径映射规则：
- 旧版：`/files/Pysio-Images/example.png`
- 新版：`/Pysio-Images/example.png`

3. 重定向说明：
- 使用 301 永久重定向
- 自动去除 `/files/` 前缀
- 保留原始查询参数
- 记录重定向日志（如果启用）

4. 日志记录：
```yaml
logs:
    redirectLog: true   # 启用重定向日志记录
```

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

### 外部URL配置

服务支持配置外部URL资源的自动同步和缓存：

```yaml
externalURLs:
    - path: "/external/banner.jpg"      # 访问路径
      mainURL: "https://example.com/banner.jpg"  # 主要下载地址
      backupURLs:                       # 备用下载地址列表
        - "https://backup1.com/banner.jpg"
        - "https://backup2.com/banner.jpg"
      minioPath: "external/banner.jpg"  # 在Minio中的存储路径
      cacheControl: "max-age=3600"      # 缓存控制头，如 "no-cache" 或 "max-age=3600"
      checkInterval: "1h"               # 更新检查间隔，可为 "1h", "1d" 等

    - path: "/external/logo.png"
      mainURL: "https://example.com/logo.png"
      backupURLs: 
        - "https://cdn.example.com/logo.png"
      minioPath: "external/logo.png"
      cacheControl: "no-cache"          # 禁用缓存
      checkInterval: "1d"               # 每天检查一次
```

配置说明：
- path: 通过API访问时的路径
- mainURL & backupURLs: 下载资源的主要和备用地址，当下载失败时依次尝试备用URL
- minioPath: 文件在Minio存储的路径
- cacheControl: 用于设置HTTP缓存头，推荐在调试时开启日志输出以便定位问题
- checkInterval: 定时检查更新的时间间隔，建议根据资源的重要性设置合理的值

工作原理：
1. 首次请求时根据配置下载对应资源，并存储至Minio。
2. 按照checkInterval定期检查文件是否已更新，若下载失败则自动尝试备用地址。
3. 用户可通过更新日志和错误提示进行调试，确保Minio的写入权限和网络连接正常。

调试建议：
- 如果资源未更新，请检查外部URL是否可访问以及网络是否通畅。
- 对于缓存问题，可使用 curl 命令中的 -H "Cache-Control: no-cache" 测试绕过本地缓存。

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

## 🛠️ API 接口

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

### 同步状态接口

获取所有仓库的同步状态信息。

```http
GET /api/files/sync/status
```

响应格式：
```json
{
    "code": 200,
    "message": "success",
    "data": {
        "repo1": {
            "lastSync": "2024-02-05T12:34:56Z",  // 最后同步时间
            "nextSync": "2024-02-05T13:34:56Z",  // 下次同步时间
            "progress": 100,                      // 同步进度(0-100)
            "totalFiles": 50,                     // 总文件数
            "currentFiles": 50,                   // 已处理文件数
            "status": "idle"                      // 状态(idle/syncing/error)
        }
    }
}
```

状态说明：
- `idle`: 空闲状态，等待下次同步
- `syncing`: 正在同步
- `error`: 同步出错，查看 error 字段获取详细信息
- `unknown`: 初始状态

监控指标：
- `lastSync`: 最后同步时间
- `nextSync`: 下次预计同步时间
- `progress`: 当前同步进度(0-100)
- `totalFiles`: 总文件数
- `currentFiles`: 已处理文件数
- `error`: 错误信息(如果有)

监控示例：
```bash
# 查看同步状态
curl http://localhost:8080/api/files/sync/status

# 使用 watch 监控同步进度
watch -n 1 'curl -s http://localhost:8080/api/files/sync/status | jq'
```

## 🔄 工作原理

1. 定期从 Git 仓库拉取最新文件
2. 使用 SHA1 校验检测文件变更
3. 只上传变更的文件到 Minio
4. 提供直接的文件访问服务

## 📈 性能优化

- 文件缓存减少Git操作
- SHA1增量更新减少传输
- 异步同步避免阻塞
- 多worker并行处理

## 🔧 调试指南

### 日志配置说明

1. 完整调试配置（记录所有信息）
```yaml
logs:
    accessLog: true     # 记录所有访问请求
    processLog: true    # 记录文件处理过程
    redirectLog: true   # 记录URL重定向
    presignLog: true    # 记录预签名URL生成
    saveToFile: true    # 同时输出到文件和控制台
    maxSize: 100        # 日志目录限制（MB）
    directory: "logs"   # 日志目录
```

2. 缓存调试配置
```yaml
cache:
    cacheLog: true      # 记录缓存操作
    hitLog: true        # 记录缓存命中
```

### 常见问题排查

1. 文件同步问题
```bash
# 检查同步状态
./Files-API --sync

# 同步指定仓库
./Files-API --rsync=static

# 查看同步日志
tail -f logs/Files-API-$(date +%Y-%m-%d).log | grep "同步"
```

2. 缓存问题
```bash
# 检查缓存状态
ls -lh .cache/files/

# 清理缓存后重试
./Files-API --clear-cache

# 监控缓存命中
tail -f logs/Files-API-$(date +%Y-%m-%d).log | grep "Cache hit"
```

3. Minio连接问题
```bash
# 检查Minio连接
curl -I http://{minio-endpoint}/
# 或使用其他工具如 s3cmd test

# 验证配置
cat config.yaml | grep minio -A 8
```

### 性能分析

1. 使用Go性能分析工具
```bash
# 启用性能分析
GODEBUG=gctrace=1 ./Files-API

# 使用pprof
go tool pprof http://localhost:8080/debug/pprof/heap
```

2. 监控系统资源
```bash
# 查看内存使用
ps -o pid,ppid,%mem,rss,cmd -p $(pgrep Files-API)

# 查看文件描述符
lsof -p $(pgrep Files-API)
```

### 开发调试

1. 启用所有日志
```bash
# 修改配置文件
vim config.yaml
# 将所有日志选项设置为 true

# 使用更短的同步间隔测试
checkInterval: "1m"
```

2. API测试
```bash
# 测试文件列表API
curl "http://localhost:8080/api/files/static/?page=1&pageSize=10"

# 测试文件访问
curl -I "http://localhost:8080/static/test.txt"
```

3. 性能测试
```bash
# 测试并发请求
ab -n 1000 -c 10 http://localhost:8080/api/files/static/

# 测试文件上传
for i in {1..10}; do
    ./Files-API --sync
done
```

### 错误码说明

| 状态码 | 说明 | 处理方法 |
|--------|------|----------|
| 400 | 请求参数错误 | 检查API参数 |
| 404 | 文件不存在 | 检查路径和同步状态 |
| 500 | 服务器内部错误 | 查看详细日志 |
| 503 | Minio服务不可用 | 检查Minio连接 |

## 🤝 参与贡献

1. Fork 本项目
2. 创建新特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 提交 Pull Request

## 📄 开源协议

本项目采用 [AGPL-3.0](./LICENSE) 协议开源。

## 更新记录

- 增加 CORS 配置支持，可通过 configuration 中的 `server.allowOrigins` 自定义允许跨域的域名。
- 更新 CORS 中间件实现，自动处理预检请求并返回正确的跨域头部。
- 默认配置文件中已添加 `allowOrigins` 配置项，默认值为 `["http://localhost:8080"]`。

<div align="center">

### 喜欢这个项目？请给它一个 ⭐️

</div>
