// Package client provides typed Go clients to the v2 substrate canonical
// stores. KB20Client targets kb-20-patient-profile (the canonical store
// for actor entities — Resident, Person, Role).
//
// Example:
//
//	c := client.NewKB20Client("http://kb-20:8131")
//	resident, err := c.GetResident(ctx, residentID)
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// KB20Client is a thin REST client over kb-20's /v2 substrate endpoints.
// Construct via NewKB20Client; the zero value is not usable.
type KB20Client struct {
	baseURL string
	http    *http.Client
}

// NewKB20Client returns a client targeting baseURL. The default underlying
// HTTP client is http.DefaultClient; callers wanting timeouts or transport
// customisation should set Client directly after construction.
func NewKB20Client(baseURL string) *KB20Client {
	return &KB20Client{baseURL: baseURL, http: http.DefaultClient}
}

// SetHTTPClient swaps the underlying http.Client (e.g. to apply a timeout
// or an authentication transport).
func (c *KB20Client) SetHTTPClient(h *http.Client) {
	if h != nil {
		c.http = h
	}
}

// ---------------------------------------------------------------------------
// Resident
// ---------------------------------------------------------------------------

func (c *KB20Client) UpsertResident(ctx context.Context, r models.Resident) (*models.Resident, error) {
	return doJSON[models.Resident](ctx, c.http, http.MethodPost, c.baseURL+"/v2/residents", r)
}

func (c *KB20Client) GetResident(ctx context.Context, id uuid.UUID) (*models.Resident, error) {
	return doJSON[models.Resident](ctx, c.http, http.MethodGet, c.baseURL+"/v2/residents/"+id.String(), nil)
}

func (c *KB20Client) ListResidentsByFacility(ctx context.Context, facilityID uuid.UUID) ([]models.Resident, error) {
	out, err := doJSON[[]models.Resident](ctx, c.http, http.MethodGet,
		c.baseURL+"/v2/facilities/"+facilityID.String()+"/residents", nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// ---------------------------------------------------------------------------
// Person
// ---------------------------------------------------------------------------

func (c *KB20Client) UpsertPerson(ctx context.Context, p models.Person) (*models.Person, error) {
	return doJSON[models.Person](ctx, c.http, http.MethodPost, c.baseURL+"/v2/persons", p)
}

func (c *KB20Client) GetPerson(ctx context.Context, id uuid.UUID) (*models.Person, error) {
	return doJSON[models.Person](ctx, c.http, http.MethodGet, c.baseURL+"/v2/persons/"+id.String(), nil)
}

func (c *KB20Client) GetPersonByHPII(ctx context.Context, hpii string) (*models.Person, error) {
	q := url.Values{}
	q.Set("hpii", hpii)
	u := c.baseURL + "/v2/persons?" + q.Encode()
	return doJSON[models.Person](ctx, c.http, http.MethodGet, u, nil)
}

// ---------------------------------------------------------------------------
// Role
// ---------------------------------------------------------------------------

func (c *KB20Client) UpsertRole(ctx context.Context, r models.Role) (*models.Role, error) {
	return doJSON[models.Role](ctx, c.http, http.MethodPost, c.baseURL+"/v2/roles", r)
}

func (c *KB20Client) GetRole(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	return doJSON[models.Role](ctx, c.http, http.MethodGet, c.baseURL+"/v2/roles/"+id.String(), nil)
}

func (c *KB20Client) ListRolesByPerson(ctx context.Context, personID uuid.UUID) ([]models.Role, error) {
	out, err := doJSON[[]models.Role](ctx, c.http, http.MethodGet,
		c.baseURL+"/v2/persons/"+personID.String()+"/roles", nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

func (c *KB20Client) ListActiveRolesByPersonAndFacility(ctx context.Context, personID, facilityID uuid.UUID) ([]models.Role, error) {
	q := url.Values{}
	q.Set("facility_id", facilityID.String())
	u := c.baseURL + "/v2/persons/" + personID.String() + "/active_roles?" + q.Encode()
	out, err := doJSON[[]models.Role](ctx, c.http, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// ---------------------------------------------------------------------------
// MedicineUse
// ---------------------------------------------------------------------------

func (c *KB20Client) UpsertMedicineUse(ctx context.Context, m models.MedicineUse) (*models.MedicineUse, error) {
	return doJSON[models.MedicineUse](ctx, c.http, http.MethodPost, c.baseURL+"/v2/medicine_uses", m)
}

func (c *KB20Client) GetMedicineUse(ctx context.Context, id uuid.UUID) (*models.MedicineUse, error) {
	return doJSON[models.MedicineUse](ctx, c.http, http.MethodGet, c.baseURL+"/v2/medicine_uses/"+id.String(), nil)
}

func (c *KB20Client) ListMedicineUsesByResident(ctx context.Context, residentID uuid.UUID, limit, offset int) ([]models.MedicineUse, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/medicine_uses?" + q.Encode()
	out, err := doJSON[[]models.MedicineUse](ctx, c.http, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// ---------------------------------------------------------------------------
// Observation
// ---------------------------------------------------------------------------

func (c *KB20Client) UpsertObservation(ctx context.Context, o models.Observation) (*models.Observation, error) {
	return doJSON[models.Observation](ctx, c.http, http.MethodPost, c.baseURL+"/v2/observations", o)
}

func (c *KB20Client) GetObservation(ctx context.Context, id uuid.UUID) (*models.Observation, error) {
	return doJSON[models.Observation](ctx, c.http, http.MethodGet, c.baseURL+"/v2/observations/"+id.String(), nil)
}

func (c *KB20Client) ListObservationsByResident(ctx context.Context, residentID uuid.UUID, limit, offset int) ([]models.Observation, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/observations?" + q.Encode()
	out, err := doJSON[[]models.Observation](ctx, c.http, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

func (c *KB20Client) ListObservationsByResidentAndKind(ctx context.Context, residentID uuid.UUID, kind string, limit, offset int) ([]models.Observation, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/observations/" + kind + "?" + q.Encode()
	out, err := doJSON[[]models.Observation](ctx, c.http, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// ---------------------------------------------------------------------------
// Internal helper
// ---------------------------------------------------------------------------

// doJSON marshals body (if non-nil), issues a request, and decodes the
// response into a fresh *T. Non-2xx responses produce an error that
// includes the response body for debuggability.
func doJSON[T any](ctx context.Context, h *http.Client, method, requestURL string, body interface{}) (*T, error) {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal: %w", err)
		}
		buf = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, requestURL, buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := h.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("kb-20 %s %s: status %d: %s", method, requestURL, resp.StatusCode, string(b))
	}
	var out T
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &out, nil
}
