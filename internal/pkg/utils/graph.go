package utils

import (
	"fmt"
	guuid "github.com/google/uuid"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"math/rand"
	"time"
)

/*
	Useage:

	graph := utils.NewDirectedGraph(10, 3, 30, 30)
	graph.Generation()
	for i, list := range graph.HeightBlockId {
		fmt.Println("Level:", i)
		fmt.Println(list)
	}
*/

type DirectedGraph struct {
	blocks           map[string]*quorumpb.Block
	maxheight        int
	currentmaxheight int
	maxchild         int
	forkrate         int
	stoprate         int
	GenesisBlockId   string
	HeightBlockId    [][]string
}

func ifFork(forkrate int) bool {
	v := rand.Intn(100)
	if v < forkrate {
		return true
	}
	return false
}

func ifStop(stoprate int) bool {
	v := rand.Intn(100)
	if v < stoprate {
		return true
	}
	return false
}
func randid(max int) int {
	v := rand.Intn(max)
	return v
}

func NewDirectedGraph(maxheight int, maxchild int, forkrate int, stoprate int) *DirectedGraph { //forkrate 0-100, 0 = never fork, 100 = always fork
	rand.Seed(time.Now().UnixNano())
	height := 0
	graph := &DirectedGraph{maxheight: maxheight, maxchild: maxchild, forkrate: forkrate, stoprate: stoprate}
	graph.blocks = make(map[string]*quorumpb.Block)
	graph.HeightBlockId = make([][]string, maxheight)
	//genesis block
	genesis := newRandomBlock("")
	graph.GenesisBlockId = genesis.BlockId
	graph.blocks[genesis.BlockId] = genesis
	graph.HeightBlockId[height] = []string{graph.GenesisBlockId}
	graph.currentmaxheight = 1
	return graph
}

func GetAllBlocks() {

}

func (g *DirectedGraph) GetBlock(blockid string) *quorumpb.Block {
	return g.blocks[blockid]
}

func (g *DirectedGraph) Generation() {
	for {
		if g.currentmaxheight < g.maxheight {
			maxheightid := g.HeightBlockId[g.currentmaxheight-1]
			idx := randid(len(maxheightid))
			g.AppendBlock(maxheightid[idx], 1)
		} else {
			return
		}
	}
}

func (g *DirectedGraph) AppendBlock(parentBlockId string, height int) int {
	if g.maxheight > height {
		if ifStop(g.stoprate) == false {
			if ifFork(g.forkrate) == true {
				childnum := rand.Intn(g.maxchild+1-2) + 2
				for i := 0; i < childnum; i++ {
					newblock := newRandomBlock(parentBlockId)
					g.blocks[newblock.BlockId] = newblock
					g.HeightBlockId[height] = append(g.HeightBlockId[height], newblock.BlockId)
					newheight := g.AppendBlock(newblock.BlockId, height+1)
					if newheight > g.currentmaxheight {
						g.currentmaxheight = newheight
					}
				}
			} else {
				newblock := newRandomBlock(parentBlockId)
				g.blocks[newblock.BlockId] = newblock
				g.HeightBlockId[height] = append(g.HeightBlockId[height], newblock.BlockId)
				newheight := g.AppendBlock(newblock.BlockId, height+1)
				if newheight > g.currentmaxheight {
					g.currentmaxheight = newheight
				}
			}
		}
	}
	return height
}

func (g *DirectedGraph) GetSubBlocks(parentBlockId string) ([]*quorumpb.Block, error) {
	var result []*quorumpb.Block
	fmt.Println("len(g.blocks)")
	fmt.Println(len(g.blocks))
	for _, block := range g.blocks {
		if parentBlockId == block.PrevBlockId {
			result = append(result, block)
		}
	}
	return result, nil
}

func newRandomBlock(parentBlockId string) *quorumpb.Block {
	var newBlock quorumpb.Block
	blockId := guuid.New()
	newBlock.BlockId = blockId.String()
	newBlock.PrevBlockId = parentBlockId
	return &newBlock
}
