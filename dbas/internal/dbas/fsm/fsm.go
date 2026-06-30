package fsm

import (
	"fmt"
	"sync"
	"time"
)

// State 是 DBAS 分支的严格有限状态。
type State string

const (
	StateINIT        State = "INIT"
	StatePLAN        State = "PLAN"
	StateEXECUTING   State = "EXECUTING"
	StateVERIFYING   State = "VERIFYING"
	StatePASS        State = "PASS"
	StateFAILRetreat State = "FAIL_RETREAT"
	StateARCHIVE     State = "ARCHIVE"
	StateERROR       State = "ERROR"
)

// validTransitions 是硬编码的状态转换表，运行时不可变。
// 任何不在此表中的转换都是非法的，将被致命拒绝。
var validTransitions = map[State][]State{
	StateINIT:        {StatePLAN},
	StatePLAN:        {StateEXECUTING},
	StateEXECUTING:   {StateVERIFYING, StateERROR},
	StateVERIFYING:   {StatePASS, StateFAILRetreat, StateERROR},
	StatePASS:        {StateARCHIVE},
	StateFAILRetreat: {StateEXECUTING, StateERROR},
	StateARCHIVE:     {},
	StateERROR:       {},
}

// TransitionLog 记录一次状态转换的历史。
type TransitionLog struct {
	From      State  `json:"from"`
	To        State  `json:"to"`
	Timestamp int64  `json:"timestamp"`
	Reason    string `json:"reason,omitempty"`
}

// StateMachine 是硬编码的有限状态机，严禁动态修改转换规则。
type StateMachine struct {
	mu      sync.RWMutex
	logs    []TransitionLog
	current State
}

// NewStateMachine 初始化一个状态机，必须从 StateINIT 开始。
func NewStateMachine() *StateMachine {
	return &StateMachine{
		current: StateINIT,
		logs:    make([]TransitionLog, 0),
	}
}

// NewStateMachineWithState 从指定状态初始化状态机（用于从持久化恢复）。
func NewStateMachineWithState(initial State) *StateMachine {
	sm := NewStateMachine()
	if initial != StateINIT {
		// 直接设置当前状态，不记录日志（恢复时不需要重复日志）
		sm.current = initial
	}
	return sm
}

// Current 返回当前状态（线程安全）。
func (sm *StateMachine) Current() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current
}

// ValidateTransition 检查从 current 到 target 的转换是否合法，不修改状态。
func (sm *StateMachine) ValidateTransition(current, target State) bool {
	allowed, ok := validTransitions[current]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == target {
			return true
		}
	}
	return false
}

// CanTransition 是 ValidateTransition 的快捷方式，使用当前状态。
func (sm *StateMachine) CanTransition(target State) bool {
	return sm.ValidateTransition(sm.Current(), target)
}

// Transition 执行状态转换。
// 如果转换非法，返回致命错误，状态被强制设置为 StateERROR。
// 这是 DBAS 的核心纪律保障：任何绕过 FSM 的 Agent 输出都被拒绝。
func (sm *StateMachine) Transition(target State, reason string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.ValidateTransition(sm.current, target) {
		errMsg := fmt.Sprintf("FSM_FATAL: illegal transition from %s to %s | reason=%s", sm.current, target, reason)
		// 强制进入 ERROR 状态，拒绝执行
		sm.current = StateERROR
		sm.logs = append(sm.logs, TransitionLog{
			From:      sm.current,
			To:        StateERROR,
			Timestamp: now(),
			Reason:    errMsg,
		})
		return fmt.Errorf("%s", errMsg)
	}

	sm.logs = append(sm.logs, TransitionLog{
		From:      sm.current,
		To:        target,
		Timestamp: now(),
		Reason:    reason,
	})
	sm.current = target
	return nil
}

// ForceError 强制将状态机设置为 ERROR（用于不可恢复的系统错误）。
func (sm *StateMachine) ForceError(reason string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.logs = append(sm.logs, TransitionLog{
		From:      sm.current,
		To:        StateERROR,
		Timestamp: now(),
		Reason:    reason,
	})
	sm.current = StateERROR
}

// Logs 返回状态转换历史（只读副本）。
func (sm *StateMachine) Logs() []TransitionLog {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	out := make([]TransitionLog, len(sm.logs))
	copy(out, sm.logs)
	return out
}

// IsTerminal 返回当前状态是否为终止状态（ARCHIVE 或 ERROR）。
func (sm *StateMachine) IsTerminal() bool {
	s := sm.Current()
	return s == StateARCHIVE || s == StateERROR
}

// IsRunning 返回当前状态是否为非终止、非 INIT 的运行中状态。
func (sm *StateMachine) IsRunning() bool {
	s := sm.Current()
	return s != StateINIT && s != StateARCHIVE && s != StateERROR
}

// AllStates 返回所有合法状态列表（用于 UI 展示、校验）。
func AllStates() []State {
	return []State{
		StateINIT, StatePLAN, StateEXECUTING, StateVERIFYING,
		StatePASS, StateFAILRetreat, StateARCHIVE, StateERROR,
	}
}

// ValidTransitions 返回硬编码转换表的深拷贝（只读）。
func ValidTransitions() map[State][]State {
	out := make(map[State][]State, len(validTransitions))
	for k, v := range validTransitions {
		vv := make([]State, len(v))
		copy(vv, v)
		out[k] = vv
	}
	return out
}

func now() int64 {
	return time.Now().Unix()
}
