package reflectx_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/goplus/reflectx"
)

func TestMethodCallDoesNotBlockProviderWrites(t *testing.T) {
	callCtx := reflectx.NewContext()
	t.Cleanup(callCtx.Reset)
	entered := make(chan struct{})
	release := make(chan struct{})

	callType := callCtx.NamedStructOf("test", "ProviderCall", nil)
	callType = callCtx.NewMethodSet(callType, 1, 1)
	err := callCtx.SetMethodSet(callType, []reflectx.Method{
		reflectx.MakeMethod(
			"Wait",
			"test",
			false,
			reflect.TypeOf(func() {}),
			func([]reflect.Value) []reflect.Value {
				close(entered)
				<-release
				return nil
			},
		),
	}, false)
	if err != nil {
		t.Fatal(err)
	}

	callDone := make(chan struct{})
	go func() {
		reflect.New(callType).Elem().MethodByName("Wait").Call(nil)
		close(callDone)
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("dynamic method did not start")
	}

	registerCtx := reflectx.NewContext()
	t.Cleanup(registerCtx.Reset)
	registerType := registerCtx.NamedStructOf("test", "ProviderRegister", nil)
	registerType = registerCtx.NewMethodSet(registerType, 1, 1)
	registerDone := make(chan error, 1)
	go func() {
		registerDone <- registerCtx.SetMethodSet(registerType, []reflectx.Method{
			reflectx.MakeMethod(
				"Run",
				"test",
				false,
				reflect.TypeOf(func() {}),
				func([]reflect.Value) []reflect.Value { return nil },
			),
		}, false)
	}()

	select {
	case err := <-registerDone:
		close(release)
		<-callDone
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		close(release)
		<-callDone
		<-registerDone
		t.Fatal("method registration blocked on a running method call")
	}
}
