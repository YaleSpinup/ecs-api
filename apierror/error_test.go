package apierror

import (
	"errors"
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	out := New(ErrBadRequest, "bad request", errors.New("fail"))
	if e := reflect.TypeOf(out).String(); e != "apierror.Error" {
		t.Errorf("expect type to be apierror.Error, got %s", e)
	}
}

func TestError(t *testing.T) {
	out := New(ErrBadRequest, "bad request", errors.New("fail"))
	if out.Error() != out.String() {
		t.Errorf("expected '%s', got '%s'", out.String(), out)
	}
}

func TestString(t *testing.T) {
	out := New(ErrBadRequest, "bad request", errors.New("fail"))
	expect := "BadRequest: bad request (fail)"
	if out.String() != expect {
		t.Errorf("expected '%s', got '%s'", expect, out)
	}
}
