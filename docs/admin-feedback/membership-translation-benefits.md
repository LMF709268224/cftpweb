# 会员方案多语言权益列表

## 现象与调用链

Admin Web 打开会员方案的“设置多语言”时，请求 `GET /api/memberships/{membership_ulid}/translations`。Admin BFF 的 `GetMembershipTranslations` 将 `membership_ulid` 作为 `membership_id` 调用会员微服务 `gmbr.GetMembershipTranslations`，并直接返回其响应。

当前方案的 `features_json` 只有四条权益，但翻译响应中的 `translations[locale].feature_labels` 仍包含以前配置过、现已不在 `features_json` 中的键。原 Web 共用翻译组件会将这些历史键补进编辑列表，造成旧权益重现。本次 Web 调整为：会员方案只展示当前 `features_json` 生成的权益名称和分类；长标签与 key 自动换行。Web 保存时不删除服务端返回的历史键，避免未经接口约定改写持久化内容。

## 需会员微服务调整

请检查 `gmbr.GetMembershipTranslations`（Admin HTTP 入口如上），按请求的 `membership_id` 取得当前方案的 `features_json`，让响应中每个 locale 的 `feature_labels` 和 `feature_categories` 只包含当前权益与分类对应的键。名称、描述、适用人群等其他翻译字段继续按原接口返回。若 `gmbr.SetMembershipTranslations` 对已移除权益的键有保留要求，请明确读写契约；目前 Web 会原样保留这些返回值，不自行做数据清理。

这属于响应数据与当前方案配置的一致性问题；仅隐藏 Web 编辑项不会修复微服务返回的历史数据。当前 Web 代码已完成，尚未部署。
