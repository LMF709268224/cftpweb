# Bundle 发布与 pricing_json.unlocks 问题说明

## 背景

admin 商品配置里，我们有一个认证 Bundle 商品，原始结构大概是：

```json
{
  "items_json": [
    {
      "ref_ulid": "认证A_pipeline_ulid",
      "item_type": "pipeline"
    }
  ],
  "pricing_json": {
    "units": [
      {
        "unit_id": "认证A_unit_id",
        "access": {
          "stripe_price_id": "price_xxx",
          "stripe_product_id": "prod_xxx"
        },
        "retake": {
          "stripe_price_id": "price_xxx",
          "stripe_product_id": "prod_xxx"
        }
      }
    ],
    "unlocks": {},
    "memberships": [],
    "qual_reviews": []
  }
}
```

现在的业务诉求是：

1. 克隆旧商品 ABundle。
2. 把克隆出来的新商品 BBundle 里的认证引用从“认证 A”替换成“认证 B”。
3. 替换时只应该改 `items_json[0].ref_ulid`，也就是 `认证A_pipeline_ulid` -> `认证B_pipeline_ulid`。
4. 替换时只应该同步改 `pricing_json.units[].unit_id`，也就是 `认证A_unit_id` -> `认证B_unit_id`。
5. 其它价格结构保持一致。
6. 商品发布后，候选人应该可以直接购买 Bundle，不应该先支付“解锁订单”，再支付“购买订单”。

## 现在遇到的问题

克隆后、发布前，结构是符合预期的：

```json
{
  "items_json": [
    {
      "ref_ulid": "认证B_pipeline_ulid",
      "item_type": "pipeline"
    }
  ],
  "pricing_json": {
    "units": [
      {
        "unit_id": "认证B_unit_id",
        "access": {
          "stripe_price_id": "price_xxx",
          "stripe_product_id": "prod_xxx"
        },
        "retake": {
          "stripe_price_id": "price_xxx",
          "stripe_product_id": "prod_xxx"
        }
      }
    ],
    "unlocks": {},
    "memberships": [],
    "qual_reviews": []
  }
}
```

但是调用发布接口时，如果 `pricing_json.unlocks` 里没有当前 pipeline 的价格，微服务会返回错误：

```text
pipeline <pipeline_ulid> pricing is missing in pricing_json unlocks
```

也就是说，`PublishBundle` 似乎强制要求下面这个字段必须存在：

```json
{
  "pricing_json": {
    "unlocks": {
      "认证B_pipeline_ulid": {
        "stripe_price_id": "price_xxx",
        "stripe_product_id": "prod_xxx"
      }
    }
  }
}
```

如果 admin 前端为了绕过这个发布校验，把 `units[0].access` 的价格复制到 `unlocks[认证B_pipeline_ulid]`，商品确实可以发布，但发布后的结构会变成：

```json
{
  "pricing_json": {
    "units": [
      {
        "unit_id": "认证B_unit_id",
        "access": {
          "stripe_price_id": "price_xxx",
          "stripe_product_id": "prod_xxx"
        },
        "retake": {
          "stripe_price_id": "price_xxx",
          "stripe_product_id": "prod_xxx"
        }
      }
    ],
    "unlocks": {
      "认证B_pipeline_ulid": {
        "stripe_price_id": "price_xxx",
        "stripe_product_id": "prod_xxx"
      }
    },
    "memberships": [],
    "qual_reviews": []
  }
}
```

然后 cand 端候选人看到这个商品时，会被引导去创建或支付 `PIPELINE_UNLOCK` 解锁订单，之后还要再创建或支付 Bundle 购买订单。

最终用户体验变成“两次支付”，不符合预期。

## 想和微服务确认的问题

1. `pricing_json.units[].access` 和 `pricing_json.unlocks[pipeline_ulid]` 分别是什么语义？
2. 对于 Bundle 商品的“直接购买”，是否应该只依赖 `units[].access` 或 Bundle purchase pricing？
3. `pricing_json.unlocks[pipeline_ulid]` 是否代表“进入购买前必须先解锁 pipeline”的价格？
4. 如果是普通认证 Bundle 商品，用户应该直接购买整个 Bundle，那么 `pricing_json.unlocks` 是否应该允许为空？
5. `PublishBundle` 强制校验 `pricing_json.unlocks[pipeline_ulid]` 是否合理？
6. 如果 `unlocks[pipeline_ulid]` 必须存在，它是否应该只用于发布价格快照，而不应该影响 `CheckPipelineEligibility` 返回 `can_unlock` 或 cand 端购买路径？
7. `CheckPipelineEligibility` 返回 `can_purchase` 和 `can_unlock` 的逻辑，是不是把“Bundle 可购买价格配置”和“Pipeline 解锁价格配置”混在一起了？
8. 对于我们这个场景，正确配置方式到底应该是什么？

## 目前 admin 端处理方式

我们已经把 admin 端“发布前自动补 `pricing_json.unlocks`”的逻辑撤掉了，因为发布动作不应该偷偷修改商品结构。

现在 admin 端行为是：

1. 克隆商品：调用 `DuplicateBundleDraft`。
2. 替换认证：调用 `UpdateBundleStructure`，只替换 `items_json` 的 pipeline ID 和 `pricing_json.units[].unit_id`。
3. 发布商品：直接调用 `PublishBundle`，不再自动修改 `pricing_json.unlocks`。

如果微服务仍然要求 `unlocks[pipeline_ulid]`，需要微服务这边明确说明这是正确用法，还是发布校验或 eligibility 判断存在 bug。
