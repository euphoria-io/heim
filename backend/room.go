package backend

import (
	"sync"
	"time"

	"golang.org/x/net/context"
)

type Listing []IdentityView

type Room interface {
	Log

	Join(context.Context, Session) error
	Part(context.Context, Session) error
	Send(context.Context, Session, Message) (Message, error)
	Listing(context.Context) (Listing, error)
}

type memRoom struct {
	sync.Mutex

	name       string
	log        *memLog
	identities map[string]Identity
	live       map[string][]Session
}

func newMemRoom(name string) *memRoom {
	return &memRoom{
		name: name,
		log:  newMemLog(),
	}
}

func (r *memRoom) Latest(ctx context.Context, n int) ([]Message, error) {
	return r.log.Latest(ctx, n)
}

func (r *memRoom) Join(ctx context.Context, session Session) error {
	r.Lock()
	defer r.Unlock()

	if r.identities == nil {
		r.identities = map[string]Identity{}
	}
	if r.live == nil {
		r.live = map[string][]Session{}
	}

	ident := session.Identity()
	id := ident.ID()

	if _, ok := r.identities[id]; !ok {
		r.identities[id] = ident
	}

	r.live[id] = append(r.live[id], session)
	return nil
}

func (r *memRoom) Part(ctx context.Context, session Session) error {
	r.Lock()
	defer r.Unlock()

	ident := session.Identity()
	id := ident.ID()
	live := r.live[id]
	for i, s := range live {
		if s == session {
			copy(live[i:], live[i+1:])
			r.live[id] = live[:len(live)-1]
		}
	}
	if len(r.live[id]) == 0 {
		delete(r.live, id)
		delete(r.identities, id)
	}
	return nil
}

func (r *memRoom) Send(ctx context.Context, session Session, message Message) (Message, error) {
	r.Lock()
	defer r.Unlock()

	msg := Message{
		UnixTime: time.Now().Unix(),
		Sender:   session.Identity().View(),
		Content:  message.Content,
	}
	r.log.post(&msg)
	return msg, r.broadcast(ctx, SendType, msg, session)
}

func (r *memRoom) broadcast(
	ctx context.Context, cmdType CommandType, payload interface{}, excluding ...Session) error {

	for _, sessions := range r.live {
	loop:
		for _, session := range sessions {
			for _, exc := range excluding {
				if exc == session {
					continue loop
				}
			}
			if err := session.Send(ctx, cmdType, payload); err != nil {
				// TODO: accumulate errors
				return err
			}
		}
	}
	return nil
}

func (r *memRoom) Listing(ctx context.Context) (Listing, error) {
	listing := Listing{}
	for _, sessions := range r.live {
		for _, session := range sessions {
			listing = append(listing, *session.Identity().View())
		}
	}
	return listing, nil
}
