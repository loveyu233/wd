package wd

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/esutil"
)

var InsEs *CustomEsClient

type CustomEsClient struct {
	*elasticsearch.TypedClient
	bulkIndexer esutil.BulkIndexer
}

type EsConfig struct {
	addresses  []string
	username   string
	password   string
	batchIndex string
}

type WithEsConfig func(*EsConfig)

func WithEsConfigBatchIndex(batchIndex string) WithEsConfig {
	return func(cfg *EsConfig) {
		cfg.batchIndex = batchIndex
	}
}
func WithEsConfigAddresses(addresses ...string) WithEsConfig {
	return func(cfg *EsConfig) {
		cfg.addresses = append([]string(nil), addresses...)
	}
}

func WithEsConfigUsername(username string) WithEsConfig {
	return func(cfg *EsConfig) {
		cfg.username = username
	}
}

func WithEsConfigPassword(password string) WithEsConfig {
	return func(cfg *EsConfig) {
		cfg.password = password
	}
}

func InitEs(opts ...WithEsConfig) error {
	cfg := &EsConfig{
		addresses: []string{"http://localhost:9200"},
		username:  "",
		password:  "",
	}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	client, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Addresses: cfg.addresses,
		Username:  cfg.username,
		Password:  cfg.password,
	})
	if err != nil {
		return err
	}

	bulkIndexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client: client,
		Index:  cfg.batchIndex,
	})
	if err != nil {
		return err
	}

	InsEs = &CustomEsClient{
		TypedClient: client,
		bulkIndexer: bulkIndexer,
	}
	return nil
}

func (c *CustomEsClient) CustomBulkInsertData(data any) error {
	marshal, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.bulkIndexer.Add(context.Background(), esutil.BulkIndexerItem{
		Action: "index",
		Body:   bytes.NewReader(marshal),
	})
}

func (c *CustomEsClient) CustomBulkClose() error {
	return c.bulkIndexer.Close(context.Background())
}

func (c *CustomEsClient) CustomBulkStats() esutil.BulkIndexerStats {
	return c.bulkIndexer.Stats()
}

func (c *CustomEsClient) Write(p []byte) (n int, err error) {
	if err := c.bulkIndexer.Add(context.Background(), esutil.BulkIndexerItem{
		Action: "index",
		Body:   bytes.NewReader(p),
	}); err != nil {
		return 0, err
	}
	return len(p), nil
}
