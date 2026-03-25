package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestRunStateStore_SaveAndLoadLastRun(t *testing.T) {
	now := time.Date(2026, 3, 22, 18, 0, 0, 0, time.UTC)
	fake := newFakeS3Client()

	store := &Store{
		client: fake,
		bucket: "test-bucket",
		prefix: "probability-engine",
		env:    "dev",
		clock:  fixedClock{now: now},
	}

	saved := time.Date(2026, 3, 22, 17, 45, 0, 0, time.UTC)

	if err := store.SaveLastRun(context.Background(), "probability-engine", saved); err != nil {
		t.Fatalf("SaveLastRun() error = %v", err)
	}

	got, err := store.LoadLastRun(context.Background(), "probability-engine")
	if err != nil {
		t.Fatalf("LoadLastRun() error = %v", err)
	}

	if !got.Equal(saved) {
		t.Fatalf("LoadLastRun() = %s, want %s", got, saved)
	}

	key := "probability-engine/dev/state/probability-engine.json"
	raw, ok := fake.objects[key]
	if !ok {
		t.Fatalf("expected object %q to be written", key)
	}

	var payload struct {
		Job       string    `json:"job"`
		LastRun   time.Time `json:"lastRun"`
		UpdatedAt time.Time `json:"updatedAt"`
		Schema    string    `json:"schema"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if payload.Job != "probability-engine" {
		t.Fatalf("Job = %q, want probability-engine", payload.Job)
	}
	if !payload.LastRun.Equal(saved) {
		t.Fatalf("LastRun = %s, want %s", payload.LastRun, saved)
	}
	if !payload.UpdatedAt.Equal(now) {
		t.Fatalf("UpdatedAt = %s, want %s", payload.UpdatedAt, now)
	}
	if payload.Schema != "v1" {
		t.Fatalf("Schema = %q, want v1", payload.Schema)
	}
}

func TestRunStateStore_LoadLastRun_MissingObjectReturnsZeroTime(t *testing.T) {
	store := &Store{
		client: newFakeS3Client(),
		bucket: "test-bucket",
		prefix: "probability-engine",
		env:    "dev",
		clock:  fixedClock{now: time.Now().UTC()},
	}

	got, err := store.LoadLastRun(context.Background(), "missing-job")
	if err != nil {
		t.Fatalf("LoadLastRun() error = %v", err)
	}
	if !got.IsZero() {
		t.Fatalf("LoadLastRun() = %s, want zero time", got)
	}
}

func TestRunStateStore_SaveLastRun_PutError(t *testing.T) {
	fake := newFakeS3Client()
	fake.putErr = errors.New("put failed")

	store := &Store{
		client: fake,
		bucket: "test-bucket",
		prefix: "probability-engine",
		env:    "dev",
		clock:  fixedClock{now: time.Now().UTC()},
	}

	err := store.SaveLastRun(context.Background(), "probability-engine", time.Now().UTC())
	if err == nil {
		t.Fatal("SaveLastRun() error = nil, want non-nil")
	}
}

func TestRunStateStore_LoadLastRun_BadJSON(t *testing.T) {
	fake := newFakeS3Client()
	fake.objects["probability-engine/dev/state/probability-engine.json"] = []byte(`{not-json`)

	store := &Store{
		client: fake,
		bucket: "test-bucket",
		prefix: "probability-engine",
		env:    "dev",
		clock:  fixedClock{now: time.Now().UTC()},
	}

	_, err := store.LoadLastRun(context.Background(), "probability-engine")
	if err == nil {
		t.Fatal("LoadLastRun() error = nil, want non-nil")
	}
}

type fakeS3Client struct {
	objects map[string][]byte
	putErr  error
	getErr  error
}

func newFakeS3Client() *fakeS3Client {
	return &fakeS3Client{
		objects: make(map[string][]byte),
	}
}

func (f *fakeS3Client) PutObject(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error) {
	if f.putErr != nil {
		return nil, f.putErr
	}

	body, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}

	f.objects[*params.Key] = append([]byte(nil), body...)
	return &awss3.PutObjectOutput{}, nil
}

func (f *fakeS3Client) GetObject(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}

	body, ok := f.objects[*params.Key]
	if !ok {
		return nil, errors.New("object not found")
	}

	return &awss3.GetObjectOutput{
		Body: io.NopCloser(bytes.NewReader(body)),
	}, nil
}

type fixedClock struct {
	now time.Time
}

func (f fixedClock) Now() time.Time { return f.now }
