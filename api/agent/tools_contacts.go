package agent

import (
	"context"
	"strings"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/mentat/qodo/api/services"
)

// ─── list_contacts ───────────────────────────────────────────────────────────

type ListContactsInput struct {
	Search string `json:"search,omitempty" jsonschema:"optional full-text search over name/company/email"`
}

type ContactOut struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email,omitempty"`
	Company string `json:"company,omitempty"`
}

type ListContactsOutput struct {
	Contacts []ContactOut `json:"contacts"`
	Count    int          `json:"count"`
	Notice   string       `json:"notice,omitempty"`
}

func NewListContactsTool(svc *services.ContactService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in ListContactsInput) (ListContactsOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return ListContactsOutput{Notice: "internal: missing user context"}, nil
		}
		var items []services.Contact
		var err error
		if strings.TrimSpace(in.Search) != "" {
			items, err = svc.Search(context.Background(), uid, in.Search, 0)
		} else {
			items, err = svc.List(context.Background(), uid)
		}
		if err != nil {
			return ListContactsOutput{Notice: err.Error()}, nil
		}
		out := ListContactsOutput{Contacts: make([]ContactOut, 0, len(items)), Count: len(items)}
		for _, c := range items {
			out.Contacts = append(out.Contacts, ContactOut{ID: c.ID, Name: c.Name, Email: c.Email, Company: c.Company})
		}
		return out, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "list_contacts",
		Description: "List the user's contacts, or search them by name/company/email. The email characters appear here.",
	}, handler)
}

// ─── get_contact ─────────────────────────────────────────────────────────────

type GetContactInput struct {
	ID string `json:"id" jsonschema:"the contact id from list_contacts"`
}

type GetContactOutput struct {
	Contact *struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email,omitempty"`
		Phone   string `json:"phone,omitempty"`
		Company string `json:"company,omitempty"`
		Notes   string `json:"notes,omitempty"`
	} `json:"contact,omitempty"`
	Error string `json:"error,omitempty"`
}

func NewGetContactTool(svc *services.ContactService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in GetContactInput) (GetContactOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return GetContactOutput{Error: "internal: missing user context"}, nil
		}
		c, err := svc.Get(context.Background(), uid, in.ID)
		if err != nil {
			return GetContactOutput{Error: errString(err)}, nil
		}
		out := GetContactOutput{}
		out.Contact = &struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Email   string `json:"email,omitempty"`
			Phone   string `json:"phone,omitempty"`
			Company string `json:"company,omitempty"`
			Notes   string `json:"notes,omitempty"`
		}{ID: c.ID, Name: c.Name, Email: c.Email, Phone: c.Phone, Company: c.Company, Notes: c.Notes}
		return out, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "get_contact",
		Description: "Get full details for one contact by id.",
	}, handler)
}

// ─── create_contact ──────────────────────────────────────────────────────────

type CreateContactInput struct {
	Name    string `json:"name" jsonschema:"contact name, required"`
	Email   string `json:"email,omitempty"`
	Phone   string `json:"phone,omitempty"`
	Company string `json:"company,omitempty"`
}

type CreateContactOutput struct {
	Contact *ContactOut `json:"contact,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func NewCreateContactTool(svc *services.ContactService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in CreateContactInput) (CreateContactOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return CreateContactOutput{Error: "internal: missing user context"}, nil
		}
		c, err := svc.Create(context.Background(), uid, services.ContactInput{
			Name: in.Name, Email: in.Email, Phone: in.Phone, Company: in.Company,
		})
		if err != nil {
			return CreateContactOutput{Error: errString(err)}, nil
		}
		return CreateContactOutput{Contact: &ContactOut{ID: c.ID, Name: c.Name, Email: c.Email, Company: c.Company}}, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "create_contact",
		Description: "Add a new contact to the user's address book.",
	}, handler)
}
