package snowflake

import (
	"errors"
	"strconv"
	"sync"
	"time"
)

/**
生成的ID为64位的二进制数据
1[符号位固定表示正数] 41[时间戳]  10[机器编号ID] 12[时间戳的当前机器唯一数会累加]
10位机器编号可表示：1-1023
*/

var (
	// epoch is set to the twitter snowflake epoch of Nov 04 2010 01:42:54 UTC in milliseconds
	// You may customize this to set a different epoch for your application.
	epoch int64 = 1288834974657

	// nodeBits holds the number of bits to use for Node
	// Remember, you have a total 22 bits to share between Node/Step
	nodeBits uint8 = 10

	// StepBits holds the number of bits to use for Step
	// Remember, you have a total 22 bits to share between Node/Step
	stepBits uint8 = 12
)

// A Node struct holds the basic information needed for a snowflake generator
type Node struct {
	mu    sync.Mutex
	epoch time.Time
	time  int64
	node  int64 // 节点ID 10位
	step  int64 // 序列号 12位

	nodeMax   int64
	nodeMask  int64
	stepMask  int64
	timeShift uint8
	nodeShift uint8
}

// NewNode returns a new snowflake node that can be used to generate snowflake IDs
// 参数为机器ID 机器注册中心ID
func NewNode(node int64) (*Node, error) {
	n := Node{}
	n.node = node
	n.nodeMax = -1 ^ (-1 << nodeBits)
	n.nodeMask = n.nodeMax << stepBits
	n.stepMask = -1 ^ (-1 << stepBits)
	n.timeShift = nodeBits + stepBits
	n.nodeShift = stepBits

	if n.node < 0 || n.node > n.nodeMax {
		return nil, errors.New("zoo-library-snowflake：节点ID必须在 0 - " + strconv.FormatInt(n.nodeMax, 10))
	}

	var curTime = time.Now()
	// add time.Duration to curTime to make sure we use the monotonic clock if available
	n.epoch = curTime.Add(time.Unix(epoch/1000, (epoch%1000)*1000000).Sub(curTime))

	return &n, nil
}

// NextIdInt64 返回ID的int64形式
func (n *Node) NextIdInt64() int64 {
	return n.generate()
}

// NextIdStr 返回ID的字符串形式
func (n *Node) NextIdStr() string {
	return strconv.FormatInt(n.generate(), 10)
}

// Generate creates and returns a unique snowflake ID
// To help guarantee uniqueness
// - Make sure your system is keeping accurate system time
// - Make sure you never have multiple nodes running with the same node ID
func (n *Node) generate() int64 {
	n.mu.Lock()
	now := time.Since(n.epoch).Nanoseconds() / 1000000
	if now == n.time {
		n.step = (n.step + 1) & n.stepMask

		if n.step == 0 {
			for now <= n.time {
				now = time.Since(n.epoch).Nanoseconds() / 1000000
			}
		}
	} else {
		n.step = 0
	}

	n.time = now

	r := ((now) << n.timeShift) | (n.node << n.nodeShift) | (n.step)
	n.mu.Unlock()

	return r
}

// ServerId 根据生成的UUID计算出机器ID
func ServerId(sid string) int64 {
	sidInt, _ := strconv.ParseInt(sid, 10, 64)
	nodeMax := int64(-1 ^ (-1 << nodeBits)) // 10为则为 1023
	snow := sidInt >> stepBits              //右移12位 暴露出 码的前 1+41+10位
	return snow & nodeMax                   //位与 过滤掉前面的，只留下后面10位数字
}
