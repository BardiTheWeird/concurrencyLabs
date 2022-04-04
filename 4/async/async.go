package async

type Promise struct {
	c chan interface{}
}

func (p *Promise) Await() interface{} {
	return <-p.c
}

func RunAsync(f func() interface{}) Promise {
	c := make(chan interface{})
	go func() {
		c <- f()
	}()
	return Promise{c}
}
