package interfacename

import (
	"math/rand"
	"strconv"
)

type NameGenerator interface {
	Generate(prefix string, maxLength int) string
}

type RandomGenerator struct {
}

func (rg *RandomGenerator) Generate(prefix string, maxLength int) string {
	randomID := rand.Intn(1000)
	randomName := prefix + strconv.Itoa(randomID)
	return randomName
}
