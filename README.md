# jks-go

`jks-go` 是 JDK `keytool -importkeystore` 命令的**零成本替换方案**，无需安装 JDK，用于将 PKCS12 (.p12/.pfx) 或 PEM 格式的证书转换为 Java KeyStore (JKS) 格式。

基于开源库 [keystore-go](https://github.com/pavel-v-chernykh/keystore-go) 和 Go 标准加密库构建，纯静态编译，无运行时依赖。




## 与 keytool 的对比

只需将脚本中的 `keytool` 替换为 `jks-go`，参数原样保留：

```bash
# 原 keytool 命令
keytool -importkeystore \
  -srckeystore cert.p12 \
  -srcstoretype PKCS12 \
  -srcstorepass "password" \
  -destkeystore keystore.jks \
  -deststoretype JKS \
  -deststorepass "password"

# 替换为 jks-go（参数一模一样）
jks-go -importkeystore \
  -srckeystore cert.p12 \
  -srcstoretype PKCS12 \
  -srcstorepass "password" \
  -destkeystore keystore.jks \
  -deststoretype JKS \
  -deststorepass "password"
```

## 功能特性

- **PKCS12 → JKS**：完全兼容 keytool 的参数风格
- **PEM → JKS**：支持组合 PEM 文件和证书/私钥分离两种方式（扩展功能）
- **跨平台**：支持 Windows、Linux、macOS，覆盖 x86_64、i386、ARM64、ARMv7 共 8 种架构
- **纯静态编译**：`CGO_ENABLED=0`，无 glibc 依赖，单文件部署

## 安装方式

### 从 GitHub Release 下载（推荐）

从 [Releases](https://github.com/certd/jks-go/releases) 页面下载对应平台的预编译二进制。

| 平台 | 架构 | 文件 |
|---|---|---|
| Windows | x64 | `jks-go_windows_amd64.exe` |
| Windows | x86 | `jks-go_windows_386.exe` |
| Linux | x86_64 | `jks-go_linux_amd64` |
| Linux | x86 | `jks-go_linux_386` |
| Linux | ARM64 | `jks-go_linux_arm64` |
| Linux | ARMv7 | `jks-go_linux_arm_armv7` |
| macOS | Intel | `jks-go_darwin_amd64` |
| macOS | Apple Silicon | `jks-go_darwin_arm64` |

### go install

```bash
go install github.com/certd/jks-go@latest
```

### 源码编译

```bash
git clone https://github.com/certd/jks-go.git
cd jks-go

make build      # 编译当前平台
make release-all # 编译全部 8 个平台
make test        # 运行测试
```

## 快速开始

### PKCS12 → JKS（与 keytool 完全一致）

```bash
jks-go -importkeystore \
  -srckeystore cert.p12 \
  -srcstoretype PKCS12 \
  -srcstorepass "your-password" \
  -destkeystore keystore.jks \
  -deststorepass "your-password"
```

### PEM → JKS — 证书与私钥在同一文件

```bash
jks-go -importkeystore \
  -srckeystore bundle.pem \
  -srcstoretype PEM \
  -destkeystore keystore.jks \
  -deststorepass "your-password"
```

### PEM → JKS — 证书与私钥分离

```bash
jks-go -importkeystore \
  -srckeystore cert.pem \
  -srcstoretype PEM \
  -srckeyfile key.pem \
  -destkeystore keystore.jks \
  -deststorepass "your-password"
```

## 完整参数说明

| 参数 | keytool 原版？ | 必需 | 说明 |
|---|---|---|---|
| `-importkeystore` | 是 | 是 | 导入密钥库模式 |
| `-srckeystore` | 是 | 是 | 源文件路径 |
| `-srcstoretype` | 是 | 是 | 源类型：`PKCS12` 或 `PEM`（扩展） |
| `-srcstorepass` | 是 | 是* | 源密钥库密码 |
| `-srckeypass` | 是 | 否 | 源密钥密码（默认同 srcstorepass） |
| `-srckeyfile` | **新增** | 否 | PEM 私钥文件路径（证书与私钥分离时使用） |
| `-destkeystore` | 是 | 是 | 目标 JKS 文件路径 |
| `-deststoretype` | 是 | 否 | 目标类型，固定为 `JKS` |
| `-deststorepass` | 是 | 是 | 目标 JKS 密钥库密码 |
| `-destkeypass` | 是 | 否 | 目标密钥密码（默认同 deststorepass） |
| `-alias` | 否 | 否 | JKS 条目别名（默认从证书 CN 自动提取） |
| `-noprompt` | 是 | 否 | 静默模式，不显示确认提示 |

> *PKCS12 模式必需，PEM 模式可选

## 许可

MIT License - 详见 [LICENSE](LICENSE)

## 致谢

本项目基于以下开源库构建：

- [keystore-go](https://github.com/pavel-v-chernykh/keystore-go) - Go 实现的 Java KeyStore 编解码库
- [golang.org/x/crypto](https://golang.org/x/crypto) - Go 官方扩展加密库（PKCS12 支持）
