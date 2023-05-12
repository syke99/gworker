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

func Work(data Data[any], params []TestStruct, errChan chan error) {

	time.Sleep(time.Second * 1)

	switch data.Value {
	case "Hello":
		data.OutChan <- fmt.Sprint("This greeting is English")
	case "Bonjour":
		data.OutChan <- fmt.Sprint("This greeting is French")
	case "Hola":
		data.OutChan <- fmt.Sprint("This greeting is Spanish")
	case "Ciao":
		data.OutChan <- fmt.Sprint("This greeting is Italian")
	case "Ni Hao":
		data.OutChan <- fmt.Sprint("This greeting is Mandarin")
	case "Kon'nichiwa":
		data.OutChan <- fmt.Sprint("This greeting is Japanese")
	}
}

func TestNewPool(t *testing.T) {
	// Arrange
	greetingChan := make(chan any, 6)

	ds1 := Data[any]{Value: "Hello", OutChan: greetingChan}
	ds2 := Data[any]{Value: "Bonjour", OutChan: greetingChan}
	ds3 := Data[any]{Value: "Hola", OutChan: greetingChan}
	ds4 := Data[any]{Value: "Ciao", OutChan: greetingChan}
	ds5 := Data[any]{Value: "Ni Hao", OutChan: greetingChan}
	ds6 := Data[any]{Value: "Kon'nichiwa", OutChan: greetingChan}

	data := []Data[any]{ds1, ds2, ds3, ds4, ds5, ds6}

	// Act
	pool, err := NewPool[Data[any]](data, Work, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, pool)
}

func TestNewPool_MissingWorkerFunc(t *testing.T) {
	// Arrange
	greetingChan := make(chan any, 6)

	ds1 := Data[any]{Value: "Hello", OutChan: greetingChan}
	ds2 := Data[any]{Value: "Bonjour", OutChan: greetingChan}
	ds3 := Data[any]{Value: "Hola", OutChan: greetingChan}
	ds4 := Data[any]{Value: "Ciao", OutChan: greetingChan}
	ds5 := Data[any]{Value: "Ni Hao", OutChan: greetingChan}
	ds6 := Data[any]{Value: "Kon'nichiwa", OutChan: greetingChan}

	data := []Data[any]{ds1, ds2, ds3, ds4, ds5, ds6}

	// Act
	pool, err := NewPool[Data[any], any](data, nil, nil)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, missingWorkerFuncError, err)
	assert.Nil(t, pool)
}

func TestPool_Start(t *testing.T) {
	// Arrange
	greetingChan := make(chan any, 6)

	ds1 := Data[any]{Value: "Hello", OutChan: greetingChan}
	ds2 := Data[any]{Value: "Bonjour", OutChan: greetingChan}
	ds3 := Data[any]{Value: "Hola", OutChan: greetingChan}
	ds4 := Data[any]{Value: "Ciao", OutChan: greetingChan}
	ds5 := Data[any]{Value: "Ni Hao", OutChan: greetingChan}
	ds6 := Data[any]{Value: "Kon'nichiwa", OutChan: greetingChan}

	data := []Data[any]{ds1, ds2, ds3, ds4, ds5, ds6}

	// Act
	pool, err := NewPool[Data[any]](data, Work, nil)

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
	greetingChan := make(chan any, 6)

	ds1 := Data[any]{Value: "Hello", OutChan: greetingChan}
	ds2 := Data[any]{Value: "Bonjour", OutChan: greetingChan}
	ds3 := Data[any]{Value: "Hola", OutChan: greetingChan}
	ds4 := Data[any]{Value: "Ciao", OutChan: greetingChan}
	ds5 := Data[any]{Value: "Ni Hao", OutChan: greetingChan}
	ds6 := Data[any]{Value: "Kon'nichiwa", OutChan: greetingChan}

	data := []Data[any]{ds1, ds2, ds3, ds4, ds5, ds6}

	// Act
	pool, err := NewPool[Data[any]](data, Work, nil)

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
