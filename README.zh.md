# gopkg

[English](README.md) | 中文

`gopkg` 是 Go 实用工具库，为 [scalebox](https://github.com/kaichao/scalebox) 提供核心功能支持。包含以下子包：

## 功能模块

### 1. `asyncbatch`
异步批处理工具，支持按批次大小或超时触发批量操作。

### 2. `pgbulk`
PostgreSQL 批量操作工具，优化批量插入（带ID返回）、更新等场景的性能。

### 3. `dbcache`
基于 [go-cache](https://github.com/patrickmn/go-cache) 的数据库缓存层，支持SQL模板化数据加载。

### 4. `exec`
跨环境命令执行工具，支持本地与SSH远程执行，捕获标准输出/错误流。


## 安装

运行以下命令安装 `gopkg` 包：

```sh
go get github.com/kaichao/gopkg
```

## 许可协议

MIT License
