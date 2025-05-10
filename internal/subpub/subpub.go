package subpub

import (
	"context"
	"sync"
)

type MessageHandler func(msg interface{})

type Subscription interface {
	Unsubscribe()
}

type SubPub interface {
	Subscribe(subject string, cb MessageHandler) (Subscription, error)
	Publish(subject string, msg interface{}) error
	Close(ctx context.Context) error
}

type subscription struct {
	subpub    *subPubImpl
	subject   string
	handler   MessageHandler
	closeChan chan struct{}
}

func (s *subscription) Unsubscribe() {
	s.subpub.unsubscribe(s)
}

type subPubImpl struct {
	mu          sync.RWMutex
	subscribers map[string][]*subscription
	closed      bool
	wg          sync.WaitGroup
}

func NewSubPub() SubPub {
	return &subPubImpl{
		subscribers: make(map[string][]*subscription),
	}
}

func (s *subPubImpl) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, context.Canceled
	}

	sub := &subscription{
		subpub:    s,
		subject:   subject,
		handler:   cb,
		closeChan: make(chan struct{}),
	}

	s.subscribers[subject] = append(s.subscribers[subject], sub)
	return sub, nil

}

func (s *subPubImpl) Publish(subject string, msg interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return context.Canceled
	}

	subs, ok := s.subscribers[subject]
	if !ok {
		return nil
	}

	for _, sub := range subs {
		sub := sub
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			select {
			case <-sub.closeChan:
				return
			default:
				sub.handler(msg)
			}
		}()
	}

	return nil
}

func (s *subPubImpl) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *subPubImpl) unsubscribe(sub *subscription) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subs, ok := s.subscribers[sub.subject]
	if !ok {
		return
	}

	for i := range subs {
		if subs[i] == sub {
			s.subscribers[sub.subject] = append(subs[:i], subs[i+1:]...)
			close(sub.closeChan)
			break
		}
	}

	if len(s.subscribers[sub.subject]) == 0 {
		delete(s.subscribers, sub.subject)
	}
}
