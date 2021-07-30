// errorcheck

package main

func main() {
L: /  // ERROR `missing statement after label`
L: ;
	{
		goto L2
	L2:
	}
	goto L
}
