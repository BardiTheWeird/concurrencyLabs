package async

type Promise[T any] struct {
	c chan T
}

func (p *Promise[T]) Await() T {
	return <-p.c
}

func RunAsync[T any](f func() T) Promise[T] {
	c := make(chan T)
	go func() {
		c <- f()
	}()
	return Promise[T]{c}
}
