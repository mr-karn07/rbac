package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/casbin/casbin/v2/model"
	"github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
)

type Adapter struct {
	client *opensearch.Client
	index  string
}

func NewAdapter(addresses []string, index string) (*Adapter, error) {
	cfg := opensearch.Config{Addresses: addresses}
	client, err := opensearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	exists, err := indexExists(client, index)
	if err != nil {
		return nil, err
	}
	if !exists {
		err = createIndex(client, index, mapping)
		if err != nil {
			return nil, err
		}
		log.Printf("Index %s created successfully", index)
	}

	return &Adapter{client: client, index: index}, nil
}

func indexExists(client *opensearch.Client, indexName string) (bool, error) {
	req := opensearchapi.IndicesExistsRequest{
		Index: []string{indexName},
	}

	res, err := req.Do(context.Background(), client)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		return true, nil
	} else if res.StatusCode == http.StatusNotFound {
		return false, nil
	}

	bodyBytes, _ := io.ReadAll(res.Body)
	return false, fmt.Errorf("unexpected response code: %d, response: %s", res.StatusCode, string(bodyBytes))
}

func createIndex(client *opensearch.Client, indexName, mapping string) error {
	req := opensearchapi.IndicesCreateRequest{
		Index: indexName,
		Body:  bytes.NewReader([]byte(mapping)),
	}

	res, err := req.Do(context.Background(), client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to create index: %s", string(bodyBytes))
	}

	return nil
}

func (a *Adapter) AddPolicy(sec string, ptype string, rule []string) error {
	policy := Policy{
		PType: ptype,
		V0:    rule[0],
		V1:    rule[1],
	}
	if len(rule) > 2 {
		policy.V2 = rule[2]
	}
	if len(rule) > 3 {
		policy.V3 = rule[3]
	}
	if len(rule) > 4 {
		policy.V4 = rule[4]
	}
	if len(rule) > 5 {
		policy.V5 = rule[5]
	}

	documentID := fmt.Sprintf("%s:%s:%s", policy.V0, policy.V1, policy.V2)
	documentID = strings.ReplaceAll(documentID, "/", "_")

	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("error marshaling policy: %v", err)
	}

	response, err := a.client.Index(
		a.index,
		bytes.NewReader(policyJSON),
		a.client.Index.WithDocumentID(documentID),
		a.client.Index.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("error indexing policy: %v", err)
	}
	defer response.Body.Close()

	if response.IsError() {
		bodyBytes, _ := io.ReadAll(response.Body)
		return fmt.Errorf("error response from OpenSearch: %s", string(bodyBytes))
	}

	return nil
}

func (a *Adapter) LoadPolicy(model model.Model) error {
	log.Println("Starting to load policies from OpenSearch...")

	searchResponse, err := a.client.Search(
		a.client.Search.WithIndex(a.index),
		a.client.Search.WithContext(context.Background()),
		a.client.Search.WithSize(1000),
		a.client.Search.WithScroll(1*time.Minute),
	)
	if err != nil {
		return fmt.Errorf("error performing initial search: %v", err)
	}
	defer searchResponse.Body.Close()

	if searchResponse.IsError() {
		bodyBytes, _ := io.ReadAll(searchResponse.Body)
		return fmt.Errorf("error response from OpenSearch: %s", string(bodyBytes))
	}

	for {
		searchByte, err := io.ReadAll(searchResponse.Body)
		if err != nil {
			return err
		}

		var res map[string]interface{}
		err = json.Unmarshal(searchByte, &res)
		if err != nil {
			return err
		}

		hits := res["hits"].(map[string]interface{})["hits"].([]interface{})
		if len(hits) == 0 {
			log.Println("No more hits, exiting scroll loop.")
			break
		}

		for _, hit := range hits {
			source, ok := hit.(map[string]interface{})["_source"].(map[string]interface{})
			if !ok {
				log.Println("Error parsing source from hit")
				continue
			}

			ptype, ok := source["ptype"].(string)
			if !ok {
				log.Println("Missing or invalid ptype, skipping document")
				continue
			}

			rule := []string{}
			for i := 0; i <= 5; i++ {
				field := fmt.Sprintf("v%d", i)
				if v, ok := source[field].(string); ok {
					rule = append(rule, v)
				} else {
					break
				}
			}

			if len(rule) > 0 {
				sec := "p"
				model[sec][ptype].Policy = append(model[sec][ptype].Policy, rule)
			}
		}

		scrollID, ok := res["_scroll_id"].(string)
		if !ok || scrollID == "" {
			log.Println("No scroll ID found, exiting loop.")
			break
		}

		searchResponse, err = a.client.Scroll(
			a.client.Scroll.WithScrollID(scrollID),
			a.client.Scroll.WithScroll(1*time.Minute),
		)
		if err != nil {
			return fmt.Errorf("error during scroll operation: %v", err)
		}
		defer searchResponse.Body.Close()
	}

	log.Println("Finished loading policies from OpenSearch")
	return nil
}

func (a *Adapter) RemovePolicy(sec string, ptype string, rule []string) error {
	documentID := fmt.Sprintf("%s:%s:%s", rule[0], rule[1], rule[2])
	documentID = strings.ReplaceAll(documentID, "/", "_")

	response, err := a.client.Delete(
		a.index,
		documentID,
		a.client.Delete.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("error deleting policy: %v", err)
	}
	defer response.Body.Close()

	if response.IsError() {
		bodyBytes, _ := io.ReadAll(response.Body)
		return fmt.Errorf("error response from OpenSearch: %s", string(bodyBytes))
	}

	return nil
}

func (a *Adapter) SavePolicy(model model.Model) error {
	if err := a.clearPolicies(); err != nil {
		return fmt.Errorf("error clearing policies: %w", err)
	}

	for section, ptypes := range model {
		for ptype, rules := range ptypes {
			for _, policy := range rules.Policy {
				if err := a.AddPolicy(section, ptype, policy); err != nil {
					return fmt.Errorf("error adding policy (section: %s, ptype: %s, policy: %v): %w", section, ptype, policy, err)
				}
			}
		}
	}

	return nil
}

func (a *Adapter) clearPolicies() error {
	response, err := a.client.DeleteByQuery(
		[]string{a.index},
		strings.NewReader(`{"query": {"match_all": {}}}`),
		a.client.DeleteByQuery.WithRefresh(true),
	)
	if err != nil {
		return fmt.Errorf("error clearing policies: %v", err)
	}
	defer response.Body.Close()

	if response.IsError() {
		bodyBytes, _ := io.ReadAll(response.Body)
		return fmt.Errorf("error response from OpenSearch: %s", string(bodyBytes))
	}

	return nil
}

func (a *Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	// Construct the query for filtering policies
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]interface{}{"ptype": ptype}},
				},
			},
		},
	}

	// Add the field filters to the query based on fieldIndex
	for i, fieldValue := range fieldValues {
		fieldName := fmt.Sprintf("v%d", fieldIndex+i)
		if fieldValue != "" {
			query["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"] = append(
				query["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"].([]map[string]interface{}),
				map[string]interface{}{"term": map[string]interface{}{fieldName: fieldValue}},
			)
		}
	}

	// Convert the query to JSON
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return fmt.Errorf("error marshaling query: %v", err)
	}

	// Execute the delete-by-query request
	response, err := a.client.DeleteByQuery(
		[]string{a.index},
		bytes.NewReader(queryJSON),
		a.client.DeleteByQuery.WithRefresh(true),
	)
	if err != nil {
		return fmt.Errorf("error executing delete by query: %v", err)
	}
	defer response.Body.Close()

	if response.IsError() {
		return fmt.Errorf("error response from OpenSearch: %s", response.String())
	}

	return nil
}
