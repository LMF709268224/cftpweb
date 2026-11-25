# Admin 客户反馈清单

统一记录本轮 Admin 客户反馈，按页面分文件，后续清单均放在此目录。全部反馈看完后，再以这些清单一起总结。

## 目录

| 反馈 | 清单 | 对应代码提交 | 当前进度 |
| --- | --- | --- | --- |
| Dashboard | [工作台](dashboard.md) | `089bd82d` | 已部署 dev / prod；统计接口及口径缺口仍在清单 |
| User Management 1/3 | [用户管理（1/3）](user-management-1.md) | `46ea9075` | 已完成部分已部署 dev / prod；筛选、操作及权限待办仍在清单 |
| User Management 2/3 | [用户管理（2/3）](user-management-2.md) | `e0e35940` | 最近登录、员工 MFA、行操作入口已部署 dev / prod；账号操作与业务口径缺口仍在清单 |
| User Management 3/3 | [权限矩阵与审计](user-management-3.md) | `fd854b99` | 明确的查看行为审计已部署 dev / prod；新角色、矩阵冲突与可靠送达未实施 |
| 前期改动复查 | [禁止保底方案复查](no-fallback-review.md) | `fd854b99` | 已移除的缺数据保底处理已部署 dev / prod；接口输出缺口仍在清单 |
| Course Config 1/2、2/2 | [课程配置与编辑器](course-config.md) | `a6c8b0c3`；后续见中文复核 | 已完成的部分编辑和 PDF 预览已部署 dev / prod；统一工作台、迁移及退休规则待确认 |
| 中文补充说明复核 | [逐页复核及大项确认清单](chinese-notes-review.md) | 与小修同次提交 | 小修已部署 dev / prod；较大流程改动未实施，等用户确定 |
| 资源授权 / 正式服报名费 | [2026-11-25 线上排查及发布](2026-11-25-resource-access-and-enrollment-fee.md) | 原修复 `c9730bd1`；prod 单独发布 `bbafddfe` | 报名费修复已随主分支部署 dev / prod；dev 会员记录与 Casdoor 角色不一致，尚未改动授权 |
| 考生端阶段列表状态 | [学习状态与考试预约](2026-11-25-candidate-pathway-status.md) | `cd320be7` | 2026-11-25 已部署 dev / prod 的 candbff、candweb，健康及线上页面回归通过；“未开始”仍需 GLMS 提供权威字段 |
| 统一发布及公开会员入口 | [2026-11-25 发布记录](2026-11-25-admin-release.md) | Admin `d1ab3434`；会员入口 `2a981d01` | 记录 dev / prod 发布与回归，以及旧分支合并的撤回 |

## 维护方式

- 每份清单记录已完成项、暂缓项、需确认的问题及所需接口。
- 新反馈继续在本目录添加清单，并更新此索引。
- 后续需求覆盖前一版展示时，更新相关清单，避免最终总结出现互相矛盾的状态。
- 实际部署后更新进度，区分“代码已完成”与“环境已上线”。
- 2026-11-25 起按用户要求优先小修；流程调整及较大重构集中在[中文复核清单](chinese-notes-review.md)说明范围、影响和待确认项，未经用户确定不实施。
