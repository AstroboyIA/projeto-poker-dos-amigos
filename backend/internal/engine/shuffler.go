package engine

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/walissonpaulo/poker-dos-amigos-backend/internal/models"
)

// Gera o baralho padrão ordenado de 52 cartas
func NewStandardDeck() []models.Card {
	suits := []models.CardSuit{models.SuitSpades, models.SuitHearts, models.SuitDiamonds, models.SuitClubs}
	values := []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14} // 11=J, 12=Q, 13=K, 14=A

	deck := make([]models.Card, 0, 52)
	for _, suit := range suits {
		for _, val := range values {
			code := formatCardCode(val, suit)
			deck = append(deck, models.Card{
				Value: val,
				Suit:  suit,
				Code:  code,
			})
		}
	}
	return deck
}

func formatCardCode(val int, suit models.CardSuit) string {
	valStr := fmt.Sprintf("%d", val)
	switch val {
	case 10:
		valStr = "T"
	case 11:
		valStr = "J"
	case 12:
		valStr = "Q"
	case 13:
		valStr = "K"
	case 14:
		valStr = "A"
	}
	return fmt.Sprintf("%s%s", valStr, suit)
}

// Shuffler com Memória Física Contínua conforme especificado no PDF do Poker dos Amigos
type PhysicalMemoryShuffler struct{}

func NewPhysicalMemoryShuffler() *PhysicalMemoryShuffler {
	return &PhysicalMemoryShuffler{}
}

// Executa o ciclo de embaralhamento sobre o baralho anterior (ou novo caso seja a primeira mão)
// Ciclo:
// 1. CSPRNG gera 30 pares de posições e troca cartas (quebra de padrões herdados)
// 2. 2 Riffle Shuffles simulados
// 3. 3 Cortes em torno de 1/3 do baralho
// 4. 2 Riffle Shuffles simulados
// 5. 1 Corte pela metade
func (s *PhysicalMemoryShuffler) Shuffle(previousDeck []models.Card) ([]models.Card, error) {
	var deck []models.Card
	if len(previousDeck) != 52 {
		deck = NewStandardDeck()
	} else {
		// Clona o baralho anterior para preservar continuidade
		deck = make([]models.Card, 52)
		copy(deck, previousDeck)
	}

	// Passo 1: CSPRNG ajusta posições gerando 30 pares de trocas
	if err := s.csprngSwaps(deck, 30); err != nil {
		return nil, err
	}

	// Passo 2: 2 riffle shuffles simulados
	for i := 0; i < 2; i++ {
		deck = s.riffleShuffle(deck)
	}

	// Passo 3: 3 cortes em torno de 1/3 do baralho
	for i := 0; i < 3; i++ {
		deck = s.cutAroundOneThird(deck)
	}

	// Passo 4: 2 riffle shuffles simulados
	for i := 0; i < 2; i++ {
		deck = s.riffleShuffle(deck)
	}

	// Passo 5: 1 corte pela metade
	deck = s.cutInHalf(deck)

	return deck, nil
}

// CSPRNG gera n pares de índices aleatórios e permuta
func (s *PhysicalMemoryShuffler) csprngSwaps(deck []models.Card, pairs int) error {
	deckLen := int64(len(deck))
	for i := 0; i < pairs; i++ {
		idx1Big, err := rand.Int(rand.Reader, big.NewInt(deckLen))
		if err != nil {
			return err
		}
		idx2Big, err := rand.Int(rand.Reader, big.NewInt(deckLen))
		if err != nil {
			return err
		}
		idx1 := idx1Big.Int64()
		idx2 := idx2Big.Int64()
		deck[idx1], deck[idx2] = deck[idx2], deck[idx1]
	}
	return nil
}

// Simulação de Riffle Shuffle (Gilbert-Shannon-Reeds model)
func (s *PhysicalMemoryShuffler) riffleShuffle(deck []models.Card) []models.Card {
	n := len(deck)
	half := n / 2

	// Variação orgânica no corte da metade (±2 cartas)
	offsetBig, _ := rand.Int(rand.Reader, big.NewInt(5))
	splitPoint := (half - 2) + int(offsetBig.Int64())
	if splitPoint < 10 {
		splitPoint = 10
	} else if splitPoint > n-10 {
		splitPoint = n - 10
	}

	left := deck[:splitPoint]
	right := deck[splitPoint:]

	shuffled := make([]models.Card, 0, n)
	li, ri := 0, 0
	for li < len(left) || ri < len(right) {
		if li >= len(left) {
			shuffled = append(shuffled, right[ri:]...)
			break
		}
		if ri >= len(right) {
			shuffled = append(shuffled, left[li:]...)
			break
		}

		// Probabilidade proporcional ao tamanho da pilha restante
		leftRemaining := len(left) - li
		rightRemaining := len(right) - ri
		totalRemaining := leftRemaining + rightRemaining

		rollBig, _ := rand.Int(rand.Reader, big.NewInt(int64(totalRemaining)))
		if int(rollBig.Int64()) < leftRemaining {
			shuffled = append(shuffled, left[li])
			li++
		} else {
			shuffled = append(shuffled, right[ri])
			ri++
		}
	}

	return shuffled
}

// Corte em torno de 1/3 do baralho (aproximadamente carta 17 ± 3)
func (s *PhysicalMemoryShuffler) cutAroundOneThird(deck []models.Card) []models.Card {
	n := len(deck)
	base := n / 3 // ~17
	jitterBig, _ := rand.Int(rand.Reader, big.NewInt(7)) // 0 a 6
	cutPoint := (base - 3) + int(jitterBig.Int64())
	if cutPoint < 5 {
		cutPoint = 5
	} else if cutPoint > n-5 {
		cutPoint = n - 5
	}

	result := make([]models.Card, 0, n)
	result = append(result, deck[cutPoint:]...)
	result = append(result, deck[:cutPoint]...)
	return result
}

// Corte pela metade (carta 26 ± 2)
func (s *PhysicalMemoryShuffler) cutInHalf(deck []models.Card) []models.Card {
	n := len(deck)
	half := n / 2
	jitterBig, _ := rand.Int(rand.Reader, big.NewInt(5)) // 0 a 4
	cutPoint := (half - 2) + int(jitterBig.Int64())

	result := make([]models.Card, 0, n)
	result = append(result, deck[cutPoint:]...)
	result = append(result, deck[:cutPoint]...)
	return result
}
