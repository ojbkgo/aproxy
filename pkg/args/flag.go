package args

type Test struct {
	Name string `json:"name" arg:"required; short:n; desc:name"`
	Skip bool   `json:"skip" arg:"opt; short:s; desc:test; default:false"`
	Val  int    `json:"val" arg:"opt; default:0"`
}

func Parse(o interface{}) error {
	//typ := reflect.TypeOf(o)
	//val := reflect.ValueOf(o)

	return nil
}

func Valid(o interface{}) error {
	return nil
}
