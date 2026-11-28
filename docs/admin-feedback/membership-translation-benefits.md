# 会员方案多语言权益列表

## 现象与调用链

Admin Web 打开会员方案的“设置多语言”时，请求 `GET /api/memberships/{membership_ulid}/translations`。Admin BFF 的 `GetMembershipTranslations` 将 `membership_ulid` 作为 `membership_id` 调用会员微服务 `gmbr.GetMembershipTranslations`，并直接返回其响应。

当前方案的 `features_json` 只有四条权益，但翻译响应中的 `translations[locale].feature_labels` 仍包含以前配置过、现已不在 `features_json` 中的键。原 Web 共用翻译组件会将这些历史键补进编辑列表，造成旧权益重现。本次 Web 调整为：会员方案只展示当前 `features_json` 生成的权益名称和分类；长标签与 key 自动换行。Web 保存时不删除服务端返回的历史键，避免未经接口约定改写持久化内容。

## 处理结论

按确认的范围，只在 Admin Web 的编辑列表隐藏不属于当前 `features_json` 的历史权益翻译键，会员微服务无需修改。`gmbr.GetMembershipTranslations` 可以继续返回这些历史键；Web 保存时也原样保留，不清理已有翻译数据。

2026-11-28：`adminweb` 提交 `1854975b` 已部署 dev / prod；两端容器健康，会员方案页面与新资源返回 HTTP 200。
