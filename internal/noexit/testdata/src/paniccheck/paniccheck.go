package paniccheck

func foo() {
	panic("something bad") // want `avoid using panic`
}
