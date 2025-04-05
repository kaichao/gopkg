# pgbulk

[English](README.md) | 中文

轻量级 PostgreSQL 批量操作 Go 工具包

## 功能特性

- **批量处理**: 自动将大数据集分块为最优批次
- **SQL模板**: 支持动态占位符的可复用模板
- **完整CRUD**: 支持 `INSERT`、`UPDATE` 及 `INSERT...RETURNING` 操作
- **PG兼容**: 严格遵循 PostgreSQL 参数限制

## 典型场景

- 从CSV/API批量导入百万级数据
- 高频指标/日志数据存储
- 需要返回ID的ETL数据管道
- 库存/订单系统的批量更新

## 安装
```bash
go get github.com/kaichao/gopkg/pgbulk
```

## 快速开始

## 1. 启动 PostgreSQL:
```sh
docker run -e POSTGRES_PASSWORD=secret -p 5432:5432 -d postgres:17.4
```

### 2. 基础用法:
```go

```
