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
)

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
