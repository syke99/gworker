package gworker

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type TestStruct struct {
	gChan    chan string
	Greeting string
}

func Work(data TestStruct, params []TestStruct, errChan chan error) {

	time.Sleep(time.Second * 1)

	switch data.Greeting {
	case "Hello":
		data.gChan <- fmt.Sprint("This greeting is English")
	case "Bonjour":
		data.gChan <- fmt.Sprint("This greeting is French")
	case "Hola":
		data.gChan <- fmt.Sprint("This greeting is Spanish")
	case "Ciao":
		data.gChan <- fmt.Sprint("This greeting is Italian")
	case "Ni Hao":
		data.gChan <- fmt.Sprint("This greeting is Mandarin")
	case "Kon'nichiwa":
		data.gChan <- fmt.Sprint("This greeting is Japanese")
	}
}

func TestNewPool(t *testing.T) {
	// Arrange
	greetingChan := make(chan string, 6)

	ds1 := TestStruct{Greeting: "Hello", gChan: greetingChan}
	ds2 := TestStruct{Greeting: "Bonjour", gChan: greetingChan}
	ds3 := TestStruct{Greeting: "Hola", gChan: greetingChan}
	ds4 := TestStruct{Greeting: "Ciao", gChan: greetingChan}
	ds5 := TestStruct{Greeting: "Ni Hao", gChan: greetingChan}
	ds6 := TestStruct{Greeting: "Kon'nichiwa", gChan: greetingChan}

	data := []TestStruct{ds1, ds2, ds3, ds4, ds5, ds6}

	// Act
	pool, err := NewPool(data, Work, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, pool)
}

func TestNewPool_MissingWorkerFunc(t *testing.T) {
	// Arrange
	greetingChan := make(chan string, 6)

	ds1 := TestStruct{Greeting: "Hello", gChan: greetingChan}
	ds2 := TestStruct{Greeting: "Bonjour", gChan: greetingChan}
	ds3 := TestStruct{Greeting: "Hola", gChan: greetingChan}
	ds4 := TestStruct{Greeting: "Ciao", gChan: greetingChan}
	ds5 := TestStruct{Greeting: "Ni Hao", gChan: greetingChan}
	ds6 := TestStruct{Greeting: "Kon'nichiwa", gChan: greetingChan}

	data := []TestStruct{ds1, ds2, ds3, ds4, ds5, ds6}

	// Act
	pool, err := NewPool[TestStruct, any](data, nil, nil)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, missingWorkerFuncError, err)
	assert.Nil(t, pool)
}

func TestPool_Start(t *testing.T) {
	// Arrange
	greetingChan := make(chan string, 6)

	ds1 := TestStruct{Greeting: "Hello", gChan: greetingChan}
	ds2 := TestStruct{Greeting: "Bonjour", gChan: greetingChan}
	ds3 := TestStruct{Greeting: "Hola", gChan: greetingChan}
	ds4 := TestStruct{Greeting: "Ciao", gChan: greetingChan}
	ds5 := TestStruct{Greeting: "Ni Hao", gChan: greetingChan}
	ds6 := TestStruct{Greeting: "Kon'nichiwa", gChan: greetingChan}

	data := []TestStruct{ds1, ds2, ds3, ds4, ds5, ds6}

	// Act
	pool, err := NewPool(data, Work, nil)

	// Assert
	assert.NoError(t, err)

	// Act
	pool.
		Size(3).
		Start(nil)

	counter := 0

	for g := range greetingChan {
		fmt.Println(g)
		counter++
		if counter == 6 {
			close(greetingChan)
		}
	}
}

func TestPool_StartWithAutoRefill(t *testing.T) {
	// Arrange
	greetingChan := make(chan string, 6)

	ds1 := TestStruct{Greeting: "Hello", gChan: greetingChan}
	ds2 := TestStruct{Greeting: "Bonjour", gChan: greetingChan}
	ds3 := TestStruct{Greeting: "Hola", gChan: greetingChan}
	ds4 := TestStruct{Greeting: "Ciao", gChan: greetingChan}
	ds5 := TestStruct{Greeting: "Ni Hao", gChan: greetingChan}
	ds6 := TestStruct{Greeting: "Kon'nichiwa", gChan: greetingChan}

	data := []TestStruct{ds1, ds2, ds3, ds4, ds5, ds6}

	// Act
	pool, err := NewPool(data, Work, nil)

	// Assert
	assert.NoError(t, err)

	// Act
	pool.
		Size(3).
		WithAutoRefill().
		Start(nil)

	counter := 0

	for g := range greetingChan {
		fmt.Println(g)
		counter++
		if counter == 6 {
			close(greetingChan)
		}
	}
}
