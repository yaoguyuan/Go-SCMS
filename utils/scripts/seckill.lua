-- parameters: ARGV[1] = discountID, ARGV[2] = userID
local discountID = ARGV[1]
local userID = ARGV[2]
-- keys:
local stockKey = 'seckill:stock:' .. discountID
local orderKey = 'seckill:order:' .. discountID

-- 1. Check if the reader has already subscribed to the author
if redis.call('sismember', orderKey, userID) == 1 then
    return 1 -- Already subscribed
end

-- 2. Check if the discount is still within the valid time range
if redis.call('exists', stockKey) == 0 then
    return 2 -- Not within valid time range
end

-- 3. Check if the discount stock is still available
if tonumber(redis.call('get', stockKey)) <= 0 then
    return 3 -- No stock available
end

-- Decrement the stock and add the user to the order set
redis.call('incrby', stockKey, -1)
redis.call('sadd', orderKey, userID)
return 0 -- Success