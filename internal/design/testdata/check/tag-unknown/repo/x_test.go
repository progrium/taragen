package x

func TestX(t *testing.T) {
	t.Run("one [a, b]", nil)
	t.Run("two [12]", nil)
	t.Run("not a test name [zzz]", nil) // ok: still a name
}

var notAName = "input [zzz]"
