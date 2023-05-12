# Gworker
[![Go Reference](https://pkg.go.dev/badge/github.com/syke99/gworker.svg)](https://pkg.go.dev/github.com/syke99/gworker)
[![Go Reportcard](https://goreportcard.com/badge/github.com/syke99/gworker)](https://goreportcard.com/report/github.com/syke99/gworker)

[//]: # ([![Codecov]&#40;https://codecov.io/gh/syke99/gworker/branch/main/graph/badge.svg?token=KDYH3JO1QI&#41;]&#40;https://codecov.io/gh/syke99/gworker&#41;)
[![LICENSE](https://img.shields.io/github/license/syke99/gworker)](https://pkg.go.dev/github.com/syke99/gworker/blob/master/LICENSE)

Gworker, the most easily configurable Generic worker pool implementation for Go

What problem does Gworker solve?
=====
Oftentimes, you'll want to implement a worker pool for a given collection of datasoruces (usually structs), but also limit how many workers are running at once. But sometimes,
you may want to run the workers in batches, and other times you may want to immediately start a new worker as soon as another finishes. And while all of these implementations
are very similar, they require slight variations for each one, requiring a user to have a lot of repeated code with only slight changes. With Gworker, you don't need to worry
about all of these implementation details. Instead, with Gworker, you can configure them easily, as well as have the same implementation across all data types thanks to Gworker
being fully generic.

How do I use Gworker?
====

### Installing
To install Gworker in a repo, simply run

```bash
$ go get github.com/syke99/gworker
```

Then you can import the package in any go file you'd like

```go
import "github.com/syke99/gworker"
```

### Basic usage

After importing, simply create a NewPool by passing in your slice of data sources and the
worker func you wish to run for each data point in your slice of data sources, along with
a slice of any values channels accompanied by an error channel if you wish to use them like
so:

```go
// TestStruct is simply an example of a data source
type TestStruct struct {
	// if you need/want to send values back
 // out of the worker func, add a value channel,
	// in this case, gChan
	gChan chan string
	Greeting string
}

// Work is an example worker func to be ran for each data source
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

func main() {
    //      !!IMPORTANT!! 
    // since Gworker defaults to
    // running in batches, you 
    // must use buffered channels
    // to prevent deadlocks from
    // the channels blocking 
    // unless you call 
    // .WithAutoPoolRefill()
    //      !!IMPORTANT!! 
    greetingChan := make(chan string, 6)

    ds1 := TestStruct{Greeting: "Hello", gChan: greetingChan}
    ds2 := TestStruct{Greeting: "Bonjour", gChan: greetingChan}
    ds3 := TestStruct{Greeting: "Hola", gChan: greetingChan}
    ds4 := TestStruct{Greeting: "Ciao", gChan: greetingChan}
    ds5 := TestStruct{Greeting: "Ni Hao", gChan: greetingChan}
    ds6 := TestStruct{Greeting: "Kon'nichiwa", gChan: greetingChan}
    
    data := []TestStruct{ds1, ds2, ds3, ds4, ds5, ds6}
   
    pool, err := NewPool(data, Work, nil)
}
```

After creating your new pool, you can configure the maximum number of worker funcs
that will run at one time, as well as opt for auto-refilling the pool with new worker
funcs as soon as one finished if any data sources are left like so:

```go
// Set the maximum number of worker funcs to run at once with .Size()
// NOTE: if .Size() is not called, Gworker will default to a size of 5
pool.Size(3)

// optionally automatically refill the pool with a new worker func 
// as soon as another finishes by calling .WithAutoPoolRefill()
// NOTE: if .WithAutoPoolRefill() is called, any channels provided
// NOTE: do not need to be buffered
pool.WithAutoPoolRefill()
```

Once your new Gworker pool has been configured, simply call .Start() and pass in any
additional func params you'd like to be used in the worker funcs. If you don't wish
to have any additional params, simply pass in 

```go
    pool.Start()
```

Put it all together:

```go
import "github.com/syke99/gworker"

// TestStruct is simply an example of a data source
type TestStruct struct {
    gChan chan string
    Greeting string
}

// Work is an example worker func to be ran for each data source
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

func main() {
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
```

Who?
====

This library was developed by Quinn Millican ([@syke99](https://github.com/syke99))


## License

This repo is under the MIT license, see [LICENSE](../LICENSE) for details.
