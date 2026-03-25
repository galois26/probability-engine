package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

type s3Client interface {
	PutObject(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error)
}

type Store struct {
	client s3Client
	bucket string
	prefix string
	env    string
	clock  Clock
}

func New(client s3Client, bucket, prefix, env string, clock Clock) *Store {
	if clock == nil {
		clock = realClock{}
	}
	return &Store{
		client: client,
		bucket: bucket,
		prefix: prefix,
		env:    env,
		clock:  clock,
	}
}

func (s *Store) latestKey(dataset string) string {
	return path.Join(s.prefix, s.env, dataset, "latest.json")
}

func (s *Store) runKey(dataset string, t time.Time) string {
	ts := t.UTC().Format("2006-01-02T15-04-05Z")
	return path.Join(s.prefix, s.env, dataset, "runs", ts+".json")
}

func (s *Store) putJSON(ctx context.Context, key string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", key, err)
	}

	_, err = s.client.PutObject(ctx, &awss3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &key,
		Body:        bytes.NewReader(b),
		ContentType: strPtr("application/json"),
	})
	if err != nil {
		return fmt.Errorf("put object %s: %w", key, err)
	}
	return nil
}

func (s *Store) getJSON(ctx context.Context, key string, out any) error {
	resp, err := s.client.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		return fmt.Errorf("get object %s: %w", key, err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read object %s: %w", key, err)
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("unmarshal object %s: %w", key, err)
	}
	return nil
}

func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found") ||
		strings.Contains(strings.ToLower(err.Error()), "nosuchkey")
}
func strPtr(v string) *string { return &v }
