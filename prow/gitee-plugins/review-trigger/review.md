# PR的review 过程

---

此文描述的是pull-request的CI测试用例由其他第三方插件控制的PR检视，合入过程

## 方案
社区的成员角色分为3类，分别是一般贡献者，reviewer 和 approver，他们拥有不同的权力，并通过不同的命令表达其对PR的检视意见。
一般贡献者的检视意见无法通过命令来作用于PR，换句话说，其无法影响PR的合入；只有reviewer和approver的指令可以影响PR的合入。
approver拥有指定模块的合入权限， reviewer无合入权限。

| 命令                       | 描述                         | 谁能使用                                 | 使用场景           |
| :------------------------- | :--------------------------- | ---------------------------------------- | ---                |
| /lgtm --- looks good to me | 认可代码                     | 任何人， 但不包括PR作者                  | 代码的质量检视    |
| /lbtm --- looks bad to me  | 不认可代码，希望修改代码     | 任何人                                   | 代码的质量检视     |
| /approve                   | 可以合入                     | approver，根据配置决定是否包含PR作者     | 合入代码的指令     |
| /reject                    | 代码需要做重大修改           | approver，但不包括PR作者                 | 拒绝代码合入的指令 |

需要说明的是，approver评论了/reject，需要PR的作者与该approver沟通，让其取消reject（评论/lgtm，/lbtm，/approve），否则该PR将无法合入。

## PR标签变化过程

![here](https://github.com/opensourceways/test-infra/blob/sync-5-22/prow/gitee-plugins/review-trigger/state.png)

### 标签变化说明
PR 创建 或 代码更新 ---> can-review： 当一个PR创建或其代码发生了变化时，将会给PR打上can-review标签，表示PR可以review了

can-review ---> lgtm： 当一个reviewer评论了/lgtm，或一个approver评论了/approve，将删除can-review标签，并打上lgtm标签

can-review ---> request-change： 当一个reviewer评论了/lbtm，或一个approver评论了/reject，将删除can-review标签，并打上request-change标签

lgtm ---> request-change： 当一个reviewer评论了/lbtm，或一个approver评论了/reject，将删除lgtm标签，并打上request-change标签

lgtm ---> approved： 当PR涉及的所有目录都被approve了，将打上approved标签

request-change ---> lgtm： 当一个reviewer评论了/lgtm，或一个approver评论了/approve，将删除request-change标签，并打上lgtm标签

approved ---> request-change： 当一个approver评论了/reject，将删除approved标签，并打上request-change标签

## PR的合入过程

![here](https://github.com/opensourceways/test-infra/blob/sync-5-22/prow/gitee-plugins/review-trigger/review_process.png)

### 合入过程说明
step1： 当一个PR创建时，机器人将自动给PR推荐reviewers，并打上can-review标签，表示PR可以review了

step2： reviewer 针对PR的实现进行review，最后打上lgtm标签。此过程中，PR可能会修改代码，此时PR的各种标签将会清除，回到step1的状态

step3： 等待PR的测试用例通过

step4： 测试用例通过后为PR推荐approvers，等待approver进行review

step5： approvers 对代码进行最后的审视，如果不认可此PR，可以评论/reject，要求PR的作者进行修改；否则评论/approve，表示同意合入PR

step6： 当PR涉及的所有模块(目录)全部被approved，则打上approved标签

step7： PR进入合入队列，等待被合入
