package redis

import goredis "github.com/redis/go-redis/v9"

const (
	voteOK        = 0
	voteNotFound  = 1
	voteForbidden = 2
	voteConflict  = 3
	voteInvalid   = 4
	voteNoop      = 5

	undoOK        = 0
	undoNotFound  = 1
	undoForbidden = 2
	undoConflict  = 3
	undoNoop      = 4

	createOK       = 0
	createNoop     = 1
	createBadInput = 2
)

var createPollScript = goredis.NewScript(`
-- KEYS:
-- 1 idem_key
-- 2 poll_meta_key
-- 3 poll_options_key
-- 4 poll_votes_key
-- 5 poll_voters_key
-- 6 polls_index_key
-- 7 polls_public_index_key
-- 8 user_created_polls_zset_key
--
-- ARGV:
-- 1 user_id
-- 2 idempotency_key
-- 3 poll_id
-- 4 question
-- 5 created_at_unix
-- 6 expires_at_unix
-- 7 is_public ("true"/"false")
-- 8 created_score (float string)
-- 9 options_count
-- 10.. options

local idem = KEYS[1]
local poll_meta = KEYS[2]
local poll_opts = KEYS[3]
local poll_votes = KEYS[4]
local poll_voters = KEYS[5]
local polls_index = KEYS[6]
local polls_public = KEYS[7]
local user_created = KEYS[8]

local user_id = ARGV[1]
local idem_key = ARGV[2]
local poll_id = ARGV[3]
local question = ARGV[4]
local created_at = ARGV[5]
local expires_at = ARGV[6]
local is_public = ARGV[7]
local score = ARGV[8]
local nopts = tonumber(ARGV[9])

if not nopts or nopts < 1 then
  return {2, ""}
end

if idem_key ~= "" then
  local existing = redis.call("GET", idem)
  if existing then
    return {1, existing}
  end
  -- reserve idempotency for 5 minutes to avoid duplicates
  local ok = redis.call("SET", idem, poll_id, "NX", "EX", 300)
  if not ok then
    local v = redis.call("GET", idem)
    if v then
      return {1, v}
    end
  end
end

-- create poll meta
redis.call("HSET", poll_meta,
  "id", poll_id,
  "question", question,
  "created_by", user_id,
  "updated_by", user_id,
  "created_at", created_at,
  "expires_at", expires_at,
  "is_closed", "false",
  "is_public", is_public
)

-- options/votes
for i=0,nopts-1 do
  local opt = ARGV[10+i]
  redis.call("HSET", poll_opts, tostring(i), opt)
  redis.call("HSET", poll_votes, opt, 0)
end

redis.call("SADD", polls_index, poll_id)
if is_public == "true" then
  redis.call("SADD", polls_public, poll_id)
end
redis.call("ZADD", user_created, score, poll_id)
redis.call("DEL", poll_voters)

return {0, poll_id}
`)

var voteScript = goredis.NewScript(`
-- KEYS:
-- 1 poll_meta_key
-- 2 poll_votes_key
-- 3 poll_voters_key
-- 4 user_votes_key
-- 5 idem_key (may be constant if empty)
--
-- ARGV:
-- 1 poll_id
-- 2 user_id
-- 3 option
-- 4 idempotency_key

local poll_meta = KEYS[1]
local poll_votes = KEYS[2]
local poll_voters = KEYS[3]
local user_votes = KEYS[4]
local idem = KEYS[5]

local poll_id = ARGV[1]
local user_id = ARGV[2]
local option = ARGV[3]
local idem_key = ARGV[4]

if idem_key ~= "" then
  if redis.call("GET", idem) then
    return 5
  end
end

if redis.call("EXISTS", poll_meta) == 0 then
  return 1
end

local is_closed = redis.call("HGET", poll_meta, "is_closed")
if is_closed == "true" then
  return 2
end

local expires_at = redis.call("HGET", poll_meta, "expires_at")
if expires_at then
  local t = redis.call("TIME")
  local now = tonumber(t[1])
  if now > tonumber(expires_at) then
    return 2
  end
end

if redis.call("SISMEMBER", poll_voters, user_id) == 1 then
  return 3
end

if redis.call("HEXISTS", poll_votes, option) == 0 then
  return 4
end

redis.call("SADD", poll_voters, user_id)
redis.call("HINCRBY", poll_votes, option, 1)
redis.call("HSET", user_votes, poll_id, option)

if idem_key ~= "" then
  redis.call("SETEX", idem, 300, "1")
end

return 0
`)

var undoScript = goredis.NewScript(`
-- KEYS:
-- 1 poll_meta_key
-- 2 poll_votes_key
-- 3 poll_voters_key
-- 4 user_votes_key
-- 5 idem_key
--
-- ARGV:
-- 1 poll_id
-- 2 user_id
-- 3 idempotency_key

local poll_meta = KEYS[1]
local poll_votes = KEYS[2]
local poll_voters = KEYS[3]
local user_votes = KEYS[4]
local idem = KEYS[5]

local poll_id = ARGV[1]
local user_id = ARGV[2]
local idem_key = ARGV[3]

if idem_key ~= "" then
  if redis.call("GET", idem) then
    return 4
  end
end

if redis.call("EXISTS", poll_meta) == 0 then
  return 1
end

local is_closed = redis.call("HGET", poll_meta, "is_closed")
if is_closed == "true" then
  return 2
end

local expires_at = redis.call("HGET", poll_meta, "expires_at")
if expires_at then
  local t = redis.call("TIME")
  local now = tonumber(t[1])
  if now > tonumber(expires_at) then
    return 2
  end
end

if redis.call("SISMEMBER", poll_voters, user_id) == 0 then
  return 3
end

local opt = redis.call("HGET", user_votes, poll_id)
if not opt then
  return 3
end

redis.call("SREM", poll_voters, user_id)
local n = redis.call("HINCRBY", poll_votes, opt, -1)
if tonumber(n) < 0 then
  redis.call("HSET", poll_votes, opt, 0)
end
redis.call("HDEL", user_votes, poll_id)

if idem_key ~= "" then
  redis.call("SETEX", idem, 300, "1")
end

return 0
`)
