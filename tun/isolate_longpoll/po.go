package main

// print out shortcut
func po(format string, a ...interface{}) {
	if Verbose {
		TSPrintf("\n\n"+format+"\n\n", a...)
	}
}
