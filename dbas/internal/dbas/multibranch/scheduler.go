package multibranch
import (
	"context"
	"fmt"
	"sync"
	"reasonix/dbas/internal/dbas/branch"
	"reasonix/dbas/internal/dbas/persistence"
)
// TaskInput 瀹氫箟涓€涓苟琛屽瓙浠诲姟鐨勮緭鍏ャ€?
type TaskInput struct {
	Name  string `json:"name"`
	Input string `json:"input"`
}
// CoordinatorFactory 鏄垱寤哄垎鏀崗璋冨櫒鐨勫伐鍘傛帴鍙ｃ€?
type CoordinatorFactory interface {
	Create(b *branch.Branch) (Runner, error)
}
// Runner 鏄墽琛屾帴鍙ｃ€?
type Runner interface {
	Run(ctx context.Context, input string) error
}
// Scheduler 鏄鍒嗘敮骞惰璋冨害鍣ㄣ€?
// 瀹冪鐞嗕竴涓叡浜?Radix-Tree 涓婁笅鏂囷紝骞跺湪姝や箣涓婂苟鍙戞墽琛屽涓垎鏀€?
type Scheduler struct {
	engine      *branch.BranchEngine
	store       *persistence.BranchStore
	factory     CoordinatorFactory
	tree        *RadixTree
	maxParallel int
}
// NewScheduler 鍒濆鍖栧鍒嗘敮璋冨害鍣ㄣ€?
func NewScheduler(
	engine *branch.BranchEngine,
	store *persistence.BranchStore,
	factory CoordinatorFactory,
	maxParallel int,
) *Scheduler {
	if maxParallel <= 0 {
		maxParallel = 4
	}
	return &Scheduler{
		engine:      engine,
		store:       store,
		factory:     factory,
		tree:        NewRadixTree(),
		maxParallel: maxParallel,
	}
}
// InitContextTree 鍒濆鍖栧叡浜笂涓嬫枃鏍戙€?
// systemPrompt 鍜?toolSchema 鏋勬垚 Zone 1锛堟牴鑺傜偣锛夈€?
// sharedContext 鏋勬垚 Zone 2锛堝瓙鑺傜偣锛夈€?
func (s *Scheduler) InitContextTree(systemPrompt, sortedToolSchema, sharedContext string) (*RadixNode, *RadixNode) {
	root := s.tree.CreateRoot(systemPrompt, sortedToolSchema)
	shared := s.tree.CreateChild(root.ID, sharedContext, ZoneAppend)
	return root, shared
}
// SpawnParallel 浠庡叡浜笂涓嬫枃鑺傜偣娲剧敓澶氫釜骞惰鍒嗘敮锛屽苟鍙戞墽琛屻€?
func (s *Scheduler) SpawnParallel(ctx context.Context, sharedNodeID string, inputs []TaskInput) ([]*branch.Branch, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("no task inputs provided")
	}
	// 鍒涘缓鎵€鏈夊垎鏀?
	branches := make([]*branch.Branch, len(inputs))
	for i, in := range inputs {
		b, err := s.engine.CreateBranch(in.Name, nil) // Phase 2: parent 涓?sharedNodeID
		if err != nil {
			return nil, fmt.Errorf("create branch %s: %w", in.Name, err)
		}
		if err := s.store.InitBranch(b); err != nil {
			return nil, fmt.Errorf("init branch %s: %w", b.BranchID, err)
		}
		branches[i] = b
	}
	// 浣跨敤 semaphore 鎺у埗骞跺彂搴?
	sem := make(chan struct{}, s.maxParallel)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	for i, in := range inputs {
		wg.Add(1)
		go func(idx int, taskInput TaskInput) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			b := branches[idx]
			// 缁戝畾 Radix-Tree 涓婁笅鏂囧埌鍒嗘敮鏃ュ織
			b.Logs = append(b.Logs, fmt.Sprintf("[MULTIBRANCH] Shared context node: %s", sharedNodeID))
			_ = s.store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
			// 鍒涘缓鍗忚皟鍣ㄥ苟鎵ц
			coord, err := s.factory.Create(b)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("branch %s coordinator: %w", b.BranchID, err))
				mu.Unlock()
				return
			}
			if err := coord.Run(ctx, taskInput.Input); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("branch %s run: %w", b.BranchID, err))
				mu.Unlock()
			}
		}(i, in)
	}
	wg.Wait()
	if len(errs) > 0 {
		return branches, fmt.Errorf("parallel execution had %d errors", len(errs))
	}
	return branches, nil
}
// SpawnSequential 浠庡叡浜笂涓嬫枃鑺傜偣娲剧敓澶氫釜椤哄簭鎵ц鍒嗘敮銆?
// 鍓嶄竴涓垎鏀畬鎴愬悗锛屽悗涓€涓墠寮€濮嬨€?
func (s *Scheduler) SpawnSequential(ctx context.Context, sharedNodeID string, inputs []TaskInput) ([]*branch.Branch, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("no task inputs provided")
	}
	branches := make([]*branch.Branch, len(inputs))
	for i, in := range inputs {
		b, err := s.engine.CreateBranch(in.Name, nil)
		if err != nil {
			return nil, fmt.Errorf("create branch %s: %w", in.Name, err)
		}
		if err := s.store.InitBranch(b); err != nil {
			return nil, fmt.Errorf("init branch %s: %w", b.BranchID, err)
		}
		branches[i] = b
	}
	for i, in := range inputs {
		b := branches[i]
		b.Logs = append(b.Logs, fmt.Sprintf("[SEQUENTIAL] Step %d/%d, shared context: %s", i+1, len(inputs), sharedNodeID))
		_ = s.store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
		coord, err := s.factory.Create(b)
		if err != nil {
			return branches, fmt.Errorf("branch %s coordinator: %w", b.BranchID, err)
		}
		if err := coord.Run(ctx, in.Input); err != nil {
			return branches, fmt.Errorf("branch %s run: %w", b.BranchID, err)
		}
	}
	return branches, nil
}
// GetTree 杩斿洖搴曞眰鐨?Radix-Tree锛堢敤浜庤皟璇?璇婃柇锛夈€?
func (s *Scheduler) GetTree() *RadixTree {
	return s.tree
}
