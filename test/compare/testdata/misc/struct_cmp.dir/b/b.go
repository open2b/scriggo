package b

var B1 = struct{ F int }{}

var B2 = struct{ f int }{}

var B3 = struct {
	F int `k:"v"`
}{}
