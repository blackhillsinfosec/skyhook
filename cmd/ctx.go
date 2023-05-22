package cmd

import "time"

type ctx struct{}

func (c ctx) Deadline() (time.Time, bool) {
	return time.Now().Add(2), false
}

func (c ctx) Done() <-chan struct{} {
	return nil
}

func (c ctx) Err() error {
	return nil
}

func (c ctx) Value(key any) any {
	return nil
}
