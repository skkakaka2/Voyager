# Voyager

Voyager 是一个基于 Wails v2、Go、Vue 3、TypeScript 与 Element Plus 的跨平台远端文件管理工具。

当前阶段完成 M0 基础工程骨架，并已打通 WebDAV、FTP、SMB 的连接测试、目录浏览和基础文件操作。传输任务已支持进度、速度、剩余时间估算、取消，以及上传/下载覆盖确认。

## 开发环境

- Go 1.23+
- Node.js 24+
- npm 11+
- Wails CLI v2.12.0

本仓库已把 Wails CLI 安装到 `.bin/wails`，不需要全局安装也可以运行以下命令：

```bash
.bin/wails doctor
.bin/wails dev
```

Linux 需要 Wails 桌面依赖。当前机器 `wails doctor` 输出显示 `libwebkit` 缺失，Ubuntu/Debian 系发行版通常需要安装 WebKitGTK 开发包后才能运行 `wails dev` 或完整 `wails build`。具体包名以 `wails doctor` 输出和当前发行版仓库为准。

## 常用命令

后端测试：

```bash
GOCACHE="$(pwd)/.cache/go-build" go test ./...
```

前端安装依赖：

```bash
cd frontend
npm install
```

前端类型检查与构建：

```bash
cd frontend
npm run build
```

Wails 构建：

```bash
.bin/wails build -tags webkit2_41 -nopackage
```

如果 Wails 构建报 `webkit2gtk-4.0` 或 `libwebkit` 缺失，先确认系统是否安装了 WebKitGTK 开发包。Ubuntu 25.10 这类只提供 `libwebkit2gtk-4.1-dev` 的环境，仓库已在 `wails.json` 中配置 `build:tags` 为 `webkit2_41`。

## 当前范围

- 已支持：连接配置新增、编辑、删除的后端服务与前端入口。
- 已支持：连接配置落盘到用户配置目录，配置文件不保存明文密码。
- 已支持：系统钥匙串保存连接密码，钥匙串不可用时连接仍可保存，并在 UI 提示密码不会保存。
- 已支持：WebDAV 连接测试、目录列表、新建文件夹、重命名、删除、上传、下载。
- 已支持：FTP 适配器基础接入，可使用同一套文件浏览与操作接口；仍需接入真实 FTP 服务做手工兼容性验收。
- 已支持：SMB 适配器基础接入，可使用同一套文件浏览与操作接口；共享名留空时可先浏览服务器共享列表，再进入具体共享执行文件操作。
- 已支持：上传/下载开始后立即记录传输任务，任务面板展示中文状态、字节级进度、速度、剩余时间、完成时间和失败原因，并可取消进行中的任务。
- 已支持：上传本地文件前基于当前目录做同名覆盖确认，下载到本地已有同名文件前也会提示覆盖确认。
- 待实现：断点续传、传输限速、传输重试、批量操作和文件夹下载等增强能力。
