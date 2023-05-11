package gworker

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type TestStruct struct {
	Greeting string
}

func Work(data any, params []any) {
	gChan := params[0].(chan any)

	d := data.(TestStruct)

	time.Sleep(time.Second * 1)

	switch d.Greeting {
	case "Hello":
		gChan <- fmt.Sprint("This greeting is English")
	case "Bonjour":
		gChan <- fmt.Sprint("This greeting is French")
	case "Hola":
		gChan <- fmt.Sprint("This greeting is Spanish")
	case "Ciao":
		gChan <- fmt.Sprint("This greeting is Italian")
	case "Ni Hao":
		gChan <- fmt.Sprint("This greeting is Mandarin")
	case "Kon'nichiwa":
		gChan <- fmt.Sprint("This greeting is Japanese")
	}
}

func TestNewPool(t *testing.T) {
	// Arrange
	ds1 := TestStruct{Greeting: "Hello"}
	ds2 := TestStruct{Greeting: "Bonjour"}
	ds3 := TestStruct{Greeting: "Hola"}
	ds4 := TestStruct{Greeting: "Ciao"}
	ds5 := TestStruct{Greeting: "Ni Hao"}
	ds6 := TestStruct{Greeting: "Kon'nichiwa"}

	data := []TestStruct{ds1, ds2, ds3, ds4, ds5, ds6}

	greetingChan := make(chan any, 6)

	// Act
	pool, err := NewPool(data, Work, []chan any{greetingChan}, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, pool)
}

func TestNewPool_MissingWorkerFunc(t *testing.T) {
	// Arrange
	ds1 := TestStruct{Greeting: "Hello"}
	ds2 := TestStruct{Greeting: "Bonjour"}
	ds3 := TestStruct{Greeting: "Hola"}
	ds4 := TestStruct{Greeting: "Ciao"}
	ds5 := TestStruct{Greeting: "Ni Hao"}
	ds6 := TestStruct{Greeting: "Kon'nichiwa"}

	data := []TestStruct{ds1, ds2, ds3, ds4, ds5, ds6}

	greetingChan := make(chan any, 6)

	// Act
	pool, err := NewPool(data, nil, []chan any{greetingChan}, nil)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, missingWorkerFuncError, err)
	assert.Nil(t, pool)
}

func TestPool_Start(t *testing.T) {
	// Arrange
	ds1 := TestStruct{Greeting: "Hello"}
	ds2 := TestStruct{Greeting: "Bonjour"}
	ds3 := TestStruct{Greeting: "Hola"}
	ds4 := TestStruct{Greeting: "Ciao"}
	ds5 := TestStruct{Greeting: "Ni Hao"}
	ds6 := TestStruct{Greeting: "Kon'nichiwa"}

	data := []TestStruct{ds1, ds2, ds3, ds4, ds5, ds6}

	greetingChan := make(chan any, 6)

	// Act
	pool, err := NewPool(data, Work, []chan any{greetingChan}, nil)

	// Assert
	assert.NoError(t, err)

	// Act
	pool.
		Size(3).
		Start(nil)

	counter := 0

	for g := range greetingChan {
		fmt.Println(g.(string))
		counter++
		if counter == 6 {
			close(greetingChan)
		}
	}
}

func TestPool_StartWithAutoRefill(t *testing.T) {
	// Arrange
	ds1 := TestStruct{Greeting: "Hello"}
	ds2 := TestStruct{Greeting: "Bonjour"}
	ds3 := TestStruct{Greeting: "Hola"}
	ds4 := TestStruct{Greeting: "Ciao"}
	ds5 := TestStruct{Greeting: "Ni Hao"}
	ds6 := TestStruct{Greeting: "Kon'nichiwa"}

	data := []TestStruct{ds1, ds2, ds3, ds4, ds5, ds6}

	greetingChan := make(chan any, 6)

	// Act
	pool, err := NewPool(data, Work, []chan any{greetingChan}, nil)

	// Assert
	assert.NoError(t, err)

	// Act
	pool.
		Size(3).
		WithAutoPoolRefill().
		Start(nil)

	counter := 0

	for g := range greetingChan {
		fmt.Println(g.(string))
		counter++
		if counter == 6 {
			close(greetingChan)
		}
	}
}
