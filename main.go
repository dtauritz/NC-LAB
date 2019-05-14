package main

import (
	"fmt"
	"bufio"
	"os"
	"math"
	"math/rand"
	"strconv"
	"sort"
	"strings"
	"github.com/Knetic/govaluate"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"

	"image/color"
)

type metal struct {
	attributes		map[string]float64
}

//initializes a metal object
func metalConstructor() metal {
	var result metal
	result.attributes = make(map[string]float64)
	return result
}

//string output for a metal
func (m metal) String() string {
	return fmt.Sprintf("Hardness %v (%v) Conductivity %v (%v) Corrosion %v (%v)",
		m.attributes["hardness"], m.attributes["hardVariance"], m.attributes["conductivity"], m.attributes["condVariance"], m.attributes["corrosion"],
		m.attributes["corrVariance"])
}

//finds which file to read from
func readMetal(kind, version int) (metal) {
	var result metal

	if kind == 0 {
		result = metalFile("goal.txt")
	} else {
		filePath := strconv.Itoa(kind) + "_" + strconv.Itoa(version) + ".txt"
		result = metalFile(filePath) 
	}

	return result
}

//reads and stores the 'metal' given by the filePath
func metalFile(filePath string) metal {
	result := metalConstructor()

	file, err := os.Open(filePath)
	if err != nil {
		panic(0)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		if err != nil {
			panic(0)
		}
		key := scanner.Text()
		scanner.Scan()
		val := scanner.Text()
		tmp, err := strconv.ParseFloat(val, 64)
		if err != nil {
			panic(1)
		}
		result.attributes[key] = tmp
	}

	return result
}

//opens a transformation file and creates the transformation output
func generalFunc(inputs []string,filename string, output string, materials map[string]metal) {
	result := metalConstructor()

	parameters := make(map[string]interface{})

	for ind,name := range inputs {
		for k,v := range materials[name].attributes {
			tmp := "v" + strconv.Itoa(ind) + k
			parameters[tmp] = v
		}
	}

	file, err := os.Open(filename)
	if err != nil {
		panic("opening file " + filename)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		if err != nil {
			panic("error scanning")
		}
		key := scanner.Text()
		scanner.Scan()
		equation := scanner.Text()
		expression, err := govaluate.NewEvaluableExpression(equation)
		if err != nil {
			panic("error making eval")
		}
		tmp, err := expression.Evaluate(parameters)
		if err != nil {
			panic("error evaling")
		}
		result.attributes[key] = tmp.(float64)
	}

	materials[output] = result
}


//runs through the ordering stated by "order.txt" and makes all
//transformation materials
func blackBox(materials map[string]metal) metal {

	file, err := os.Open("order.txt")
	if err != nil {
		fmt.Println("the machine order file could not be opened")
		panic(err)	
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	var line string
	err = nil

	for err == nil {
		line, err = reader.ReadString('\n')
	    line = strings.Replace(line, "\n", "", -1)
		if line != "new row" {
			initSplit := strings.Split(line, "->")

			inPreSplit := initSplit[0]
			filesPreSplit := initSplit[1]
			outPreSplit := initSplit[2]

			inputs := strings.Split(inPreSplit, ",")

			files := strings.Split(filesPreSplit, ",")

			outputs := strings.Split(outPreSplit, ",")

			for k := range files {
				generalFunc(inputs,files[k],outputs[k],materials)
			}
		}
	}

	result := materials["final"]

	return result
}


// ea

//sorting by Pareto rank
type byScore []permutation

func (s byScore) Len() int {
	return len(s)
}

func (s byScore) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byScore) Less(i, j int) bool {
	return s[i].pareto < s[j].pareto
}

//sorting by fitness1
type byFit1 []permutation

func (s byFit1) Len() int {
	return len(s)
}

func (s byFit1) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byFit1) Less(i, j int) bool {
	return s[i].fitness < s[j].fitness
}

//EA variables for population sizes and modifier chances
var mu = 20
var lambda = 25
var recombRate = .5
var mutateRate = .25


type permutation struct {
	assignment	[]int

	finalMetal	metal

	fitness		float64 //accuracy
	fitness2	float64 //affordability

	pareto 		int
}

//randomly generate permutation
func (perm *permutation) genPermutation() {
	perm.assignment = make([]int, 4)

	//each choice has options 1-4
	for i := 0; i < len(perm.assignment); i++ {
		perm.assignment[i] = rand.Int()%4 + 1
	}
}

//makes a standardized output for a permutation
func (perm *permutation) String() string {
	tmpStr := "["

	for i := 0; i < len(perm.assignment); i++ {
		tmpStr = tmpStr + strconv.Itoa(perm.assignment[i]) + ", " 
	}

	tmpStr = tmpStr + "]"

	return tmpStr
}

//creates new population member from 2 parents. guarantee is for parallelization
func (kid *permutation) recombination(parent1 permutation, parent2 permutation, guarantee chan bool) {
	kid.assignment = make([]int, len(parent1.assignment))

	for i := 0; i < len(kid.assignment); i++ {
		if rand.Float64() < recombRate {
			kid.assignment[i] = parent1.assignment[i]
		} else {
			kid.assignment[i] = parent2.assignment[i]
		}
	}

	kid.mutation(parent1, parent2)
	guarantee <- true
}

//mutates a population member
func (kid *permutation) mutation(parent1 permutation, parent2 permutation) {
	if rand.Float64() < mutateRate {
		kid.assignment[rand.Int()%4] = rand.Int()%4+1
	}
}

//calculates and sets fitness1
func (perm *permutation) getFitness(guarantee chan bool) {
	goal := readMetal(0,0)

	materials := make(map[string]metal)

	for key, val := range perm.assignment {
		materials["in"+strconv.Itoa(key)] = readMetal(key+1, val)
	}

	perm.finalMetal = blackBox(materials)

	//hardness section
	hardRating := 0.0
	if perm.finalMetal.attributes["hardness"] > goal.attributes["hardness"] {
		hardRating = 1.0
	} else {
		hardRating = 1.0 - math.Abs(math.Abs(goal.attributes["hardness"] - perm.finalMetal.attributes["hardness"])/perm.finalMetal.attributes["hardVariance"])
	}
	perm.fitness += hardRating/3.0

	//conductivity section
	condRating := 0.0
	if perm.finalMetal.attributes["conductivity"] > goal.attributes["conductivity"] {
		condRating = 1.0
	} else {
		condRating = 1.0 - math.Abs(math.Abs(goal.attributes["conductivity"] - perm.finalMetal.attributes["conductivity"])/perm.finalMetal.attributes["condVariance"])
	}
	perm.fitness += condRating/3.0

	//corrosion section
	corrRating := 0.0
	if perm.finalMetal.attributes["corrosion"] > goal.attributes["corrosion"] {
		corrRating = 1.0
	} else {
		corrRating = 1.0 - math.Abs(math.Abs(goal.attributes["corrosion"] - perm.finalMetal.attributes["corrosion"])/perm.finalMetal.attributes["corrVariance"])
	}
	perm.fitness += corrRating/3.0


	if math.IsInf(perm.fitness, 1) {
		perm.fitness = math.MaxFloat64
	} else if math.IsInf(perm.fitness, -1) {
		perm.fitness = -math.MaxFloat64
	}

	if perm.fitness > 100 {
		perm.fitness = 100
	} else if perm.fitness < -100{
		perm.fitness = -100
	}

	perm.getFitness2()

	guarantee <- true
}

//calculates and sets the fitness2 value
func (perm *permutation) getFitness2() {
	perm.fitness2 = 0.0

	for i := 0; i < len(perm.assignment); i++ {
		perm.fitness2 += float64(perm.assignment[i]*500)
	}
	perm.fitness2 = 8000.0/perm.fitness2
}

//run EA, returns final Pareto front
func runEA() []permutation {
	//setup Pareto progress graph
	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	p.Title.Text = "Pareto Front"
	p.X.Label.Text = "Accuracy"
	p.Y.Label.Text = "Affordability"
	p.Add(plotter.NewGrid())


	pop := make([]permutation, mu)

	tmpGuarantee := make(chan bool, len(pop))
	for i := 0; i < len(pop); i++ {
		pop[i].genPermutation()
		go pop[i].getFitness(tmpGuarantee)
	}

	for j := 0; j < len(pop); j++ {
		_ = <- tmpGuarantee
	}

	bestFront := setPareto(pop)

	sort.Sort(byScore(pop))

	//if there has been no change in best fitness over 10 generations, cut run
	bestCount := -1

	for i := 0; i < 1000 && bestCount < 10; i++ {
		fmt.Printf("gen: %v\n", i)
		kids := make([]permutation, lambda)

		guarantee := make(chan bool, len(kids))
		for j := 0; j < len(kids); j++ {
			if i%25 == 0 && i > 0 {
				kids[j].genPermutation()
				guarantee <- true
			} else {
				p1, p2 := proportionPareto(pop)

				go kids[j].recombination(pop[p1], pop[p2], guarantee)
			}
		}

		for j := 0; j < len(kids); j++ {
			_ = <- guarantee
		}

		for j := 0; j < len(kids); j++ {
			go kids[j].getFitness(guarantee)
		}

		for j := 0; j < len(kids); j++ {
			_ = <- guarantee
		}

		pop = append(pop, kids...)

		newFront := setPareto(pop)

		pop = pop[:mu]

		changeFront := false

		if len(newFront) != len(bestFront) {
			changeFront = true
		}
		for k := 0; !changeFront && k < len(newFront); k++ {
			inFront := false
			for j := range bestFront {
				if assignmentEquality(newFront[k].assignment,
						bestFront[j].assignment) {
					inFront = true
					break
				}
			}
			if !inFront {
				changeFront = true
				break
			}
		}

		if changeFront {
			bestCount = 0
			bestFront = newFront
			fmt.Printf("Best Fit (gen %d): Front %d\n", i, len(bestFront))
		} else {
			bestCount++
		}

		if i > 0 && i%(20*len(pop)/5) == 0 && recombRate > 0.1 {
			recombRate -= 0.1
		}

		if i == 0 {
			//adding starting Pareto fronts to graph
			sort.Sort(byFit1(bestFront))

			pts := make(plotter.XYs, len(bestFront))
			for i := range pts {
				pts[i].X = bestFront[i].fitness
				pts[i].Y = bestFront[i].fitness2
			}

			lpLine, lpPoints, err := plotter.NewLinePoints(pts)
			if err != nil {
				panic(err)
			}
			rNum := uint8(0)
			gNum := uint8(0)
			bNum := uint8(0)
			lpLine.Color = color.RGBA{R: rNum, G: gNum, B: bNum, A: 128}
			lpPoints.Shape = draw.PyramidGlyph{}
			lpPoints.Color = color.RGBA{R: rNum, G: gNum, B: bNum, A: 128}

			p.Add(lpLine, lpPoints)
			p.Legend.Add("Gen 0", lpLine, lpPoints)
		} else if i%300 == 0 || i == 100 {
			//adding intermediate Pareto fronts to graph
			sort.Sort(byFit1(bestFront))

			pts := make(plotter.XYs, len(bestFront))
			for i := range pts {
				pts[i].X = bestFront[i].fitness
				pts[i].Y = bestFront[i].fitness2
			}

			lpLine, lpPoints, err := plotter.NewLinePoints(pts)
			if err != nil {
				panic(err)
			}
			rNum := uint8(0)
			gNum := uint8(0)
			bNum := uint8(0)
			if i/300%2 == 0 {
				gNum = uint8(16*(i/100)+64)
			} else {
				bNum = uint8(16*(i/100)+64)
			}
			lpLine.Color = color.RGBA{R: rNum, G: gNum, B: bNum, A: 128}
			lpPoints.Shape = draw.BoxGlyph{}
			lpPoints.Color = color.RGBA{R: rNum, G: gNum, B: bNum, A: 128}

			p.Add(lpLine, lpPoints)
			p.Legend.Add("Gen " + strconv.Itoa(i), lpLine, lpPoints)
		}
	}

	fmt.Println("EA Done")

	sort.Sort(byFit1(bestFront))

	//adds final generation to Pareto graph
	pts := make(plotter.XYs, len(bestFront))
	for i := range pts {
		pts[i].X = bestFront[i].fitness
		pts[i].Y = bestFront[i].fitness2
	}

	lpLine, lpPoints, err := plotter.NewLinePoints(pts)
	if err != nil {
		panic(err)
	}
	lpLine.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255}
	lpPoints.Shape = draw.CrossGlyph{}
	lpPoints.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255}

	p.Add(lpLine, lpPoints)
	p.Legend.Add("Final Best", lpLine, lpPoints)
	p.Legend.Top = true

	if err := p.Save(6*vg.Inch, 6*vg.Inch, "points.png"); err != nil {
		panic(err)
	}

	return bestFront
}

//checking that no duplicate solutions are in the Pareto front
func assignmentEquality(a, b []int) bool {
	if (a == nil) != (b == nil) { 
		return false; 
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

//used to find Pareto ranks for the population (pop)
func setPareto(pop []permutation) []permutation {
	for i := range pop {
		pop[i].pareto = 1
	}

	dominates := make([][]int, len(pop))
	
	for i := 0; i < len(pop); i++ {
		for j := 0; j < len(pop); j++ {
			if (pop[i].fitness > pop[j].fitness && pop[i].fitness2 >= pop[j].fitness2) || (pop[i].fitness >= pop[j].fitness && pop[i].fitness2 > pop[j].fitness2) {
				dominates[i] = append(dominates[i], j)
			}
		}
	}

	for i := range pop {
		incPareto(pop, dominates, i)
	}

	front := make([]permutation, 0)

	for i := range pop {
		if pop[i].pareto == 1 {
			dupe := false
			for j := range front {
				if assignmentEquality(pop[i].assignment, front[j].assignment) {
					dupe = true
				}
			}
			if !dupe {
				front = append(front, pop[i])
			}
		}
	}

	return front
}

//helper function for finding Pareto rank
func incPareto(pop []permutation, dom [][]int, index int) {
	for i := 0; i < len(dom[index]); i++ {
		if pop[index].pareto >= pop[dom[index][i]].pareto{
			pop[dom[index][i]].pareto = pop[index].pareto + 1
			incPareto(pop, dom, dom[index][i])
		}
	}
}

//selects parent indices proportionally based on the Pareto rank
func proportionPareto(pop []permutation) (int, int) {
	p1 := -1
	p2 := -1

	total := 0
	sum := 0

	for _, p := range pop {
		total += int(1.0/float64(p.pareto)*10000.0)
	}

	choice := rand.Int()%total

	for i, p := range pop {
		sum += int(1.0/float64(p.pareto)*10000.0)
	
		if sum > choice {
			p1 = i
			p2 = p1
			break
		}
	}

	for p2 == p1 {
		sum = 0
		choice = rand.Int()%total
		for i, p := range pop {
			sum += int(1.0/float64(p.pareto)*10000.0)
		
			if sum > choice {
				p2 = i
				break
			}
		}
	}

	return p1, p2
}

// end ea


//runs the program and EA, outputs a sorted list of pareto front.
func main() {
	rand.Seed(1)

	front := runEA()

	fmt.Println()
	goal := readMetal(0,0)
	fmt.Println("goal: ", goal, "\n")

	for i := 0; i < len(front); i++ {
		fmt.Printf("i: %v perm %v accuracy %v affordability %v\n", i,
			front[i].assignment, front[i].fitness, front[i].fitness2)
		fmt.Println(front[i].finalMetal, "\n")
	}
}