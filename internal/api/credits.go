package api

import (
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TierLimits represents the resource limits for a tier
type TierLimits struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	Pods   int32  `json:"pods"`
	PVCs   int32  `json:"pvcs"`
}

// TierInfo represents the current tier and upgrade path
type TierInfo struct {
	CurrentTier       string      `json:"current_tier"`
	HighestTier       string      `json:"highest_tier"` // Peak tier achieved - never drops below this
	CurrentLimits     TierLimits  `json:"current_limits"`
	NextTier          string      `json:"next_tier,omitempty"`
	NextTierLimits    *TierLimits `json:"next_tier_limits,omitempty"`
	CreditsToNextTier float64     `json:"credits_to_next_tier"`
	CanUpgrade        bool        `json:"can_upgrade"`
}

// CreditBalance represents the organization's credit balance
type CreditBalance struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	Balance        float64   `json:"balance"`
	Currency       string    `json:"currency"`
	UpdatedAt      time.Time `json:"updated_at"`
	Tier           *TierInfo `json:"tier,omitempty"` // Tier information based on credits balance
}

// CreditTransaction represents a credit transaction
type CreditTransaction struct {
	TransactionID   uuid.UUID `json:"transaction_id"`
	OrganizationID  uuid.UUID `json:"organization_id"`
	Amount          float64   `json:"amount"`
	TransactionType string    `json:"transaction_type"`
	Description     string    `json:"description"`
	CreatedAt       time.Time `json:"created_at"`
}

// MachineUsage represents machine usage for billing
type MachineUsage struct {
	UsageID        uuid.UUID `json:"usage_id"`
	MachineID      uuid.UUID `json:"machine_id"`
	MachineName    string    `json:"machine_name"`
	OrganizationID uuid.UUID `json:"organization_id"`
	HoursUsed      float64   `json:"hours_used"`
	Cost           float64   `json:"cost"`
	Status         string    `json:"status"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time,omitempty"`
}

// TopupRequest represents a credit topup request
type TopupRequest struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency,omitempty"`
}

// TopupResponse represents a credit topup response
type TopupResponse struct {
	SessionID  string  `json:"session_id"`
	PaymentURL string  `json:"payment_url"`
	Amount     float64 `json:"amount"`
	Status     string  `json:"status"`
}

// Invoice represents an organization invoice
type Invoice struct {
	InvoiceID      uuid.UUID `json:"invoice_id"`
	InvoiceNumber  string    `json:"invoice_number"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Amount         float64   `json:"amount"`
	Status         string    `json:"status"`
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
	DueDate        time.Time `json:"due_date,omitempty"`
	PaidAt         time.Time `json:"paid_at,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// InvoiceSummary represents invoice summary statistics
type InvoiceSummary struct {
	TotalInvoices int     `json:"total_invoices"`
	TotalAmount   float64 `json:"total_amount"`
	PaidAmount    float64 `json:"paid_amount"`
	PendingAmount float64 `json:"pending_amount"`
	OverdueAmount float64 `json:"overdue_amount"`
}

// GenerateInvoiceRequest represents a request to generate an invoice
type GenerateInvoiceRequest struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// GetCreditBalance gets the credit balance for an organization
func GetCreditBalance(orgID string) (*CreditBalance, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/credits/organizations/%s/balance", orgID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var balance CreditBalance
	if err := json.Unmarshal(data, &balance); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal credit balance: %s", err.Error()), nil)
	}
	return &balance, nil
}

// GetCreditTransactions gets transaction history for an organization
func GetCreditTransactions(orgID string, limit, offset int) ([]CreditTransaction, error) {
	path := fmt.Sprintf("/credits/organizations/%s/transactions", orgID)
	if limit > 0 || offset > 0 {
		path = fmt.Sprintf("%s?limit=%d&offset=%d", path, limit, offset)
	}

	var resp apiResponse
	err := makeRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var transactions []CreditTransaction
	if err := json.Unmarshal(data, &transactions); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal transactions: %s", err.Error()), nil)
	}
	return transactions, nil
}

// GetMachineUsageHistory gets machine usage history for an organization
func GetMachineUsageHistory(orgID string, days int) ([]MachineUsage, error) {
	path := fmt.Sprintf("/credits/organizations/%s/usage", orgID)
	if days > 0 {
		path = fmt.Sprintf("%s?days=%d", path, days)
	}

	var resp apiResponse
	err := makeRequest("GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var usages []MachineUsage
	if err := json.Unmarshal(data, &usages); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal machine usages: %s", err.Error()), nil)
	}
	return usages, nil
}

// InitiateTopup initiates a credit topup for an organization
func InitiateTopup(orgID string, amount float64) (*TopupResponse, error) {
	req := TopupRequest{
		Amount:   amount,
		Currency: "USD",
	}

	var resp apiResponse
	err := makeRequest("POST", fmt.Sprintf("/credits/organizations/%s/topup", orgID), req, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var topupResp TopupResponse
	if err := json.Unmarshal(data, &topupResp); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal topup response: %s", err.Error()), nil)
	}
	return &topupResp, nil
}

// GetInvoices gets all invoices for an organization
func GetInvoices(orgID string) ([]Invoice, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/credits/organizations/%s/invoices", orgID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var invoices []Invoice
	if err := json.Unmarshal(data, &invoices); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal invoices: %s", err.Error()), nil)
	}
	return invoices, nil
}

// GetInvoiceSummary gets invoice summary for an organization
func GetInvoiceSummary(orgID string) (*InvoiceSummary, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/credits/organizations/%s/invoices/summary", orgID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var summary InvoiceSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal invoice summary: %s", err.Error()), nil)
	}
	return &summary, nil
}

// GetInvoiceByID gets a specific invoice by ID
func GetInvoiceByID(orgID, invoiceID string) (*Invoice, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/credits/organizations/%s/invoices/%s", orgID, invoiceID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var invoice Invoice
	if err := json.Unmarshal(data, &invoice); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal invoice: %s", err.Error()), nil)
	}
	return &invoice, nil
}

// DownloadInvoicePDF downloads invoice as PDF bytes
func DownloadInvoicePDF(orgID, invoiceID string) ([]byte, error) {
	var resp apiResponse
	err := makeRequest("GET", fmt.Sprintf("/credits/organizations/%s/invoices/%s/pdf", orgID, invoiceID), nil, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	return data, nil
}

// GenerateInvoice generates a new invoice for a date range
func GenerateInvoice(orgID string, startDate, endDate time.Time) (*Invoice, error) {
	req := GenerateInvoiceRequest{
		StartDate: startDate,
		EndDate:   endDate,
	}

	var resp apiResponse
	err := makeRequest("POST", fmt.Sprintf("/credits/organizations/%s/invoices/generate", orgID), req, &resp)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to marshal response data: %s", err.Error()), nil)
	}

	var invoice Invoice
	if err := json.Unmarshal(data, &invoice); err != nil {
		return nil, utils.NewError(fmt.Sprintf("failed to unmarshal invoice: %s", err.Error()), nil)
	}
	return &invoice, nil
}
