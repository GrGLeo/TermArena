package game

type Node struct {
	x, y    int
	g, h, f int
	parent  *Node
	index   int
}

type PriorityQueue []*Node

func (pq PriorityQueue) Len() int {
  return len(pq)
}

func (pq PriorityQueue) Less(i, j int) bool {
  return pq[i].f < pq[j].f
}

func (pq PriorityQueue) Swap(i, j int) {
  pq[i], pq[j] = pq[j], pq[i]
  pq[i].index = i
  pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
  n := len(*pq)
  node := x.(*Node)
  node.index = n
  *pq = append(*pq, node)
}

func (pq *PriorityQueue) Pop() interface{} {
  old := *pq
  n := len(old)
  node := old[n-1]
  node.index = -1
  *pq = old[0: n-1]
  return node
}

func heuristic(a, b *Node) int {
  return Absolute(a.x - b.x) + Absolute(a.y - b.y)
}

//func aStarSearch(grid *Board, start, goal *Node) []*Node {
//neighbors := [][2]int{{0, 1}, {1, 0}, {0, -1}, {-1, 0}}
//  openSet := &PriorityQueue{}
//  heap.Init()
//  head.Push(openSet, start)
//  
//  cameFrom := make(map[*Node]*Node)
//  gScore := make(map[*Node]int)
//  gScore[start] = 0
//  fScore := make(map[*Node]int)
//  fScore[start] = heuristic(start, goal)
//
//  for openSet.Len() > 0 {
//    current := heap.Pop(openSet).(*Node)
//    if current.x == goal.x && current.y == goal.y {
//      return reconstructPath(cameFrom, current)
//    }
//
//    for _, dir := range neighbors {
//      x, y := current.x + dir[0], current.y + dir[1]
//      if x < 0 || x >= len(grid.CurrentGrid) || y < 0 || len(grid.CurrentGrid[0]) || grid.CurrentGrid[x][y] == 1 {
//        continue
//      }
//
//      neighbor := &Node{x: x, y: y}
//      tentativeGScore := gScore[current] + heuristic(current, neighbor)
//
//      if _, exists := gScore[neighbor]; !exists || tentativeGScore < gScore[neighbor] {
//        cameFrom[neighbor] = current
//        gScore[neighbor] = tentativeGScore
//        fScore[neighbor] = tentativeGScore + heuristic(neighbor, goal)
//        if _, exists := openSet.indexMap[neighbor]; !exists {
//          heap.Push(openSet, neighbor)
//        }
//      }
//    }
//  }
//  return nil
//}
