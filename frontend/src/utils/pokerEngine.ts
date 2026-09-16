import type { Card, CardSuit } from '../types';

export interface HandEvaluation {
  rank: number; // 1 (High Card) to 10 (Royal Flush)
  rankName: string;
  score: number;
  description: string;
}

// Cria um baralho ordenado de 52 cartas
export function createDeck(): Card[] {
  const suits: CardSuit[] = ['S', 'H', 'D', 'C'];
  const values = [2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14];
  const deck: Card[] = [];

  for (const suit of suits) {
    for (const val of values) {
      let valStr = String(val);
      if (val === 10) valStr = 'T';
      if (val === 11) valStr = 'J';
      if (val === 12) valStr = 'Q';
      if (val === 13) valStr = 'K';
      if (val === 14) valStr = 'A';
      deck.push({
        value: val,
        suit,
        code: `${valStr}${suit}`,
      });
    }
  }
  return deck;
}

// Algoritmo de Embaralhamento Contínuo com Memória Física (do PDF)
export function shuffleContinuousDeck(previousDeck: Card[] | null): Card[] {
  let deck = previousDeck && previousDeck.length === 52 ? [...previousDeck] : createDeck();

  // 1. CSPRNG 30 Trocas de Posições
  for (let i = 0; i < 30; i++) {
    const idx1 = Math.floor(Math.random() * deck.length);
    const idx2 = Math.floor(Math.random() * deck.length);
    const temp = deck[idx1];
    deck[idx1] = deck[idx2];
    deck[idx2] = temp;
  }

  // 2. 2 Riffle Shuffles
  for (let i = 0; i < 2; i++) {
    deck = riffleShuffle(deck);
  }

  // 3. 3 Cortes de 1/3
  for (let i = 0; i < 3; i++) {
    deck = cutAroundOneThird(deck);
  }

  // 4. 2 Riffle Shuffles
  for (let i = 0; i < 2; i++) {
    deck = riffleShuffle(deck);
  }

  // 5. 1 Corte na Metade
  deck = cutInHalf(deck);

  return deck;
}

function riffleShuffle(deck: Card[]): Card[] {
  const half = Math.floor(deck.length / 2);
  const jitter = Math.floor(Math.random() * 5) - 2;
  const split = Math.max(10, Math.min(deck.length - 10, half + jitter));

  const left = deck.slice(0, split);
  const right = deck.slice(split);
  const result: Card[] = [];

  let li = 0, ri = 0;
  while (li < left.length || ri < right.length) {
    if (li >= left.length) {
      result.push(...right.slice(ri));
      break;
    }
    if (ri >= right.length) {
      result.push(...left.slice(li));
      break;
    }

    const leftRemaining = left.length - li;
    const rightRemaining = right.length - ri;
    if (Math.random() < leftRemaining / (leftRemaining + rightRemaining)) {
      result.push(left[li++]);
    } else {
      result.push(right[ri++]);
    }
  }
  return result;
}

function cutAroundOneThird(deck: Card[]): Card[] {
  const base = Math.floor(deck.length / 3);
  const jitter = Math.floor(Math.random() * 7) - 3;
  const cut = Math.max(5, Math.min(deck.length - 5, base + jitter));
  return [...deck.slice(cut), ...deck.slice(0, cut)];
}

function cutInHalf(deck: Card[]): Card[] {
  const half = Math.floor(deck.length / 2);
  const jitter = Math.floor(Math.random() * 5) - 2;
  const cut = Math.max(10, Math.min(deck.length - 10, half + jitter));
  return [...deck.slice(cut), ...deck.slice(0, cut)];
}

// Avaliador de Mãos de Poker (Texas Hold'em 7 cartas)
export function evaluate7Cards(cards: Card[]): HandEvaluation {
  if (cards.length < 5) {
    if (cards.length === 2) {
      if (cards[0].value === cards[1].value) {
        return { rank: 2, rankName: 'Par', score: 200 + cards[0].value, description: `Par de ${getCardName(cards[0].value)}` };
      }
      const high = Math.max(cards[0].value, cards[1].value);
      return { rank: 1, rankName: 'Carta Alta', score: 100 + high, description: `Carta Alta ${getCardName(high)}` };
    }
    return { rank: 1, rankName: 'Carta Alta', score: 100, description: 'Carta Alta' };
  }

  // Agrupamentos por valor e naipe
  const valCounts: { [val: number]: number } = {};
  const suitCounts: { [suit: string]: Card[] } = {};

  for (const c of cards) {
    valCounts[c.value] = (valCounts[c.value] || 0) + 1;
    if (!suitCounts[c.suit]) suitCounts[c.suit] = [];
    suitCounts[c.suit].push(c);
  }

  const sortedValues = Array.from(new Set(cards.map((c) => c.value))).sort((a, b) => b - a);

  // 1. Flush e Straight Flush
  let flushSuit: string | null = null;
  for (const suit in suitCounts) {
    if (suitCounts[suit].length >= 5) {
      flushSuit = suit;
      break;
    }
  }

  if (flushSuit) {
    const flushCards = suitCounts[flushSuit].sort((a, b) => b.value - a.value);
    const flushValues = Array.from(new Set(flushCards.map((c) => c.value))).sort((a, b) => b - a);
    const sfHigh = getStraightHigh(flushValues);
    if (sfHigh) {
      if (sfHigh === 14) {
        return { rank: 10, rankName: 'Royal Flush', score: 1000, description: 'Royal Flush Imbatível!' };
      }
      return { rank: 9, rankName: 'Straight Flush', score: 900 + sfHigh, description: `Straight Flush para o ${getCardName(sfHigh)}` };
    }
  }

  // 2. Quadra (Four of a Kind)
  for (const val in valCounts) {
    if (valCounts[val] === 4) {
      return { rank: 8, rankName: 'Quadra', score: 800 + Number(val), description: `Quadra de ${getCardName(Number(val))}` };
    }
  }

  // 3. Full House
  const triples: number[] = [];
  const pairs: number[] = [];
  for (const val in valCounts) {
    const v = Number(val);
    if (valCounts[v] === 3) triples.push(v);
    else if (valCounts[v] === 2) pairs.push(v);
  }
  triples.sort((a, b) => b - a);
  pairs.sort((a, b) => b - a);

  if (triples.length >= 2 || (triples.length >= 1 && pairs.length >= 1)) {
    const mainTriple = triples[0];
    const mainPair = triples.length >= 2 ? triples[1] : pairs[0];
    return {
      rank: 7,
      rankName: 'Full House',
      score: 700 + mainTriple * 2 + mainPair,
      description: `Full House de ${getCardName(mainTriple)} com ${getCardName(mainPair)}`,
    };
  }

  // 4. Flush
  if (flushSuit) {
    const high = suitCounts[flushSuit][0].value;
    return { rank: 6, rankName: 'Flush', score: 600 + high, description: `Flush com Carta Alta ${getCardName(high)}` };
  }

  // 5. Straight (Sequência)
  const straightHigh = getStraightHigh(sortedValues);
  if (straightHigh) {
    return { rank: 5, rankName: 'Sequência (Straight)', score: 500 + straightHigh, description: `Sequência para o ${getCardName(straightHigh)}` };
  }

  // 6. Trinca (Three of a Kind)
  if (triples.length >= 1) {
    return { rank: 4, rankName: 'Trinca', score: 400 + triples[0], description: `Trinca de ${getCardName(triples[0])}` };
  }

  // 7. Dois Pares (Two Pair)
  if (pairs.length >= 2) {
    return {
      rank: 3,
      rankName: 'Dois Pares',
      score: 300 + pairs[0] * 2 + pairs[1],
      description: `Dois Pares de ${getCardName(pairs[0])} e ${getCardName(pairs[1])}`,
    };
  }

  // 8. Um Par (One Pair)
  if (pairs.length === 1) {
    return { rank: 2, rankName: 'Um Par', score: 200 + pairs[0], description: `Par de ${getCardName(pairs[0])}` };
  }

  // 9. Carta Alta
  const high = sortedValues[0] || 14;
  return { rank: 1, rankName: 'Carta Alta', score: 100 + high, description: `Carta Alta ${getCardName(high)}` };
}

function getStraightHigh(values: number[]): number | null {
  const vals = [...values];
  if (vals.includes(14)) vals.push(1); // Ás pode ser baixo (A-2-3-4-5)

  for (let i = 0; i <= vals.length - 5; i++) {
    const slice = vals.slice(i, i + 5);
    if (
      slice[0] - slice[1] === 1 &&
      slice[1] - slice[2] === 1 &&
      slice[2] - slice[3] === 1 &&
      slice[3] - slice[4] === 1
    ) {
      return slice[0];
    }
  }
  return null;
}

function getCardName(value: number): string {
  switch (value) {
    case 14: return 'Ás';
    case 13: return 'Rei';
    case 12: return 'Dama';
    case 11: return 'Valete';
    case 10: return 'Dez';
    default: return String(value);
  }
}
