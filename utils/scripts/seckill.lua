-- parameters: ARGV[1] = readerID, ARGV[2] = authorID, ARGV[3] = discountID, ARGV[4] = credits
local readerID = ARGV[1]
local authorID = ARGV[2]
local discountID = ARGV[3]
local credits = ARGV[4]
-- keys:
local stockKey = 'seckill:stock:' .. discountID
local orderKey = 'seckill:order:' .. discountID

-- 1. Check if the reader has already subscribed to the author
if redis.call('sismember', orderKey, readerID) == 1 then
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
redis.call('sadd', orderKey, readerID)

-- Add the order to Redis Stream
redis.call('xadd', 'stream.orders', '*',
    'readerID', readerID,
    'authorID', authorID,
    'discountID', discountID,
    'credits', credits
)
return 0 -- Success