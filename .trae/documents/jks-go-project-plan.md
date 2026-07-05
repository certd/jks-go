# jks-go 项目开发计划

## 一、项目概要

**目标**：基于开源库 `github.com/pavel-v-chernykh/keystore-go` (v4) 和 `golang.org/x/crypto/pkcs12`，开发一个 Go 命令行工具 `jks-go`，**完全复刻 keytool 的 `-importkeystore` 命令参数风格**，零学习成本替换，实现 **PKCS12 (.p12/.pfx) → JKS** 和 **PEM → JKS** 的证书格式转换。

用户只需将脚本中的 `keytool` 改为 `jks-go`，原有参数原样保留即可。

**Module 路径**：`github.com/certd/jks-go`
**二进制名称**：`jks-go`
**License**：MIT

---

## 二、项目目录结构

```
jks-go/
├── main.go                  # CLI 入口，参数解析
├── convert.go               # 核心转换逻辑（PKCS12→JKS / PEM→JKS）
├── convert_test.go          # 单元测试
├── go.mod                   # Go module 定义
├── go.sum                   # 依赖锁定
├── Makefile                 # 构建、测试、lint 脚本
├── LICENSE                  # MIT 许可证
├── README.md                # 项目文档
├── .gitignore               # Git 忽略规则
└── .github/
    └── workflows/
        └── release.yml      # GitHub Actions 自动构建 & 发布
```

项目结构极简，全部逻辑放在根目录（少于 5 个 .go 文件），无需 `cmd/` 子目录或 `internal/` 包。

---

## 三、核心依赖

| 依赖 | 版本 | 用途 |
|---|---|---|
| `github.com/pavel-v-chernykh/keystore-go/v4` | v4.3.0+ | JKS 文件创建与写入 |
| `golang.org/x/crypto` | latest | PKCS12 解析 |
| Go 标准库 (`crypto/x509`, `crypto/rsa`, `crypto/ecdsa`, `encoding/pem`, `flag`, `os`, `fmt` 等) | Go 1.21+ | PEM 解析、CLI、文件 I/O |

---

## 四、CLI 设计（完全复刻 keytool 参数风格）

### 4.1 设计原则

**零学习成本替换**：用户只需将脚本中的 `keytool` 替换为 `jks-go`，其余参数原样保留即可正常运行。对于 keytool 原生不支持的场景（PEM 源），新增最小化的补充参数，不影响原有 keytool 参数的兼容性。

### 4.2 命令格式

```
jks-go -importkeystore \
       -srckeystore <path> \
       -srcstoretype <PKCS12|PEM> \
       -srcstorepass <password> \
       -destkeystore <path> \
       -deststoretype JKS \
       -deststorepass <password> \
       [-destkeypass <password>] \
       [-srckeypass <password>] \
       [-srckeyfile <path>] \
       [-alias <name>] \
       [-noprompt]
```

### 4.3 参数说明

| 参数 | keytool 原版？ | 必需 | 默认值 | 说明 |
|---|---|---|---|---|
| `-importkeystore` | 是 | 是 | — | 导入密钥库模式（与 keytool 一致） |
| `-srckeystore` | 是 | 是 | — | 源文件路径（.p12/.pfx 或 .pem） |
| `-srcstoretype` | 是 | 是 | — | 源类型：`PKCS12` 或 `PEM`（扩展支持） |
| `-srcstorepass` | 是 | 是 | — | 源密钥库密码（PKCS12 解密密码） |
| `-srckeypass` | 是 | 否 | 同 `-srcstorepass` | 源密钥密码（仅加密 PEM 私钥时使用） |
| `-srckeyfile` | **否（新增）** | PEM模式可选 | — | PEM 私钥文件路径。仅当证书和私钥分离为两个文件时需要 |
| `-destkeystore` | 是 | 是 | — | 目标 JKS 文件路径 |
| `-deststoretype` | 是 | 否 | `JKS` | 目标类型，固定为 `JKS`（可省略） |
| `-deststorepass` | 是 | 是 | — | 目标 JKS 密钥库密码 |
| `-destkeypass` | 是 | 否 | 同 `-deststorepass` | 目标密钥密码（默认与密钥库密码相同） |
| `-alias` | 否 | 否 | 从证书 CN 提取 | JKS 条目别名 |
| `-noprompt` | 是 | 否 | — | 静默模式，不显示确认提示（兼容 keytool 脚本场景） |

### 4.4 与 keytool 的兼容性

用户原来的 keytool 命令：
```bash
keytool -importkeystore \
  -srckeystore cert.p12 \
  -srcstoretype PKCS12 \
  -srcstorepass "password" \
  -destkeystore keystore.jks \
  -deststoretype JKS \
  -deststorepass "password"
```

替换为 jks-go（**参数一模一样，仅替换二进制名称**）：
```bash
jks-go -importkeystore \
  -srckeystore cert.p12 \
  -srcstoretype PKCS12 \
  -srcstorepass "password" \
  -destkeystore keystore.jks \
  -deststoretype JKS \
  -deststorepass "password"
```

### 4.5 PEM 模式（jks-go 扩展功能）

PEM 模式兼容两种输入方式：

**方式一：证书与私钥在同一文件**（推荐）
```bash
jks-go -importkeystore \
  -srckeystore bundle.pem \
  -srcstoretype PEM \
  -destkeystore keystore.jks \
  -deststorepass "password"
```
自动从 `bundle.pem` 中解析所有 PEM blocks，识别证书和私钥。

**方式二：证书与私钥分离**
```bash
jks-go -importkeystore \
  -srckeystore cert.pem \
  -srcstoretype PEM \
  -srckeyfile key.pem \
  -destkeystore keystore.jks \
  -deststorepass "password"
```
> `-srckeyfile` 为 jks-go 新增参数，keytool 原生不支持此能力。

### 4.6 错误处理

- 源文件不存在 → 明确报错并退出码 1
- 密码错误（PKCS12 解密失败）→ 报 "incorrect password" 并退出码 1
- `-srcstoretype` 值非法 → 提示仅支持 `PKCS12`/`PEM` 并退出码 2
- PEM 解析失败 → 报具体解析错误并退出码 1
- JKS 写入失败 → 报 I/O 错误并退出码 1
- 必需参数缺失 → 显示用法帮助并退出码 2

---

## 五、核心转换逻辑

### 5.1 PKCS12 → JKS 流程

```
1. 读取 PKCS12 文件 → []byte
2. 调用 pkcs12.Decode(pfxData, srcPass) 获取：
   - privateKey interface{} (RSA/ECDSA 私钥)
   - certificate *x509.Certificate
3. 将私钥序列化为 PKCS8 DER 格式 (x509.MarshalPKCS8PrivateKey)
4. 将证书序列化为 DER 格式 (certificate.Raw)
5. 构造 keystore.PrivateKeyEntry:
   - CreationTime: time.Now()
   - PrivateKey: PKCS8 DER bytes
   - CertificateChain: []keystore.Certificate{{Type: "X.509", Content: cert.Raw}}
6. ks := keystore.New()
7. ks.SetPrivateKeyEntry(alias, entry, dstPass)
8. ks.Store(outputFile, dstPass)
```

### 5.2 PEM → JKS 流程

```
1. 读取证书 PEM 文件 → 解析 PEM block → x509.ParseCertificate
2. 读取私钥 PEM 文件 → 解析 PEM block → x509.ParsePKCS8PrivateKey (或 PKCS1/EC)
3. 将私钥序列化为 PKCS8 DER 格式
4. 构造 keystore.PrivateKeyEntry（同上）
5. ks.SetPrivateKeyEntry → ks.Store
```

**PEM 私钥格式兼容**：支持 PKCS#1 (RSA)、PKCS#8、SEC 1 (EC) 三种常见格式，依次尝试解析。

### 5.3 Alias 自动提取

如果用户未指定 `--alias`，从证书的 `Subject.CommonName` 提取。若 CN 为空或解析失败，回退为 `"certificate"`。

---

## 六、构建与发布

### 6.1 Makefile

```makefile
.PHONY: build test lint clean release-all

BINARY=jks-go
BUILD_DIR=build

build:
	go build -o $(BINARY) .

test:
	go test -v ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BINARY) $(BUILD_DIR)

release-all:
	$(MAKE) release GOOS=windows GOARCH=amd64 EXT=.exe
	$(MAKE) release GOOS=windows GOARCH=386 EXT=.exe
	$(MAKE) release GOOS=linux GOARCH=amd64
	$(MAKE) release GOOS=linux GOARCH=386
	$(MAKE) release GOOS=linux GOARCH=arm64
	$(MAKE) release GOOS=linux GOARCH=arm GOARM=7
	$(MAKE) release GOOS=darwin GOARCH=amd64
	$(MAKE) release GOOS=darwin GOARCH=arm64

release:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) \
		go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY)_$(GOOS)_$(GOARCH)$(if $(GOARM),_armv$(GOARM),)$(EXT) .
```

- `make build` — 编译当前平台二进制
- `make test` — 运行测试
- `make lint` — 代码静态检查
- `make release-all` — 一键编译全部 8 个平台

### 6.2 跨平台编译矩阵

| GOOS | GOARCH | GOARM | 产物文件名 | 适用场景 |
|---|---|---|---|---|
| windows | amd64 | — | `jks-go_windows_amd64.exe` | Windows x64 |
| windows | 386 | — | `jks-go_windows_386.exe` | Windows 32位 |
| linux | amd64 | — | `jks-go_linux_amd64` | Linux x86_64 |
| linux | 386 | — | `jks-go_linux_386` | Linux x86 32位 |
| linux | arm64 | — | `jks-go_linux_arm64` | ARM64 服务器 |
| linux | arm | 7 | `jks-go_linux_arm_armv7` | ARMv7（树莓派 3/4/Zero 2W 等） |
| darwin | amd64 | — | `jks-go_darwin_amd64` | macOS Intel |
| darwin | arm64 | — | `jks-go_darwin_arm64` | macOS Apple Silicon |

构建参数：
- `CGO_ENABLED=0` — 禁用 CGo，生成纯静态二进制，无 glibc 依赖
- `-ldflags="-s -w"` — 去除符号表和调试信息，减小二进制体积

### 6.3 GitHub Actions 自动发布 (.github/workflows/release.yml)

当推送 `v*` 格式的 tag（如 `v1.0.0`）时触发：
1. 检出代码
2. 设置 Go 1.21+
3. 运行 `go test ./...`
4. 按 Go 矩阵并行编译全部 8 个平台/架构组合（使用 `CGO_ENABLED=0`）
5. 创建 GitHub Release 并上传所有编译产物（含 SHA256 校验和文件）

---

## 七、测试计划

### 7.1 单元测试 (convert_test.go)

| 测试用例 | 说明 |
|---|---|
| `TestPKCS12ToJKS` | 使用预生成的测试 .p12 文件，验证转换成功且 JKS 可被 keystore 库重新读取 |
| `TestPEMToJKS` | 使用预生成的测试 .pem 证书+私钥，验证转换成功 |
| `TestAutoAlias` | 验证从证书 CN 自动提取别名 |
| `TestInvalidPassword` | 传入错误密码，验证报错 |
| `TestMissingFile` | 源文件不存在，验证报错 |

### 7.2 集成验证

编译完成后，用真实的 `keytool` 命令验证输出的 JKS 文件可被 JDK 正确读取：
```bash
keytool -list -keystore output.jks -storepass password
```

---

## 八、README.md 内容大纲

1. **项目简介** — 一句话描述：`jks-go` 是 keytool `-importkeystore` 的零成本替换方案
2. **与 keytool 的对比** — 直接替换对照表：`keytool ...` → `jks-go ...`，参数一模一样
3. **功能特性** — PKCS12→JKS、PEM→JKS（扩展）、跨平台 8 种架构、纯静态编译无依赖
4. **安装方式**
   - 从 GitHub Release 下载预编译二进制（推荐）
   - `go install github.com/certd/jks-go@latest`
   - 源码编译 `make build`
5. **快速开始**
   - 替换 keytool（参数不变）：PKCS12 → JKS 示例
   - PEM 模式（组合文件 / 分离文件）：两种方式示例
6. **完整参数说明** — 表格列出所有参数，标注 keytool 原版 vs 新增
7. **构建说明** — `make build` / `make release-all` / `make test`
8. **License** — MIT

---

## 九、LICENSE

使用标准 **MIT License**，版权归属 `certd`。

---

## 十、实施步骤

| 步骤 | 任务 | 产出文件 |
|---|---|---|
| 1 | 初始化 Go module，安装依赖 | `go.mod`, `go.sum` |
| 2 | 实现核心转换逻辑 | `convert.go` |
| 3 | 实现 CLI 入口 | `main.go` |
| 4 | 编写单元测试 | `convert_test.go` |
| 5 | 创建 Makefile | `Makefile` |
| 6 | 创建 .gitignore | `.gitignore` |
| 7 | 创建 MIT LICENSE | `LICENSE` |
| 8 | 编写 README.md | `README.md` |
| 9 | 创建 GitHub Actions 发布工作流 | `.github/workflows/release.yml` |
| 10 | 本地验证：编译、测试、lint | — |
| 11 | 集成验证：用 keytool 检验输出文件 | — |

---

## 十一、假设与决策

1. **Go 版本**：最低要求 Go 1.21（当前主流稳定版本，支持泛型等特性）
2. **CLI 框架**：使用标准库 `flag` 包，不引入 `cobra` 等第三方 CLI 库（项目足够简单）
3. **参数风格**：完全复刻 keytool 的 `-importkeystore` 参数名（单横线前缀），用户可零修改替换
4. **PEM 私钥格式**：私钥必须为未加密的 PEM 格式（如需加密私钥支持，后续版本扩展）
5. **PKCS12 证书链**：使用 `pkcs12.Decode`（适合单证书+单私钥场景），如需支持完整证书链，后续版本可切换到 `pkcs12.ToPEM`
6. **没有配置文件**：所有参数通过命令行传递，保持简单
7. **单文件输出**：一次只输出一个 JKS 文件，不产生中间文件
8. **-srckeyfile**：jks-go 唯一新增的参数，用于 PEM 模式下证书与私钥分离的场景
