package foo

func TestX(t *testing.T) {
	proposaltest.Run(t, "works [base, next]", nil)
	t.Run("untagged", nil)
}
