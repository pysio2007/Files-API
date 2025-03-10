# Files-API OpenAPI 规范

本文档提供了Files-API服务的OpenAPI规范，可用于生成API文档和客户端代码。

## 概述

Files-API是一个文件存储和管理服务，提供以下功能：

- 文件列表查询
- 文件内容获取
- 多存储桶支持
- Git仓库同步状态查询

## API端点

API服务器地址：`https://files.pysio.online`

### 主要端点

1. **获取文件列表**
   - 端点：`/api/files/{path}`
   - 方法：GET
   - 描述：获取指定路径下的文件和目录列表，支持分页

2. **获取同步状态**
   - 端点：`/api/files/sync/status`
   - 方法：GET
   - 描述：获取所有Git仓库的同步状态

3. **获取指定桶中的文件信息**
   - 端点：`/{bucket}/{path}`
   - 方法：PATCH
   - 描述：获取指定存储桶中的文件信息

4. **获取文件内容**
   - 端点：`/{path}`
   - 方法：GET
   - 描述：直接获取文件内容，支持从配置的仓库或暴露路径访问

### 特定仓库和存储桶端点

1. **Pysio-FontAwesome仓库**
   - 端点：`/Pysio-FontAwesome/{path}`
   - 方法：GET
   - 描述：获取Pysio-FontAwesome仓库中的文件

2. **Avatar仓库**
   - 端点：`/Avatar/{path}`
   - 方法：GET
   - 描述：获取Avatar仓库中的文件

3. **Images存储桶**
   - 端点：`/Images/{path}`
   - 方法：GET
   - 描述：获取Images存储桶中的文件

4. **状态摘要**
   - 端点：`/status/summary.json`
   - 方法：GET
   - 描述：获取从外部URL同步的状态摘要信息

## 服务器环境

Files-API支持以下服务器环境：

1. **生产环境**：`https://files.pysio.online`
2. **允许的跨域来源**：`https://www.pysio.online`
3. **允许的跨域来源**：`https://pysio.online`
4. **本地开发环境**：`http://localhost:8080`

## 使用OpenAPI规范

### 在线查看

您可以使用以下工具在线查看API文档：

1. **Swagger UI**：将`openapi.json`文件导入到[Swagger Editor](https://editor.swagger.io/)

2. **ReDoc**：使用[ReDoc](https://redocly.github.io/redoc/)查看API文档

### 生成客户端代码

您可以使用OpenAPI生成器生成各种编程语言的客户端代码：

```bash
# 安装OpenAPI生成器
npm install @openapitools/openapi-generator-cli -g

# 生成JavaScript客户端
openapi-generator-cli generate -i openapi.json -g javascript -o ./client-js

# 生成Python客户端
openapi-generator-cli generate -i openapi.json -g python -o ./client-python

# 生成Go客户端
openapi-generator-cli generate -i openapi.json -g go -o ./client-go
```

## 示例请求

### 获取文件列表

```bash
curl -X GET "https://files.pysio.online/api/files/Pysio-FontAwesome?page=1&pageSize=20" -H "accept: application/json"
```

### 获取同步状态

```bash
curl -X GET "https://files.pysio.online/api/files/sync/status" -H "accept: application/json"
```

### 获取指定桶中的文件信息

```bash
curl -X PATCH "https://files.pysio.online/Images/example.jpg" \
  -H "Content-Type: application/json" \
  -d '{"bucket":"Images","path":"example.jpg"}'
```

### 获取文件内容

```bash
curl -X GET "https://files.pysio.online/Pysio-FontAwesome/css/all.min.css"
```

### 获取状态摘要

```bash
curl -X GET "https://files.pysio.online/status/summary.json"
```

## 配置说明

Files-API服务使用`config.yaml`文件进行配置，主要配置项包括：

- 服务器设置
- Minio存储设置
- Git仓库设置
- 日志设置
- 缓存设置
- 多存储桶设置
- 外部URL设置

详细配置说明请参考项目文档。

## 错误处理

API返回的错误格式如下：

```json
{
  "code": 500,
  "message": "服务器错误"
}
```

常见错误码：

- 400：无效的请求格式
- 403：未授权的访问或存储桶为只读
- 404：文件或存储桶不存在
- 405：方法不允许
- 500：服务器错误

## 联系方式

如有问题，请访问[Pysio官网](https://pysio.online)获取支持。 