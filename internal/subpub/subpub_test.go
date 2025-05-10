package subpub

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSubscribePublish(t *testing.T) {
	sp := NewSubPub()
	defer sp.Close(context.Background())

	var wg sync.WaitGroup
	var recieved bool

	handler := func(msg interface{}) {
		defer wg.Done()
		if msg.(string) != "test message" {
			t.Errorf("Unexpected message: %v", msg)
		}
		recieved = true
	}

	wg.Add(1)
	sub, err := sp.Subscribe("test", handler)
	if err != nil {
		t.Fatalf("Subscrive failed: %v", err)
	}
	defer sub.Unsubscribe()

	if err := sp.Publish("test", "test message"); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	wg.Wait()
	if !recieved {
		t.Error("Message was not recieved")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	sp := NewSubPub()
	defer sp.Close(context.Background())

	const subscribersCount = 5
	var wg sync.WaitGroup
	var counter int
	var mu sync.Mutex

	handler := func(msg interface{}) {
		defer wg.Done()
		mu.Lock()
		counter++
		mu.Unlock()
	}

	wg.Add(subscribersCount)
	for i := 0; i < subscribersCount; i++ {
		sub, err := sp.Subscribe("test", handler)
		if err != nil {
			t.Fatalf("Subscribe failed: %v", err)
		}
		defer sub.Unsubscribe()
	}

	if err := sp.Publish("test", "message"); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	wg.Wait()
	if counter != subscribersCount {
		t.Errorf("Expected %d messages received, got %d", subscribersCount, counter)
	}
}

func TestUnsubscribe(t *testing.T) {
	sp := NewSubPub()
	defer sp.Close(context.Background())

	var received bool
	handler := func(msg interface{}) {
		received = true
	}

	sub, err := sp.Subscribe("test", handler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	sub.Unsubscribe()

	if err := sp.Publish("test", "message"); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	if received {
		t.Error("Handler was called after unsubscribe")
	}
}

func TestClose(t *testing.T) {
	sp := NewSubPub()

	_, err := sp.Subscribe("test", func(msg interface{}) {})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := sp.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	_, err = sp.Subscribe("test", func(msg interface{}) {})
	if err != context.Canceled {
		t.Errorf("Expected Canceled error after close, got %v", err)
	}
}

func TestSlowSubscriber(t *testing.T) {
	sp := NewSubPub()
	defer sp.Close(context.Background())

	var fastDone, slowDone bool
	var wg sync.WaitGroup

	wg.Add(2)

	// Быстрый подписчик
	sp.Subscribe("test", func(msg interface{}) {
		defer wg.Done()
		fastDone = true
	})

	// Медленный подписчик
	sp.Subscribe("test", func(msg interface{}) {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)
		slowDone = true
	})

	sp.Publish("test", "message")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if !fastDone || !slowDone {
			t.Error("Not all handlers completed")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Test timed out")
	}
}
