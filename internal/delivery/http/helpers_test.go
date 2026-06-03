package httpdelivery

import "testing"

func TestOKResponse(t *testing.T) {
	r := ok("hello")
	if r["success"] != true {
		t.Errorf("success should be true")
	}
	if r["data"] != "hello" {
		t.Errorf("data should be 'hello'")
	}
}

func TestPageResponse(t *testing.T) {
	data := []string{"a", "b", "c"}
	r := pageResponse(data, 25, 2, 10)
	if r["success"] != true {
		t.Errorf("success should be true")
	}
	meta, ok := r["meta"].(map[string]int)
	if !ok {
		t.Fatal("meta should be map[string]int")
	}
	if meta["page"] != 2 || meta["limit"] != 10 || meta["total"] != 25 {
		t.Errorf("meta values incorrect: %+v", meta)
	}
	// totalPages = ceil(25/10) = 3
	if meta["total_pages"] != 3 {
		t.Errorf("total_pages = %d, want 3", meta["total_pages"])
	}
}

func TestPageResponseEdge(t *testing.T) {
	// Exactly fits one page
	r := pageResponse([]int{}, 10, 1, 10)
	meta := r["meta"].(map[string]int)
	if meta["total_pages"] != 1 {
		t.Errorf("10/10 = 1 page, got %d", meta["total_pages"])
	}
	// Zero total
	r2 := pageResponse([]int{}, 0, 1, 10)
	meta2 := r2["meta"].(map[string]int)
	if meta2["total_pages"] != 0 {
		t.Errorf("0 total = 0 pages, got %d", meta2["total_pages"])
	}
}

func TestErrResponse(t *testing.T) {
	r := errResponse("VALIDATION", "bad input")
	if r["success"] != false {
		t.Errorf("success should be false")
	}
	if r["error"] != "VALIDATION" {
		t.Errorf("error code mismatch: %v", r["error"])
	}
	if r["message"] != "bad input" {
		t.Errorf("message mismatch: %v", r["message"])
	}
}
