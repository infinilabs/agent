---
weight: 20
title: 认证
---

# 认证

## 登录 Token 与 API Token

登录 Token 用于登录 Agent 界面。

API Token 用于直接调用 Agent API，放入 `X-API-TOKEN` 请求头。它具有独立的权限和有效期，调用范围取决于创建时授予的权限。

## 获取登录 Token

### 自动生成与重新显示

未显式配置登录 Token 时，Agent 会自动生成，并保存至 keystore。正常情况下，后续启动会复用该 Token，无需每次重启后重新关联 Transfer。

首次生成时，Agent 会将登录 Token 打印到标准输出：

```text
 --  Login Token  -----------------
|   Token: <LOGIN_TOKEN>    |
 ----------------------------------
```

从 keystore 恢复已有 Token 时，默认不再打印。需要再次查看时，在启动前设置 `AGENT_SHOW_LOGIN_TOKEN=1` 或 `true`。该变量只控制显示，不会重新生成 Token。

macOS 或 Linux：

```bash
AGENT_SHOW_LOGIN_TOKEN=1 ./agent
```

Windows PowerShell：

```powershell
$env:AGENT_SHOW_LOGIN_TOKEN = "1"
.\agent.exe
```

如果 keystore 无法保存 Token，请检查启动警告；无法持久化的 Token 可能在重启后变化。

### 显式配置

可以通过环境变量 `AGENT_LOGIN_TOKEN` 指定登录 Token，也可以在 `agent.yml` 顶层配置：

```yaml
security:
  login_token: "<YOUR_LOGIN_TOKEN>"
```

环境变量优先于配置文件，两者均优先于 keystore 中保存的 Token。当前版本会在启动时打印显式配置的登录 Token。

## 创建 API Token

登录 Agent 后，在界面中创建 API Token：

1. 打开服务管理页面，点击右上角的设置按钮（齿轮图标）。
2. 从下拉菜单中选择“API Token”，进入“Token 管理”页面。
3. 点击“新增”，填写名称、选择所需权限，并按需填写描述，然后点击“创建”。
4. 创建成功后，复制并保存 API Token。关闭弹窗后，无法再通过管理界面查看完整 Token；如果丢失，需要重新创建。

{{% load-img "/img/api-token-management-zh.png" "" %}}

## 使用 API Token 调用 API

将 API Token 放入 `X-API-TOKEN` 请求头：

```bash
curl http://localhost:23000/elasticsearch/node/_discovery \
  -H "X-API-TOKEN: <API_TOKEN>"
```

受保护接口在认证失败时返回 HTTP 401；API Token 已通过认证但缺少接口要求的权限时，返回 HTTP 403。