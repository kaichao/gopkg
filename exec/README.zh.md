# exec

[English](README.md) | 中文

跨环境命令执行工具包，支持本地与SSH远程执行，完整捕获输出流。

## 功能特性

- **统一接口**: 本地与远程执行使用相同API
- **输出捕获**: 同步处理标准输出/错误流
- **超时控制**: 可配置执行超时与进程清理
- **SSH支持**: 密钥与密码认证

## 典型场景

- 服务器集群批量操作
- CI/CD 流水线任务执行
- 分布式系统监控
- 批量日志收集分析

## 安装
```sh
go get github.com/kaichao/gopkg/exec
```

## 快速开始
### 本地执行
```go
code, stdout, stderr, err := exec.RunReturnAll("ls -l", 10) // 10s timeout
fmt.Printf("Exit: %d\nOutput:\n%s\nError:\n%s", code, stdout, stderr)
```

### SSH远程执行
```go
config := exec.SSHConfig{
    User:    "admin",
    Host:    "192.168.1.100",
    Port:    22,
    KeyPath: "/path/to/private_key",
}

code, out, errOut, err := exec.RunSSHCommand(config, "docker ps", 30) // 30s timeout
```

## API参考
### RunReturnAll
```go
func RunReturnAll(command string, timeout int) (int, string, string, error)
```

### RunSSHCommand
```go
func RunSSHCommand(config SSHConfig, command string, timeout int) (int, string, string, error)
```

### SSHConfig
```go
type SSHConfig struct {
    User     string // SSH username
    Host     string // Server IP/hostname
    Port     int    // SSH port (default: 22)
    KeyPath  string // Path to private key (default: ~/.ssh/id_rsa)
    Password string // Password authentication
}
```

## 高级用法
### 自定义信号处理
```go
// Terminate process group on SIGINT
signal.Notify(sigChan, syscall.SIGINT)
go func() {
    <-sigChan
    syscall.Kill(-pid, syscall.SIGTERM)
}()
```

## 测试

- 创建测试用的singularity镜像
```sh
mkdir -p ~/singularity
docker save debian:12-slim -o debian.tar
singularity build ~/singularity/debian.sif docker-archive://debian.tar
```
