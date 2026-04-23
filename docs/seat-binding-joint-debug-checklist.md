# Seat Binding 联调清单（D1-B5/B6）

## 前置条件

- 管理员账户可登录控制台。
- 后端已启用 `/api/seat_binding` 相关接口。

## 用例清单

- 打开 `/console/seat-binding` 页面成功，菜单可见。
- 点击“新增 Seat 绑定”打开弹窗。
- 输入合法参数后保存成功，列表刷新可见。
- 删除指定记录成功，列表中消失。
- 缺少 `user_id/seat_id/github_token` 时前端阻断并提示。
- 后端异常时显示错误消息（message）。

## 回归检查

- 原有 `/console/channel`、`/console/token` 页面行为无回归。
- 侧边栏展开/收起、路由跳转正常。