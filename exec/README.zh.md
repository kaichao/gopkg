# exec

[English](README.md) | 中文

## 目录
1. [功能特性](#功能特性)
2. [典型场景](#典型场景)
3. [安装指南](#安装指南)
4. [快速开始](#快速开始)
5. [API参考](#api参考)
6. [安全注意事项](#安全注意事项)
7. [最佳实践](#最佳实践)
8. [高级用法](#高级用法)
9. [常见问题](#常见问题)
10. [测试方法](#测试方法)

## 功能特性

跨环境命令执行工具包，支持以下功能：

- **统一接口**: 本地、远程SSH和容器内执行使用相同API
- **完整输出捕获**: 同步获取标准输出、错误流和退出码
- **灵活超时控制**: 支持命令级和连接级超时设置
- **多认证方式**: SSH支持密钥、密码和代理转发
- **进程管理**: 支持后台进程执行(PID返回)和进程组管理

## 典型场景

### 服务器批量运维
```go
// 批量执行服务器维护命令
for _, host := range servers {
    config.Host = host
    exec.RunSSHCommand(config, "apt update && apt upgrade -y", 300)
}
```

### CI/CD流水线
```go
// 部署后验证
if code, out, _ := exec.RunReturnAll("curl -sSf http://localhost:8080/health", 10); code != 0 {
    log.Fatal("服务健康检查失败")
}
```

### 容器管理
```go
// 在容器内执行诊断命令
exec.RunSSHCommand(config, "singularity exec app.sif df -h", 30)
```

## 安装指南

```sh
go get github.com/kaichao/gopkg/exec
```

## 快速开始

### 本地命令执行
```go
code, stdout, stderr, err := exec.RunReturnAll("ls -l /tmp", 10)
if err != nil {
    log.Printf("执行失败: %v\n输出: %s\n错误: %s", err, stdout, stderr)
}
```

### SSH远程执行
```go
config := exec.SSHConfig{
    User:    "admin",
    Host:    "10.0.0.1", 
    KeyPath: "/home/user/.ssh/id_rsa",
}

// 执行并获取完整输出
code, out, errOut, err := exec.RunSSHCommand(config, "docker ps -a", 30)
```

### 容器内执行
```go
// 在Singularity容器中运行命令
cmd := "singularity exec /images/debian.sif apt-get update"
RunSSHCommand(config, cmd, 60)
```

## API参考

### 核心方法
```go
// 本地执行
func RunReturnAll(command string, timeout int) (code int, stdout string, stderr string, err error)

// SSH执行 
func RunSSHCommand(config SSHConfig, command string, timeout int) (code int, stdout string, stderr string, err error)
```

### SSHConfig结构体
```go
type SSHConfig struct {
    User       string // 必须
    Host       string // 必须
    Port       int    // 默认22
    KeyPath    string // 优先于密码
    Password   string 
    Timeout    int    // 连接超时(秒)
    Background bool   // 后台运行命令
}
```

## 安全注意事项

1. **认证安全**
   - SSH私钥权限应设为600
   - 避免在代码中硬编码密码

2. **命令注入防护**
   ```go
   // 不安全
   cmd := fmt.Sprintf("ls %s", userInput)
   
   // 安全做法
   cmd := fmt.Sprintf("ls %s", filepath.Clean(userInput))
   ```

3. **日志记录**
   - 记录关键操作的元数据（用户、命令、时间）
   - 避免记录敏感输出

## 最佳实践

### 连接复用
```go
var client *ssh.Client

func getClient(config SSHConfig) (*ssh.Client, error) {
    if client == nil {
        // 初始化连接...
    }
    return client, nil
}
```

### 错误处理
```go
// 检查特定错误类型
if errors.Is(err, exec.ErrTimeout) {
    // 处理超时
}
```

### 资源清理
```go
defer func() {
    if cmd.Process != nil {
        cmd.Process.Kill()
    }
}()
```

## 高级用法

### 信号处理
```go
// 终止整个进程组
syscall.Kill(-pid, syscall.SIGTERM)
```

### 后台进程管理
```go
// 使用Background模式运行后台进程
config := exec.SSHConfig{
    Host: "10.0.0.1",
    User: "admin",
    Background: true, // 启用后台执行模式
    // ... 其他配置
}

// 立即返回PID，进程继续在后台运行
_, pid, _, _ := exec.RunSSHCommand(config, "long-running-command", 0)
```

## 常见问题

### 连接超时
- 检查网络防火墙设置
- 增加连接超时时间：
  ```go
  config.Timeout = 30 // 秒
  ```

### 输出截断
- 使用缓冲区或临时文件处理大输出
- 设置合理的执行超时

## 测试方法

1. 准备测试用的singularity镜像
```sh
mkdir -p ~/singularity
docker save debian:12-slim -o debian.tar
singularity build ~/singularity/debian.sif docker-archive://debian.tar
```

2. 运行单元测试：
```sh
cd exec && go test -v
