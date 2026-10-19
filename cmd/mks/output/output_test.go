package output

import "testing"

type sample struct {
	Name  string            `json:"name" yaml:"name"`
	Count int               `json:"count" yaml:"count"`
	Tags  map[string]string `json:"tags" yaml:"tags"`
}

func TestRender(t *testing.T) {
	v := sample{Name: "test", Count: 2, Tags: map[string]string{"a": "b"}}

	if _, err := Render("", v); err != nil {
		t.Fatalf("Render('') returned error: %v", err)
	}

	printed, err := Render(JSON, v)
	if err != nil {
		t.Fatalf("Render(json) returned error: %v", err)
	}
	if !printed {
		t.Fatal("Render(json) did not print")
	}

	printed, err = Render(YAML, v)
	if err != nil {
		t.Fatalf("Render(yaml) returned error: %v", err)
	}
	if !printed {
		t.Fatal("Render(yaml) did not print")
	}

	if _, err := Render("xml", v); err == nil {
		t.Fatal("Render(xml) should return an error")
	}
}
