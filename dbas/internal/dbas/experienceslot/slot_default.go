package experienceslot

// slot_default.go 包含硬编码的默认经验槽配置。
// 这些值是 DBAS 的"出厂纪律"，可以在运行时通过配置文件覆盖，
// 但核心规则（escalation_threshold=3, bug_loop max_retries=3）强烈建议保留。

// DefaultProjectID 是默认项目标识。
const DefaultProjectID = "default-dbas-project"

// DefaultEscalationThreshold 是强制升级前的最大失败次数。
// 连续失败达到此值，执行器从 deepseek-v4-flash 升级为 qwen-pro-fallback。
const DefaultEscalationThreshold = 3

// DefaultVerifierCodeSyntax 是默认的代码语法检查工具。
// Phase 1: "ast-check" (占位)
// Phase 2: 根据语言自动选择 (go-vet, eslint, pyflake, etc.)
const DefaultVerifierCodeSyntax = "ast-check"

// DefaultTestPassRate 是验收通过的最低测试通过率（百分比）。
const DefaultTestPassRate = 100

// DefaultBugLoopMaxRetries 是 bug_loop 工作流的最大重试次数。
const DefaultBugLoopMaxRetries = 3

// DefaultBugLoopWorkflowName 是默认的 bug 回退工作流名称。
const DefaultBugLoopWorkflowName = "bug_loop"
