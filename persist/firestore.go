package persist

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
)

// FireStore is a Store that uses a firestore collection to store cache data
type FireStore struct {
	client *firestore.Client
}

// NewFireStore creates a new FireStore, the client is used to interact with the store.
func NewFireStore(client *firestore.Client) *FireStore {
	return &FireStore{
		client: client,
	}
}

// Get attempts to get the firestore document that matches the provided key. If the document does not
// exist no error will be returned. If the document does exist, it's value and last updated time will be returned
func (s *FireStore) Get(ctx context.Context, key string) ([]byte, time.Time, error) {
	doc := s.client.Doc(SafeKey(key))

	snap, err := doc.Get(ctx)
	if err != nil {
		return nil, time.Time{}, err
	}

	var raw []byte
	err = snap.DataTo(&raw)
	if err != nil {
		return nil, time.Time{}, err
	}

	return raw, snap.UpdateTime, nil
}

// Set attempts to update or creates a firestore document that matches the provided key. In order to ensure the key does
// not contain illegal characters, the key will be converted to a 'safe' key.
func (s *FireStore) Set(ctx context.Context, key string, val []byte) error {
	doc := s.client.Doc(SafeKey(key))

	_, err := doc.Set(ctx, val)
	if err != nil {
		return err
	}

	return nil
}
