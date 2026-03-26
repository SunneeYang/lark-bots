# 配置文件更新说明

## 新的 tasks 字段格式

Executor 配置现在支持参数化的任务配置：

```yaml
bots:
  - name: "dev-executor"
    role: "executor"
    allowed_dispatchers:
      - "cli_xxxxxxxxxxxxxxxxx"
    tasks:
      miwu-dev-restart:
        script: "/opt/scripts/build.sh"
        params: ["dev", "zh"]
      potato-prod-deploy:
        script: "/opt/scripts/deploy.sh"
        params: ["prod", "us"]
      check-logs:
        script: "/opt/scripts/check_logs.sh"
        params: []
```

### 字段说明
- `tasks`: 任务映射表（对象）
  - `<task_name>`: 任务名称（键）
    - `script`: 脚本路径（字符串，必填）
    - `params`: 参数列表（数组，必填，可以为空）

### 迁移说明
旧的 `task_scripts` 字段已被 `tasks` 字段替代。
