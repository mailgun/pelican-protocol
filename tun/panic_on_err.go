package gohttptun

func panicOn(err error) {
	if err != nil {
		panic(err)
	}
}
