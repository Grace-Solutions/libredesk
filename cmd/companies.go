package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/company/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

// companyReq is the writable shape of a company. Read-only fields the list query
// derives, such as the parent name and the counts, are deliberately not accepted.
type companyReq struct {
	Name                   string      `json:"name"`
	Description            null.String `json:"description"`
	Website                null.String `json:"website"`
	PhoneNumberCountryCode null.String `json:"phone_number_country_code"`
	PhoneNumber            null.String `json:"phone_number"`
	Country                null.String `json:"country"`
	ParentID               null.Int    `json:"parent_id"`
}

func (c companyReq) toCompany() models.Company {
	return models.Company{
		Name:                   c.Name,
		Description:            c.Description,
		Website:                c.Website,
		PhoneNumberCountryCode: c.PhoneNumberCountryCode,
		PhoneNumber:            c.PhoneNumber,
		Country:                c.Country,
		ParentID:               c.ParentID,
	}
}

// handleGetCompanies returns a paginated list of companies, optionally narrowed to the
// children of one company so a tree can be expanded a level at a time.
func handleGetCompanies(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		query       = string(r.RequestCtx.QueryArgs().Peek("q"))
		parentID, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("parent_id")))
		total       = 0
	)
	if parentID < 0 {
		parentID = 0
	}
	page, pageSize := getPagination(r)
	companies, err := app.company.GetAll(query, parentID, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(companies) > 0 {
		total = companies[0].Total
	}
	return r.SendEnvelope(envelope.PageResults{
		Results:    companies,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetCompaniesCompact returns companies as id and name, all of them without page params.
func handleGetCompaniesCompact(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		query = string(r.RequestCtx.QueryArgs().Peek("q"))
	)
	if ids := getIDsParam(r, "ids"); len(ids) > 0 {
		companies, err := app.company.GetAllCompactByIDs(ids)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		return r.SendEnvelope(companies)
	}
	page, pageSize := getOptionalPagination(r)
	companies, err := app.company.GetAllCompact(query, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(companies)
}

// handleGetCompany returns a single company.
func handleGetCompany(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	company, err := app.company.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(company)
}

// handleCreateCompany creates a company.
func handleCreateCompany(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = companyReq{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	company, err := app.company.Create(req.toCompany())
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(company)
}

// handleUpdateCompany updates a company.
func handleUpdateCompany(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		req   = companyReq{}
	)
	if id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	company, err := app.company.Update(id, req.toCompany())
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(company)
}

// handleDeleteCompany deletes a company, leaving its contacts and child companies in place.
func handleDeleteCompany(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := app.company.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
