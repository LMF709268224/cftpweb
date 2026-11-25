# 资源授权与正式服报名费排查（2026-11-25）

初次排查只读核对 dev / prod 实际配置、线上静态资源和已有代码。用户随后要求先更新 prod 管理端，现已单独发布报名费修复，详情见本文发布记录。会员记录、Casdoor 角色和商品价格未修改。

## 1. dev 无会员记录的考生可以打开会员资源

### 已确认原因

会员服务中的会员记录与 Casdoor 角色不一致。该账号在会员服务中没有记录，但被直接列入 Casdoor 的附属会员角色；Casdoor 对资源的新路径确实返回允许访问。

### 实际数据与请求

- 考生业务 ID：`01M2SJKF2YJFTEQHWAW3VVGN4W`，通过 Casdoor `get-user-ids` 从截图账号对应的 UUID 解析，未将 UUID 当作业务 ID。
- 文件 ID：`01KWXSTE37B612WBVQ1EQWAYZS`。
- 文件所属资源包 ID：`01KWXSTE2Z5M8S3Q74AAXSCY55`，标题 `Insights`，状态 `Active`。
- 数据库中的当前 `respath`：`/res-packages/membership/insights`，与用户修改后的路径一致。
- dev `gmbr.user_memberships` 按该考生业务 ID 查询：返回空数组，包含历史在内也没有会员记录。
- Casdoor `cftp-org/role_membership_affiliate` 的 `users` 数组直接包含该账号；使用 JSON 数组精确匹配确认，角色已启用。
- 已启用的 `permission_membership_resource_packages_read` 向附属会员等角色授予 `/res-packages/membership` 的 `Read` 权限。
- 实际鉴权：`POST /api/enforce?enforcerId=cftp-org/enforcer_cftp`，请求体如下（服务凭据未记录）：

```json
["01M2SJKF2YJFTEQHWAW3VVGN4W", "/res-packages/membership/insights", "Read:hi"]
```

- 实际响应：`status="ok"`、`data=[true]`，执行模型/适配器为 `cftp-org/model_v3/cftp-org/adapter_modelv3`。
- 父路径 `/res-packages/membership` 的相同鉴权也返回 `true`。

### 代码核对

- `candbff/handler/resource_pack.go`：文件预览调用 `GetResourcePackFileViewURL`，传入当前登录考生业务 ID 和文件 ID。
- 微服务 `glms/server/grpc_resource_pack.go`：先根据文件找到所属资源包，再校验访问权限。
- 微服务 `glms/server/permissions.go`：按资源包 ID 从数据库读取当前 `respath`，交给 Casdoor 执行 `Read:hi` 鉴权。因此本例已经能用新路径重新得到放行结果，不能归因为仍在使用旧资源路径。

### 后续处理及边界

- 需要核实这个附属会员角色的分配来源；当前证据不能确定是手动配置、历史测试数据还是同步遗留，不能断言是哪一种。
- 若该账号应为非会员，应通过受支持的角色管理入口移除这项不符合实际会员状态的授权，并复查同一鉴权返回 `false`，资源卡片仅展示缩略图、内容接口拒绝访问。
- 不通过前端会员标签覆盖 Casdoor 的鉴权结果，不为账号补造会员记录，也不直接修改权限数据库。
- 若要将“没有有效会员记录就不能看会员资源”升级为独立于角色的后端强制规则，需要微服务侧明确访问规则和会员角色同步责任，作为单独的流程/权限调整确认后处理。

## 2. prod 已选认证仍不能保存 Pipeline 报名费

### 已确认原因

发布前，正式服 `adminweb` 运行旧版 `PricingEditor`：编辑器绑定和序列化的是 `pipeline_id`，而同一页面的保存校验读取 `pipeline_gpath`。因此下拉框有选中值，校验依然认为未选择关联认证；报错发生在前端提交前。

### 发布前的线上构建证据

| 环境 | 入口脚本 | 商品页面脚本 | 编辑器字段 | 校验字段 |
| --- | --- | --- | --- | --- |
| dev | `index-CDIdInmt.js` | `BundlesPage-BWKblF4J.js` | `pipeline_gpath` | `pipeline_gpath` |
| prod | `index-C2mOt8_4.js` | `BundlesPage-BrLn7ewF.js` | `pipeline_id` | `pipeline_gpath` |

- 以上脚本从两个环境当前返回的页面及其引用资源读取，说明正式服部署产物本身仍有问题，不只是某个浏览器缓存。
- 微服务的商品定价契约使用 `pricing_json.pipelines[].pipeline_gpath`；应按该契约保存，不能增加 `pipeline_id` 与 `pipeline_gpath` 之间的猜测兼容。
- 已有修复提交：`c9730bd1`（2026-11-04），`Fix admin pipeline enrollment fee GPath binding`，统一编辑器的解析、绑定、去重、新增和序列化字段。
- 本次运行已有浏览器回归 `pipeline enrollment fee keeps its GPath when saved`：1 项通过，验证保存请求保留所选认证 GPath 及 Stripe Product/Price ID。测试使用本地模拟接口，没有向正式服提交价格配置；这不等于已验证截图中 Stripe 价格的有效性。

### 发布记录（2026-11-25）

- dev 已包含原修复；本次按用户要求单独更新 prod 的 `adminweb`。
- 从正式服原容器二进制的 Go 构建信息确认原版本为 `7301f50c`，源码未修改。
- 以该版本建立发布分支 `release/prod-admin-enrollment-fee-20261125`，只摘取 `c9730bd1`；发布提交为 `bbafddfe`。相对原版本，仅修改 `PricingEditor.vue`，并增加对应浏览器回归测试。
- 使用仓库原有脚本：`bash -l /home/ubuntu/cftpweb-prod-admin-enrollment-fee-20261125/docker_abc.sh --env prod adminweb`。部署脚本及 Compose 定义与主分支一致，复用原 `.env.prod`，未另写部署流程。
- 之前 `admin-feedback` 中的 Dashboard、用户管理、课程编辑器等改动仍未随此次发布上线。dev、prod 考生端和 prod `adminbff` 的容器 ID / 镜像与发布前一致。
- 发布前检查：Vue 类型检查、E2E 类型检查、商品配置 6 项浏览器测试均通过。服务端原脚本完成前端生产构建、压缩、Go 编译及镜像构建。
- 发布后检查：`cftp-prod-adminweb` 为 `running / healthy`，`/bundles` 返回 HTTP 200；线上 HTML、入口及商品页面脚本与本次构建文件逐字节一致。
- 线上新入口：`/assets/index-LHF3Cwxz.js`；商品页面：`BundlesPage-CvgS06T4.js`；商品脚本 SHA-256：`2af0a6d8d0dc730a57d3df934bfa72463f317ae88de98ccd059ccc02c71a7f22`。
- 使用正式服真实页面静态资源执行报名费保存回归：1 项通过。所有 `/api/**` 请求均由测试拦截并模拟返回，验证前端提交的 GPath 与价格 ID，未向正式服写入商品价格，也不宣称已验证截图中 Stripe 价格的有效性。
- 新镜像：`sha256:6d25c00a0cd0dbea3a4223a70d51667f42c18892181a3a30b9f5e1cb4c7f2470`。
- 原镜像已保留为 `localhost/adminweb:prod-before-enrollment-fee-20261125`，镜像 ID：`sha256:8eb2b7e7ba584298f711c8137e0de2d5c73ff464f1e8b62513ae00df4e2c8080`。
