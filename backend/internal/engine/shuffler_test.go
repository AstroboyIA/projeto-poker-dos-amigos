package engine

import (
	"testing"
)

func TestPhysicalMemoryShuffler(t *testing.T) {
	shuffler := NewPhysicalMemoryShuffler()

	// 1. Primeira mão (baralho inicial)
	deckA, err := shuffler.Shuffle(nil)
	if err != nil {
		t.Fatalf("Erro no shuffle inicial: %v", err)
	}

	if len(deckA) != 52 {
		t.Fatalf("Tamanho esperado do baralho: 52, obteve: %d", len(deckA))
	}

	// Verifica se todas as cartas são únicas
	seen := make(map[string]bool)
	for _, c := range deckA {
		if seen[c.Code] {
			t.Fatalf("Carta duplicada encontrada: %s", c.Code)
		}
		seen[c.Code] = true
	}

	// 2. Segunda mão (baseada no baralho final A)
	deckB, err := shuffler.Shuffle(deckA)
	if err != nil {
		t.Fatalf("Erro no shuffle contínuo B: %v", err)
	}
	if len(deckB) != 52 {
		t.Fatalf("Tamanho esperado do baralho B: 52, obteve: %d", len(deckB))
	}

	// 3. Terceira mão (baseada no baralho final B)
	deckC, err := shuffler.Shuffle(deckB)
	if err != nil {
		t.Fatalf("Erro no shuffle contínuo C: %v", err)
	}
	if len(deckC) != 52 {
		t.Fatalf("Tamanho esperado do baralho C: 52, obteve: %d", len(deckC))
	}

	// Verifica se a ordem mudou entre os baralhos
	identicalCount := 0
	for i := 0; i < 52; i++ {
		if deckA[i].Code == deckB[i].Code {
			identicalCount++
		}
	}

	if identicalCount > 15 {
		t.Errorf("Muitas cartas na mesma posição entre mãos consecutivas: %d", identicalCount)
	}
}
