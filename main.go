package main

import (
	"fmt"
	"bufio"
	"os"
	"math/rand"
	"strconv"
	"sort"
)

type metal struct {
	//lower better for variances
	//higher the better for others

	hardness 		float64 //[0, 100]
	hardVariance	float64

	conductivity	float64 //[0, 100]
	condVariance	float64

	corrosion		float64 //[0, 100]
	corrVariance	float64
}

func (m metal) String() string {
	return fmt.Sprintf("Hardness %v (%v) Conductivity %v (%v) Corrosion %v (%v)",
		m.hardness, m.hardVariance, m.conductivity, m.condVariance, m.corrosion,
		m.corrVariance)
}

// kind=0 for goal alloy
func generateMetal(kind, version int) (metal) {
	var result metal

	switch(kind) {
	case 0:
		result.hardness = 61
		result.conductivity = 71
		result.corrosion = 85
	case 1: // hard metal 1
		result.hardness = 50
		result.conductivity = 30
		result.corrosion = 10
		switch(version){
		case 1:
			result.hardVariance = 8
			result.hardness -= 4
		case 2:
			result.hardVariance = 6
			result.hardness -= 3
		case 3:
			result.hardVariance = 4
			result.hardness -= 2
		case 4:
			result.hardVariance = 2
			result.hardness -= 1
		}
	case 2: // hard metal 2
		result.hardness = 60
		result.conductivity = 20
		result.corrosion = 30
		switch(version){
		case 1:
			result.hardVariance = 6
			result.hardness -= 3
		case 2:
			result.hardVariance = 4
			result.hardness -= 2
		case 3:
			result.hardVariance = 2
			result.hardness -= 1
		case 4:
			result.hardVariance = 1
			result.hardness -= 0.5
		}
	case 3: // conductivity metal
		result.hardness = 30
		result.conductivity = 50
		result.corrosion = 10
		switch(version){
		case 1:
			result.condVariance = 7
			result.conductivity -= 3.5
		case 2:
			result.condVariance = 5
			result.conductivity -= 2.5
		case 3:
			result.condVariance = 3
			result.conductivity -= 1.5
		case 4:
			result.condVariance = 1
			result.conductivity -= 0.5
		}
	case 4: // corrosive metal
		result.hardness = 20
		result.conductivity = 20
		result.corrosion = 70
		switch(version){
		case 1:
			result.corrVariance = 8
			result.corrosion -= 4
		case 2:
			result.corrVariance = 5
			result.corrosion -= 2.5
		case 3:
			result.corrVariance = 4
			result.corrosion -= 2
		case 4:
			result.corrVariance = 3
			result.corrosion -= 1.5
		}
	}

	return result
}

//combines two metals
func smelting(met1, met2 metal) metal {
	var result metal

	result.hardness = (met1.hardness + met2.hardness)/2
	result.hardVariance = met1.hardVariance + met2.hardVariance
	result.hardness += 10 - result.hardVariance

	result.conductivity = (met1.conductivity + met2.conductivity)/2
	result.corrosion = (met1.corrosion + met2.corrosion)/2

	return result
}

//increases hardness
// func hardening(met metal) metal {
	
// }

//increases conductivity with two metals
func conductivityTreat(met1, met2 metal) metal {
	var result metal

	result.hardness = met1.hardness
	result.hardVariance = met1.hardVariance
	result.corrosion = met1.corrosion
	result.corrVariance = met1.corrVariance

	result.condVariance = met2.condVariance
	result.conductivity = met1.conductivity + met2.conductivity - met2.condVariance

	return result
}

//increases corrosion with two metals
func plating(met1, met2 metal) metal {
	var result metal

	result.hardness = met1.hardness
	result.hardVariance = met1.hardVariance
	result.conductivity = met1.conductivity
	result.condVariance = met1.condVariance

	result.corrVariance = met2.corrVariance
	result.corrosion = met1.corrosion + met2.corrosion - met2.corrVariance

	return result
}


func blackBox(in1, in2, in3, in4 metal) metal {
	smelt := smelting(in1, in2)
	conduct := conductivityTreat(smelt, in3)
	result := plating(conduct, in4)

	return result
}


// ea
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


var mu = 10
var lambda = 5
var recombRate = .5
var mutateRate = .25


type permutation struct {
	assignment	[]int

	finalMetal	metal

	fitness		float64 //accuracy
	fitness2	float64 //cost

	pareto 		int
}

func (perm *permutation) genPermutation() {
	perm.assignment = make([]int, 4)

	//each choice has options 1-4
	for i := 0; i < len(perm.assignment); i++ {
		perm.assignment[i] = rand.Int()%4 + 1
	}
}

func (perm *permutation) String() string {
	tmpStr := "["

	for i := 0; i < len(perm.assignment); i++ {
		tmpStr = tmpStr + strconv.Itoa(perm.assignment[i]) + ", " 
	}

	tmpStr = tmpStr + "]"

	return tmpStr
}


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
	// return kid
	guarantee <- true
}

func (kid *permutation) mutation(parent1 permutation, parent2 permutation) {
	if rand.Float64() < mutateRate {
		kid.assignment[rand.Int()%4] = rand.Int()%4+1
	}
}

func find(arr []int, elem int) int {
	index := -1

	for i, e := range arr {
		if e == elem {
			index = i
			break
		}
	}

	return index
}

func (perm *permutation) getFitness(guarantee chan bool) {
	goal := generateMetal(0,0)

	var in1, in2, in3, in4 metal

	in1 = generateMetal(1, perm.assignment[0])
	in2 = generateMetal(2, perm.assignment[1])
	in3 = generateMetal(3, perm.assignment[2])
	in4 = generateMetal(4, perm.assignment[3])

	perm.finalMetal = blackBox(in1, in2, in3, in4)

	//hardness section
	hardRating := 0.0
	if perm.finalMetal.hardness > goal.hardness {
		hardRating = 1.0
	} else {
		hardRating = 1.0 - (goal.hardness - perm.finalMetal.hardness)/perm.finalMetal.hardVariance
	}
	perm.fitness += hardRating/3.0

	//conductivity section
	condRating := 0.0
	if perm.finalMetal.conductivity > goal.conductivity {
		condRating = 1.0
	} else {
		condRating = 1.0 - (goal.conductivity - perm.finalMetal.conductivity)/perm.finalMetal.condVariance
	}
	perm.fitness += condRating/3.0

	//corrosion section
	corrRating := 0.0
	if perm.finalMetal.corrosion > goal.corrosion {
		corrRating = 1.0
	} else {
		corrRating = 1.0 - (goal.corrosion - perm.finalMetal.corrosion)/perm.finalMetal.corrVariance
	}
	perm.fitness += corrRating/3.0


	perm.getFitness2()

	guarantee <- true
}

func (perm *permutation) getFitness2() {
	perm.fitness2 = 0.0

	for i := 0; i < len(perm.assignment); i++ {
		perm.fitness2 += float64(perm.assignment[i]*500)
	}
}

func runEA() []permutation {
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

	//if there has been no change in best fitness over 15 generations, cut run
	bestCount := -1

	for i := 0; i < 10 && bestCount < 10; i++ {
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

		sort.Sort(byScore(pop))

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
	}

	fmt.Println("EA Done")
	// fmt.Println(bestFront)

	return bestFront
}

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

func proportionSelect(pop []permutation) (int, int) {
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

func setPareto(pop []permutation) []permutation {
	for i := range pop {
		pop[i].pareto = 1
	}

	dominates := make([][]int, len(pop))
	
	for i := 0; i < len(pop); i++ {
		for j := 0; j < len(pop); j++ {
			if (pop[i].fitness > pop[j].fitness && pop[i].fitness2 <= pop[j].fitness2) || (pop[i].fitness >= pop[j].fitness && pop[i].fitness2 < pop[j].fitness2) {
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

func incPareto(pop []permutation, dom [][]int, index int) {
	for i := 0; i < len(dom[index]); i++ {
		if pop[index].pareto >= pop[dom[index][i]].pareto{
			pop[dom[index][i]].pareto = pop[index].pareto + 1
			incPareto(pop, dom, dom[index][i])
		}
	}
}

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


func main() {
	rand.Seed(1)

	front := runEA()

	fmt.Println()
	goal := generateMetal(0,0)
	fmt.Println("goal: ", goal, "\n")

	for i := 0; i < len(front); i++ {
		fmt.Printf("i: %v perm %v accuracy %v cost %v\n", i,
			front[i].assignment, front[i].fitness, front[i].fitness2)
		fmt.Println(front[i].finalMetal, "\n")
	}

	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
}