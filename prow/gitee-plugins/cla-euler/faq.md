---
title: "Sign CLA"
weight: 6
description: |
  An overview of how to sign CLA.
---

## 常见的问题


### 怎么设置本地开发环境

在开发前请按如下方式配置git。这里的邮箱必须是已经签署过CLA

```sh
git config user.email example@xx.com
```


### 如果需要签署CLA，请先按下图所示配置Gitee的提交邮箱，之后再签署

![gitee_email](gitee_email.png)


### 机器人提示未签署CLA，去签署又提示已签署

机器人是通过检查PullRequest中所有commit的作者是否签署了CLA来判定PR是否完成了CLA签署

此种情况请按如下步骤检查。

1. 在开发环境中运行 *git log --pretty="fuller"* ，查看PR涉及的每个commit的作者的邮箱

```sh
$ git log --pretty="fuller"

commit 6c5e70b984a60b3cecd395edd5b48a7575bf58e0
Author:     Jessica Smith <jessica@example.com>
Date:       Sun Apr 6 10:17:23 2008 -0700
Commit:     zhangsan <zhangsan@example.com>
CommitDate: Fri May 14 14:31:56 2021 +0800

   add limit to log function

   Limit log functionality to the first 20

```

**Note**: 此commit作者的邮箱是**zhangsan@example.com**, 不是jessica@example.com

2. 到Gitee账号中查看已经签署过CLA的邮箱，

![gitee_email](gitee_email.png)

3. 如果这2个邮箱地址不一致，请按如下方式处理

   step1: 更新Gitee 账号中的提交邮箱为commit作者的邮箱，并重新签署CLA

   step2: 到代码仓再次执行 **/check-cla** ，以便更新PR的CLA标签

