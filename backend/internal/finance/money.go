package finance

const ChipsPerReal int64 = 100

func MoneyToChips(cents int64) int64 {
	return cents * ChipsPerReal / 100
}

func ChipsToMoney(chips int64) int64 {
	return chips * 100 / ChipsPerReal
}
