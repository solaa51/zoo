package main

import "github.com/solaa51/zoo/system/library/snowflake"

func main() {
	node, _ := snowflake.NewNode(2)
	node.NextIdStr()
}
