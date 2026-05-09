package redis

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"vote-system/internal/service"

	goredis "github.com/redis/go-redis/v9"
)

func newIntegrationStore(t *testing.T) (*Store, *goredis.Client) {
	t.Helper()

	// 这是“集成测试”，不是普通单元测试。
	//
	// 单元测试通常只测一小段 Go 代码，不依赖外部服务；而这里会真实连接 Redis，
	// 所以必须先通过环境变量 REDIS_ADDR 指定 Redis 地址，例如：
	//
	//   REDIS_ADDR=127.0.0.1:6379 go test ./internal/store/redis
	//
	// 如果没有设置 REDIS_ADDR，就跳过这些测试，避免平时运行 go test 时因为本机
	// 没开 Redis 而失败。
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("REDIS_ADDR is not set; skipping Redis integration test")
	}

	// 创建 Redis 客户端。goredis.NewClient 只是创建连接配置，真正检查 Redis
	// 是否可用要看下面的 Ping。
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
	})

	// t.Cleanup 会在当前测试结束后自动执行。这里用来关闭 Redis 客户端，
	// 防止测试跑完后连接资源没有释放。
	t.Cleanup(func() {
		_ = rdb.Close()
	})

	// context.WithTimeout 表示：下面这个 Redis Ping 最多等 5 秒。
	// 如果 Redis 无响应，测试不会无限卡住。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("redis ping %s failed: %v", addr, err)
	}

	// New(rdb) 会把 Redis 客户端包装成本项目自己的 Store。
	// 返回 rdb 是为了测试里可以直接检查 Redis 内部 key 的状态。
	return New(rdb), rdb
}

func createIntegrationPoll(t *testing.T, ctx context.Context, store *Store, pollID string) service.Poll {
	t.Helper()

	// 构造一个测试用投票。
	//
	// pollID 由每个测试自己传进来，一般会带上当前时间戳。
	// 这样做是为了让每次测试使用不同的 Redis key，避免多次运行测试时数据互相影响。
	poll := service.Poll{
		ID:        pollID,
		Question:  "Redis concurrency test",
		Options:   []string{"Go", "Redis", "Kubernetes"},
		Votes:     map[string]int64{},
		CreatedBy: "redis_test_creator",
		UpdatedBy: "redis_test_creator",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().Add(time.Hour).UTC(),
		IsPublic:  true,
	}

	// 调用真正的 Redis Store 创建投票。
	// 如果这里失败，后面的并发投票测试就没有意义，所以直接 t.Fatalf 终止当前测试。
	created, err := store.CreatePoll(ctx, poll)
	if err != nil {
		t.Fatalf("CreatePoll returned error: %v", err)
	}
	return created
}

func cleanupIntegrationPoll(t *testing.T, ctx context.Context, rdb *goredis.Client, pollID string, users []string, idemKeys []string) {
	t.Helper()

	// 清理当前测试可能写入 Redis 的所有 key。
	//
	// Redis 集成测试如果不清理数据，下次运行测试时可能读到旧数据，导致结果不稳定。
	// 这里不仅删除投票本身的数据，还会删除：
	//   1. 每个用户的投票记录；
	//   2. 幂等请求记录；
	//   3. 投票列表索引；
	//   4. 公开投票索引；
	//   5. 创建者创建过的投票索引。
	keys := []string{
		pollMetaKey(pollID),
		pollOptionsKey(pollID),
		pollVotesKey(pollID),
		pollVotersKey(pollID),
	}
	for _, userID := range users {
		keys = append(keys, userVotesKey(userID))
		for _, idem := range idemKeys {
			// idemKey 用来记录“某个用户的某个幂等请求是否已经处理过”。
			// 如果不删掉，后续测试可能会把新请求误判为旧请求重试。
			keys = append(keys, idemKey(userID, idem))
		}
	}

	// keys... 是 Go 的可变参数写法。
	// rdb.Del(ctx, keys...) 等价于把切片里的每个 key 一个个传给 Del。
	if err := rdb.Del(ctx, keys...).Err(); err != nil {
		t.Fatalf("cleanup Del returned error: %v", err)
	}
	if err := rdb.SRem(ctx, pollsIndexKey(), pollID).Err(); err != nil {
		t.Fatalf("cleanup SRem polls index returned error: %v", err)
	}
	if err := rdb.SRem(ctx, pollsPublicIndexKey(), pollID).Err(); err != nil {
		t.Fatalf("cleanup SRem public index returned error: %v", err)
	}
	if err := rdb.ZRem(ctx, userCreatedPollsKey("redis_test_creator"), pollID).Err(); err != nil {
		t.Fatalf("cleanup ZRem creator index returned error: %v", err)
	}
}

func TestRedisConcurrentSameUserVotesOnce(t *testing.T) {
	store, rdb := newIntegrationStore(t)
	ctx := context.Background()

	// 使用当前纳秒时间拼出唯一 pollID，降低测试之间 Redis key 冲突的概率。
	pollID := fmt.Sprintf("redis-it-same-user-%d", time.Now().UnixNano())
	userID := "redis_it_same_user"

	// defer 表示当前测试函数结束时再执行 cleanup。
	// 即使中间 t.Fatalf 失败退出，也会尽量执行清理逻辑。
	cleanup := func() { cleanupIntegrationPoll(t, ctx, rdb, pollID, []string{userID}, nil) }
	defer cleanup()

	createIntegrationPoll(t, ctx, store, pollID)

	// 测试目标：
	// 同一个用户同时发起 100 次投票请求，系统应该只接受其中 1 次。
	//
	// 原因：
	// 一个用户对同一个投票只能投一次。如果并发控制没做好，可能会出现同一个用户
	// 被重复计票的问题。这个测试就是专门检查 Redis 里的并发写入是否安全。
	const attempts = 100

	// 这几个计数器会被多个 goroutine 同时修改，所以使用 int64 + atomic。
	// 普通的 success++ 在并发场景下不是安全的。
	var success int64
	var conflicts int64
	var unexpected int64

	// WaitGroup 用来等待所有 goroutine 执行完。
	// 可以理解为：每启动一个并发任务就 wg.Add(1)，任务结束时 wg.Done()，
	// 最后 wg.Wait() 会一直等到所有任务都 Done。
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)

		// go func() { ... }() 会启动一个 goroutine，也就是一个轻量级并发任务。
		// 这里故意同时发起很多 Vote，模拟真实环境下的并发请求。
		go func() {
			defer wg.Done()
			_, err := store.Vote(ctx, pollID, userID, "Go", "")
			switch {
			case err == nil:
				atomic.AddInt64(&success, 1)
			case errors.Is(err, service.ErrConflict):
				atomic.AddInt64(&conflicts, 1)
			default:
				atomic.AddInt64(&unexpected, 1)
			}
		}()
	}
	wg.Wait()

	// 期望只有 1 次投票成功。
	if success != 1 {
		t.Fatalf("success count = %d, want 1", success)
	}

	// 其余 99 次都应该返回 ErrConflict，表示“这个用户已经投过票了”。
	if conflicts != attempts-1 {
		t.Fatalf("conflict count = %d, want %d", conflicts, attempts-1)
	}

	// unexpected 表示既不是成功，也不是预期中的冲突错误。
	// 例如 Redis 报错、代码 panic 前返回了别的错误等，都算异常。
	if unexpected != 0 {
		t.Fatalf("unexpected error count = %d, want 0", unexpected)
	}

	// 再从 Redis 读回投票结果，确认“Go”真的只增加了 1 票。
	poll, err := store.GetPoll(ctx, pollID)
	if err != nil {
		t.Fatalf("GetPoll returned error: %v", err)
	}
	if got := poll.Votes["Go"]; got != 1 {
		t.Fatalf("Go votes = %d, want 1", got)
	}

	// pollVotersKey(pollID) 对应的是“这个 poll 已投票用户集合”。
	// SCard 是 Redis Set 的成员数量。
	// 这里检查集合里也只有 1 个用户，避免出现“票数是 1，但投票人记录错误”的情况。
	if got := rdb.SCard(ctx, pollVotersKey(pollID)).Val(); got != 1 {
		t.Fatalf("voter set size = %d, want 1", got)
	}
}

func TestRedisConcurrentDifferentUsersAllCounted(t *testing.T) {
	store, rdb := newIntegrationStore(t)
	ctx := context.Background()
	pollID := fmt.Sprintf("redis-it-many-users-%d", time.Now().UnixNano())

	// voters 表示本测试会模拟 120 个不同用户同时投票。
	const voters = 120
	users := make([]string, 0, voters)
	for i := 0; i < voters; i++ {
		users = append(users, fmt.Sprintf("redis_it_voter_%03d", i))
	}
	defer cleanupIntegrationPoll(t, ctx, rdb, pollID, users, nil)

	createIntegrationPoll(t, ctx, store, pollID)

	// 测试目标：
	// 120 个不同用户同时投票，应该全部成功。
	//
	// 和上一个测试不同：上一个是“同一个用户重复投票”，应该只有 1 次成功；
	// 这里是“不同用户投票”，不应该互相冲突。
	options := []string{"Go", "Redis", "Kubernetes"}
	var success int64
	var unexpected int64
	var wg sync.WaitGroup
	for i, userID := range users {
		wg.Add(1)

		// 注意这里把 userID 和 option 作为参数传进匿名函数。
		// 这是 Go 并发代码里常见的写法，可以避免 goroutine 里误用循环变量。
		go func(userID, option string) {
			defer wg.Done()
			if _, err := store.Vote(ctx, pollID, userID, option, ""); err != nil {
				atomic.AddInt64(&unexpected, 1)
				return
			}
			atomic.AddInt64(&success, 1)
		}(userID, options[i%len(options)]) // i%len(options) 用来让用户轮流投给 3 个选项。
	}
	wg.Wait()

	// 所有不同用户都应该投票成功。
	if success != voters {
		t.Fatalf("success count = %d, want %d", success, voters)
	}
	if unexpected != 0 {
		t.Fatalf("unexpected error count = %d, want 0", unexpected)
	}

	// 读回投票结果，检查总票数是否等于用户数。
	//
	// 这里不强行检查每个选项必须是多少票，而是把所有选项票数加起来。
	// 对这个测试来说，最重要的是“并发写入没有丢票”，也就是总数必须正确。
	poll, err := store.GetPoll(ctx, pollID)
	if err != nil {
		t.Fatalf("GetPoll returned error: %v", err)
	}
	var total int64
	for _, n := range poll.Votes {
		total += n
	}
	if total != voters {
		t.Fatalf("total votes = %d, want %d; votes=%#v", total, voters, poll.Votes)
	}

	// 投票人集合数量也应该等于 120，说明每个用户的投票记录都写进 Redis 了。
	if got := rdb.SCard(ctx, pollVotersKey(pollID)).Val(); got != voters {
		t.Fatalf("voter set size = %d, want %d", got, voters)
	}
}

func TestRedisConcurrentUndoIsSingleEffective(t *testing.T) {
	store, rdb := newIntegrationStore(t)
	ctx := context.Background()
	pollID := fmt.Sprintf("redis-it-undo-%d", time.Now().UnixNano())
	userID := "redis_it_undo_user"
	defer cleanupIntegrationPoll(t, ctx, rdb, pollID, []string{userID}, nil)

	createIntegrationPoll(t, ctx, store, pollID)

	// 先让用户成功投一票。后面才能测试“撤销投票”的并发安全性。
	if _, err := store.Vote(ctx, pollID, userID, "Redis", ""); err != nil {
		t.Fatalf("initial Vote returned error: %v", err)
	}

	// 测试目标：
	// 同一个用户已经投过 1 票，然后同时发起 80 次撤销投票请求。
	// 正确结果应该是：只有 1 次撤销真正成功，其余请求返回冲突。
	//
	// 如果并发控制有问题，可能会把票数减成负数，或者用户投票记录被删了多次。
	const attempts = 80
	var success int64
	var conflicts int64
	var unexpected int64
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.UndoVote(ctx, pollID, userID, "")
			switch {
			case err == nil:
				atomic.AddInt64(&success, 1)
			case errors.Is(err, service.ErrConflict):
				atomic.AddInt64(&conflicts, 1)
			default:
				atomic.AddInt64(&unexpected, 1)
			}
		}()
	}
	wg.Wait()

	if success != 1 {
		t.Fatalf("undo success count = %d, want 1", success)
	}
	if conflicts != attempts-1 {
		t.Fatalf("undo conflict count = %d, want %d", conflicts, attempts-1)
	}
	if unexpected != 0 {
		t.Fatalf("unexpected error count = %d, want 0", unexpected)
	}

	// 撤销成功后，“Redis”这个选项的票数应该回到 0。
	poll, err := store.GetPoll(ctx, pollID)
	if err != nil {
		t.Fatalf("GetPoll returned error: %v", err)
	}
	if got := poll.Votes["Redis"]; got != 0 {
		t.Fatalf("Redis votes after undo = %d, want 0", got)
	}

	// 投票人集合也应该为空，因为这个用户已经撤销投票了。
	if got := rdb.SCard(ctx, pollVotersKey(pollID)).Val(); got != 0 {
		t.Fatalf("voter set size after undo = %d, want 0", got)
	}

	// 用户自己的投票历史里也不应该再保存这个 pollID 对应的选择。
	// HGet 读不到值时这里会得到空字符串。
	if got := rdb.HGet(ctx, userVotesKey(userID), pollID).Val(); got != "" {
		t.Fatalf("user vote history after undo = %q, want empty", got)
	}
}

func TestRedisConcurrentIdempotentVoteRetryDoesNotDoubleCount(t *testing.T) {
	store, rdb := newIntegrationStore(t)
	ctx := context.Background()
	pollID := fmt.Sprintf("redis-it-idem-%d", time.Now().UnixNano())
	userID := "redis_it_idem_user"

	// idem 是 idempotency key，也就是“幂等键”。
	// 它表示：多次请求其实是同一个逻辑请求的重试。
	// 例如客户端提交投票后网络超时，于是又重试发送同一个请求。
	idem := "vote-once"
	defer cleanupIntegrationPoll(t, ctx, rdb, pollID, []string{userID}, []string{idem})

	createIntegrationPoll(t, ctx, store, pollID)

	// 测试目标：
	// 同一个用户、同一个幂等键，同时发起 100 次投票请求。
	//
	// 正确结果：
	//   1. 每次调用都可以返回成功，因为它们被视为同一个请求的重试；
	//   2. Redis 里最终只能增加 1 票，不能重复计票。
	const attempts = 100
	var success int64
	var unexpected int64
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.Vote(ctx, pollID, userID, "Kubernetes", idem); err != nil {
				atomic.AddInt64(&unexpected, 1)
				return
			}
			atomic.AddInt64(&success, 1)
		}()
	}
	wg.Wait()

	if success != attempts {
		t.Fatalf("successful idempotent calls = %d, want %d", success, attempts)
	}
	if unexpected != 0 {
		t.Fatalf("unexpected error count = %d, want 0", unexpected)
	}

	// 虽然上面 100 次调用都成功了，但真实票数必须只有 1。
	// 这就是幂等性的核心：同一个逻辑请求执行多次，最终效果和执行一次一样。
	poll, err := store.GetPoll(ctx, pollID)
	if err != nil {
		t.Fatalf("GetPoll returned error: %v", err)
	}
	if got := poll.Votes["Kubernetes"]; got != 1 {
		t.Fatalf("Kubernetes votes = %d, want 1", got)
	}
	if got := rdb.SCard(ctx, pollVotersKey(pollID)).Val(); got != 1 {
		t.Fatalf("voter set size = %d, want 1", got)
	}
}
