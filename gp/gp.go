package main

import(
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"sort"
	"github.com/Knetic/govaluate"
	"bufio"

	"genEq"
)

//begin metal
type metal struct {
	attributes		map[string]float64
}

func metalConstructor() metal {
	var result metal
	result.attributes = make(map[string]float64)
	return result
}

func (m metal) String() string {
	return fmt.Sprintf("Hardness %v (%v) Conductivity %v (%v) Corrosion %v (%v)",
		m.attributes["hardness"], m.attributes["hardVariance"], m.attributes["conductivity"], m.attributes["condVariance"], m.attributes["corrosion"],
		m.attributes["corrVariance"])
}
//end metal

//begin gp
type byScore []individual

func (s byScore) Len() int {
	return len(s)
}

func (s byScore) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byScore) Less(i, j int) bool {
	return s[i].fitness < s[j].fitness
}

type individual struct {
	eq 		equation
	fitness	float64
}

type equation struct {
	root	*Node
}

func (ind *individual) copyTree(other individual) {
	ind.eq.copyTree(other.eq)
}

func (eq *equation) copyTree(other equation) {
	eq.root = other.root.copyTree()
}

func (n *Node) copyTree() *Node {
	if n == nil {
		return nil
	} else {
		copiedNode := &Node {
			n.leaf,
			n.value,
			n.left.copyTree(),
			n.right.copyTree(),
		}
		return copiedNode
	}
}

func genEquation(depth, numinputs int, keywords []string) (equation) {
	var eq equation

	eq.root = NodeConstructor(depth, numinputs, keywords)

	return eq
}

func growEquation(depth, size, numinputs int, keywords []string) (equation) {
	var eq equation

	eq.root = NodeGrowConstructor(depth, size, numinputs, keywords)

	return eq
}

func (ind *individual) getNodes() []*Node {
	return getNodes(ind.eq.root)
}

func getNodes(n *Node) []*Node {
	set := []*Node{}
	if n != nil {
		set = append(set, n)

		set = append(set, getNodes(n.left)...)
		set = append(set, getNodes(n.right)...)
	}
	return set
}

func (ind *individual) treeDepth() float64 {
	return ind.eq.root.treeDepth(0)
}

func (n *Node) treeDepth(depth float64) float64 {
	if n == nil {
		return depth
	}
	depth+=1
	return math.Max(n.left.treeDepth(depth), n.right.treeDepth(depth))
}

type Node struct {
	leaf	bool
	value	string

	left	*Node
	right	*Node
}

func NodeDefaultConstructor() (*Node) {
	n := &Node{}

	return n
}

func NodeConstructor(depth, numinputs int, keywords []string) (*Node) {
	n := NodeDefaultConstructor()
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

func NodeGrowConstructor(depth, size, numinputs int, keywords []string) (*Node) {
	n := NodeDefaultConstructor()
	
	var choice int
	if size > 0 {
		choice = rand.Int()%size
	} else {
		choice = 0
	}

	if depth <= 1 && choice > 1 {
		n.leaf = true
	}

	operators := []string{"+","-","*"}

	if n.leaf {
		choice := rand.Int()%100
		if choice < 33 {
			n.value = "v" + strconv.Itoa(rand.Int()%numinputs) + keywords[rand.Int()%len(keywords)]
		} else if choice < 66 {
			n.value = strconv.FormatFloat((1-rand.Float64())*5,'f',-1, 64)
		} else {
			n.value = strconv.Itoa(rand.Int()%4 + 1)
		}
	} else {
		n.value = operators[rand.Int()%len(operators)]

		depth--
		size++
		n.left = NodeGrowConstructor(depth, size, numinputs, keywords)
		n.right = NodeGrowConstructor(depth, size, numinputs, keywords) 
	}

	return n
}

var mu = 100
var lamda = 50
var recombRate = .3
var mutateRate = .2

func runGp(depth, numInputs int, keywords []string, materials [][]metal, goal string) (string, float64) {
	pop := make([]individual, mu)

	tmpGuarantee := make(chan bool, len(pop))
	for i := range pop {
		pop[i].eq = genEquation(depth, numInputs, keywords)
		go pop[i].getFitness(materials, goal, tmpGuarantee)
	}

	for _ = range pop {
		_ = <- tmpGuarantee
	}

	sort.Sort(byScore(pop))
	bestInd := pop[0]

	bestCount := -1

	for i := 0; i < 1000 && bestCount < 35; i++ {
		if i % 25 == 0 {
			fmt.Printf("gen %v\n", i)
		}
		kids := make([]individual, lamda)

		guarantee := make(chan bool, len(kids))
		for j := 0; j < len(kids); j++ {
			if i%25 == 0 && i > 0 {
				kids[j].eq = genEquation(depth, numInputs, keywords)
				guarantee <- true
			} else {
				p1, p2 := proportionSelect(pop)

				go kids[j].recombination(pop[p1], pop[p2], depth, numInputs, keywords, guarantee)
			}
		}

		for j := 0; j < len(kids); j++ {
			_ = <- guarantee
		}


		for j := 0; j < len(kids); j++ {
			go kids[j].getFitness(materials, goal, guarantee)
		}

		for j := 0; j < len(kids); j++ {
			_ = <- guarantee
		}

		pop = append(pop, kids...)

		sort.Sort(byScore(pop))
		if bestInd.fitness > pop[0].fitness {
			bestInd = pop[0]
			bestCount = 0
			fmt.Println("new best")
			fmt.Printf("gen %v fitness %v\n", i, bestInd.fitness)
			bestInd.eq.Fancy()

			if bestInd.fitness - bestInd.treeDepth() == 0 {
				break
			}
		} else {
			bestCount++
		}
		pop = pop[:mu]
	}

	return bestInd.eq.toString(), (bestInd.fitness - bestInd.treeDepth())
}

func (ind *individual) getFitness(materials [][]metal, goal string, guarantee chan bool) {
	ind.fitness = 0.0
	// counter := 1

	for mx1 := range materials {
	for mx2 := range materials {
		// if mx1 != mx2 {
			for my1 := range materials[mx1] {
			for my2 := range materials[mx2] {
				mat1 := materials[mx1][my1]
				mat2 := materials[mx2][my2] 
				
				parameters := make(map[string]interface{})

				for k,v := range mat1.attributes {
					tmp := "v0" + k
					parameters[tmp] = v
				}
				for k,v := range mat2.attributes {
					tmp := "v1" + k
					parameters[tmp] = v
				}

				expression, err := govaluate.NewEvaluableExpression(ind.eq.toString())
				if err != nil {
					panic(err)
				}
				tmp, err := expression.Evaluate(parameters)
				if err != nil {
					panic(err)
				}
				have := tmp.(float64)

				expression, err = govaluate.NewEvaluableExpression(goal)
				if err != nil {
					panic(err)
				}
				tmp, err = expression.Evaluate(parameters)
				if err != nil {
					panic(err)
				}
				want := tmp.(float64)

				//SSE
				ind.fitness += math.Pow(want - have, 2.0)
				// counter++
			}
			}
		// }
	}
	}
	// ind.fitness /= float64(counter)

	ind.fitness += ind.treeDepth()

	guarantee <- true
}

func proportionSelect(pop []individual) (int, int) {
	p1 := -1
	p2 := -1

	totalScore := 0.0

	for i := 0; i < len(pop); i++ {
		totalScore += pop[i].fitness * float64(len(pop) - i)
	}

	r := rand.Float64() * totalScore

	tmpScore := 0.0

	//select parent 1
	for i, p := range pop {
		tmpScore += p.fitness * float64(len(pop) - i)

		if totalScore - tmpScore < r {
			p1 = i
			p2 = p1
			break
		}
	}

	//select different parent for parent 2
	for p2 == p1 {
		r := rand.Float64() * totalScore

		tmpScore := 0.0

		for i, p := range pop {
			tmpScore += p.fitness * float64(len(pop) - i)

			if totalScore - tmpScore < r {
				p2 = i
				break
			}
		}
	}

	return p1, p2
}


func (kid *individual) recombination(parent1 individual, parent2 individual, depth, numinputs int, keywords []string, guarantee chan bool) {
	choices := []*Node{}
	var point1 *Node
	var point2 *Node

	if rand.Int() % 2 == 0 {
		kid.copyTree(parent1)
		choices = kid.getNodes()

		if len(choices) <= 0 {
			point1 = genEquation(depth, numinputs, keywords).root
		} else {
			point1 = choices[rand.Int()%len(choices)]
		}

		choices = []*Node{}
		choices = parent2.getNodes()
		if len(choices) <= 0 {
			point2 = genEquation(depth, numinputs, keywords).root
		} else {
			point2 = choices[rand.Int()%len(choices)]
		}

		*point1 = *point2.copyTree()
	} else {
		kid.copyTree(parent2)
		choices = kid.getNodes()

		if len(choices) <= 0 {
			point1 = genEquation(depth, numinputs, keywords).root
		} else {
			point1 = choices[rand.Int()%len(choices)]
		}

		choices = []*Node{}
		choices = parent1.getNodes()
		if len(choices) <= 0 {
			point2 = genEquation(depth, numinputs, keywords).root
		} else {
			point2 = choices[rand.Int()%len(choices)]
		}

		*point1 = *point2.copyTree()
	}

	kid.mutation(depth, numinputs, keywords)
	guarantee <- true
}

func (kid *individual) mutation(depth, numinputs int, keywords []string) {
	if rand.Float64() < mutateRate {
		choices := kid.getNodes()

		var point1 *Node 
		if len(choices) <= 0 {
			point1 = genEquation(depth, numinputs, keywords).root
		} else {
			point1 = choices[rand.Int()%len(choices)]	
		}

		tmpEq := growEquation(depth, 2, numinputs, keywords)
		*point1 = *tmpEq.root.copyTree()
	}
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
//end gp


func main() {
	keywords := []string{"hardness","hardVariance","corrosion","corrVariance","conductivity","condVariance"}
	depth := 4
	numinputs := 2
	files := []string{"plating.txt", "smelting.txt", "condTreat.txt"}
	// files := []string{"plating.txt"}
	genEq.CreateFiles(keywords, files, depth, numinputs)

	//generate materials for testing
	numinputs = 4
	numtypes := 4
	materials := make([][]metal, numinputs)
	for i := range materials {
		materials[i] = make([]metal, numtypes)
	}
	for _, v := range materials {
		for key := range v {
			v[key] = metalConstructor()
			for _, name := range keywords {
				v[key].attributes[name] = rand.Float64()
			}
		}
	}

	var goalString string
	var gpFile string
	var goalKeyword string
	for _, filename := range files {
		gpFile = "gp" + filename
		file, err := os.Open(filename)
		if err != nil {
			panic("opening file " + filename)
		}
		defer file.Close()
		tmp, err := os.Create(gpFile)
		tmp.Close()


		fmt.Println("File: ", filename)
		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			if err != nil {
				panic("error scanning")
			}
			goalKeyword = scanner.Text()
			scanner.Scan()
			goalString = scanner.Text()
			fmt.Printf("key %v\n", goalKeyword)
			fmt.Printf("eq %v\n", goalString)

			//gp to estimate func
			bestString, bestVal := runGp(depth, 2, keywords, materials, goalString)


			//record best gp
			gpF, err := os.OpenFile(gpFile, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				panic("error opening " +gpFile)
			}

			_, err = gpF.WriteString(goalKeyword+"\n")
			_, err = gpF.WriteString(goalString+"\n")
			_, err = gpF.WriteString(bestString+"\n")

			bestValString := strconv.FormatFloat(bestVal,'f',-1,64)
			_, err = gpF.WriteString(bestValString+"\n")
			gpF.Close()
		}
	} 
}