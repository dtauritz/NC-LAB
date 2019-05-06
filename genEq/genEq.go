package genEq

import(
	"fmt"
	"strconv"
	"math/rand"
	"time"

	"os"
)

type equation struct {
	root	*Node
}

func genEquation(depth, numinputs int, keywords []string) (equation) {
	var eq equation

	eq.root = NodeConstructor(depth, numinputs, keywords)

	return eq
}

type Node struct {
	leaf	bool
	value	string

	left	*Node
	right	*Node
}

func NodeConstructor(depth, numinputs int, keywords []string) (*Node) {
	n := &Node{}
	if depth <= 1 {
		n.leaf = true
	}

	operators := []string{"+","-","*"}

	if n.leaf {
		choice := rand.Int()%100
		if choice < 90 {
			n.value = "v" + strconv.Itoa(rand.Int()%numinputs) + keywords[rand.Int()%len(keywords)]
		} else if choice < 95 {
			n.value = strconv.FormatFloat((1-rand.Float64())*5,'f',-1, 64)
		} else {
			n.value = strconv.Itoa(rand.Int()%4 + 1)
		}
	} else {
		n.value = operators[rand.Int()%len(operators)]

		depth--
		n.left = NodeConstructor(depth, numinputs, keywords)
		n.right = NodeConstructor(depth, numinputs, keywords) 
	}

	return n
}

//printing equations
func (eq *equation) String() {
	stringify(eq.root)
}

func stringify(n *Node) {
	if n != nil {
		fmt.Printf("(")
		stringify(n.left)
		fmt.Printf("%v", n.value)
		stringify(n.right)
		fmt.Printf(")")
	}
}


func (eq *equation) toString() (string) {
	return stringer(eq.root)
}

func stringer(n *Node) (string) {
	var s string
	if n != nil {
		s = "(" + stringer(n.left) + n.value + stringer(n.right) + ")"
	}
	return s
}


func (eq *equation) Fancy() {
	fmt.Println("------------------------------------------------")
    fancyStringify(eq.root, 0)
    fmt.Println("------------------------------------------------")
}

func fancyStringify(n *Node, level int) {
    if n != nil {
        format := ""
        for i := 0; i < level; i++ {
            format += "       "
        }
        format += "---[ "
        level++
        fancyStringify(n.left, level)
        fmt.Printf(format+"%v\n", n.value)
        fancyStringify(n.right, level)
    }
}


func check(e error) {
	if e != nil {
		panic(e)
	}
}


// func main() {
//     rand.Seed(time.Now().UTC().UnixNano())
// 	keywords := []string{"hardness","hardVariance","corrosion","corrVariance","conductivity","condVariance"}
// 	depth := 3
// 	numinputs := 2
// 	files := []string{"plating.txt","smelting.txt","condTreat.txt"}

// 	for _, file := range files {
// 		f, err := os.Create(file)
// 		check(err)
// 		for _, key := range keywords {
// 			_, err = f.WriteString(key + "\n")
		
// 			eq := genEquation(depth, numinputs, keywords)
		
// 			_, err = f.WriteString(eq.toString() + "\n")
// 		}
// 		f.Close()
// 	}
// }

func CreateFiles(keywords, files []string, depth, numinputs int) {
    rand.Seed(time.Now().UTC().UnixNano())
	// keywords := []string{"hardness","hardVariance","corrosion","corrVariance","conductivity","condVariance"}
	// depth := 3
	// numinputs := 2
	// files := []string{"plating.txt","smelting.txt","condTreat.txt"}

	for _, file := range files {
		f, err := os.Create(file)
		check(err)
		for _, key := range keywords {
			_, err = f.WriteString(key + "\n")
		
			eq := genEquation(depth, numinputs, keywords)
		
			_, err = f.WriteString(eq.toString() + "\n")
		}
		f.Close()
	}
}