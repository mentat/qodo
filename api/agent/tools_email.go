package agent

import (
	"context"
	"strings"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/mentat/qodo/api/seed"
	"github.com/mentat/qodo/api/services"
)

// ─── list_emails ─────────────────────────────────────────────────────────────

type ListEmailsInput struct {
	UnreadOnly bool `json:"unread_only,omitempty" jsonschema:"only return unread inbound mail"`
	Limit      int  `json:"limit,omitempty" jsonschema:"max emails to return (default 25)"`
}

type EmailOut struct {
	ID        string `json:"id"`
	ThreadID  string `json:"thread_id"`
	From      string `json:"from"`
	FromName  string `json:"from_name,omitempty"`
	Subject   string `json:"subject"`
	Snippet   string `json:"snippet,omitempty"`
	Direction string `json:"direction"`
	Read      bool   `json:"read"`
}

type ListEmailsOutput struct {
	Emails []EmailOut `json:"emails"`
	Count  int        `json:"count"`
	Notice string     `json:"notice,omitempty"`
}

func NewListEmailsTool(svc *services.EmailService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in ListEmailsInput) (ListEmailsOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return ListEmailsOutput{Notice: "internal: missing user context"}, nil
		}
		limit := in.Limit
		if limit <= 0 {
			limit = 25
		}
		items, err := svc.ListInbox(context.Background(), uid, limit)
		if err != nil {
			return ListEmailsOutput{Notice: err.Error()}, nil
		}
		out := ListEmailsOutput{Emails: make([]EmailOut, 0, len(items))}
		for _, e := range items {
			if in.UnreadOnly && (e.Read || e.Direction != services.DirectionInbound) {
				continue
			}
			out.Emails = append(out.Emails, EmailOut{
				ID: e.ID, ThreadID: e.ThreadID, From: e.From, FromName: e.FromName,
				Subject: e.Subject, Snippet: snippet(e.Body, 140), Direction: e.Direction, Read: e.Read,
			})
		}
		out.Count = len(out.Emails)
		return out, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "list_emails",
		Description: "List the user's recent emails (newest first). Set unread_only to focus on new inbound mail. Returns id, thread_id, from, subject, and a snippet.",
	}, handler)
}

// ─── read_email ──────────────────────────────────────────────────────────────

type ReadEmailInput struct {
	ID string `json:"id" jsonschema:"the email id from list_emails"`
}

type ReadEmailOutput struct {
	Email *struct {
		ID       string `json:"id"`
		ThreadID string `json:"thread_id"`
		From     string `json:"from"`
		FromName string `json:"from_name,omitempty"`
		Subject  string `json:"subject"`
		Body     string `json:"body"`
	} `json:"email,omitempty"`
	Error string `json:"error,omitempty"`
}

func NewReadEmailTool(svc *services.EmailService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in ReadEmailInput) (ReadEmailOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return ReadEmailOutput{Error: "internal: missing user context"}, nil
		}
		e, err := svc.Get(context.Background(), uid, in.ID)
		if err != nil {
			return ReadEmailOutput{Error: errString(err)}, nil
		}
		out := ReadEmailOutput{}
		out.Email = &struct {
			ID       string `json:"id"`
			ThreadID string `json:"thread_id"`
			From     string `json:"from"`
			FromName string `json:"from_name,omitempty"`
			Subject  string `json:"subject"`
			Body     string `json:"body"`
		}{ID: e.ID, ThreadID: e.ThreadID, From: e.From, FromName: e.FromName, Subject: e.Subject, Body: e.Body}
		return out, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "read_email",
		Description: "Read the full body of one email by id.",
	}, handler)
}

// ─── compose_email ───────────────────────────────────────────────────────────

type ComposeEmailInput struct {
	To      string `json:"to" jsonschema:"recipient: a character's name, id, or email (e.g. 'Capt. Nimbus' or 'nimbus@synthwave.os')"`
	Subject string `json:"subject" jsonschema:"the email subject"`
	Body    string `json:"body" jsonschema:"the email body"`
}

type ComposeEmailOutput struct {
	OK        bool   `json:"ok"`
	ThreadID  string `json:"thread_id,omitempty"`
	Recipient string `json:"recipient,omitempty"`
	Notice    string `json:"notice,omitempty"`
	Error     string `json:"error,omitempty"`
}

func NewComposeEmailTool(svc *services.EmailService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in ComposeEmailInput) (ComposeEmailOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return ComposeEmailOutput{Error: "internal: missing user context"}, nil
		}
		if strings.TrimSpace(in.Body) == "" {
			return ComposeEmailOutput{Error: "body is required"}, nil
		}
		to, charID, toName := in.To, "", ""
		ch, ok := seed.ResolveCharacter(in.To)
		if ok {
			to, charID, toName = ch.Email, ch.ID, ch.Name
		}
		e, err := svc.Send(context.Background(), uid, services.SendInput{
			To: to, ToName: toName, Subject: in.Subject, Body: in.Body, CharacterID: charID,
		})
		if err != nil {
			return ComposeEmailOutput{Error: errString(err)}, nil
		}
		out := ComposeEmailOutput{OK: true, ThreadID: e.ThreadID, Recipient: to}
		if !ok {
			out.Notice = "Recipient isn't a known character, so no auto-reply will arrive."
		}
		return out, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "compose_email",
		Description: "Send a new email. 'to' may be a character's name, id, or address. Mailing a known character (e.g. Dot Matrix, Capt. Nimbus) will prompt an in-character reply shortly.",
	}, handler)
}

// ─── reply_email ─────────────────────────────────────────────────────────────

type ReplyEmailInput struct {
	ThreadID string `json:"thread_id" jsonschema:"the thread_id from list_emails to reply within"`
	Body     string `json:"body" jsonschema:"your reply body"`
}

type ReplyEmailOutput struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	Notice string `json:"notice,omitempty"`
}

func NewReplyEmailTool(svc *services.EmailService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in ReplyEmailInput) (ReplyEmailOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return ReplyEmailOutput{Error: "internal: missing user context"}, nil
		}
		if strings.TrimSpace(in.ThreadID) == "" || strings.TrimSpace(in.Body) == "" {
			return ReplyEmailOutput{Error: "thread_id and body are required"}, nil
		}
		thread, err := svc.ListThread(context.Background(), uid, in.ThreadID)
		if err != nil {
			return ReplyEmailOutput{Error: errString(err)}, nil
		}
		if len(thread) == 0 {
			return ReplyEmailOutput{Error: "thread not found"}, nil
		}
		var charID, to, subject string
		for _, m := range thread {
			if m.CharacterID != "" {
				charID = m.CharacterID
			}
			if m.Direction == services.DirectionInbound && m.From != "" {
				to = m.From
			} else if to == "" && m.To != "" {
				to = m.To
			}
			subject = m.Subject
		}
		_, err = svc.Send(context.Background(), uid, services.SendInput{
			To: to, Subject: replySubject(subject), Body: in.Body, ThreadID: in.ThreadID, CharacterID: charID,
		})
		if err != nil {
			return ReplyEmailOutput{Error: errString(err)}, nil
		}
		out := ReplyEmailOutput{OK: true}
		if charID == "" {
			out.Notice = "Replied, but this thread has no character, so no auto-reply will arrive."
		}
		return out, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "reply_email",
		Description: "Reply within an existing email thread (use the thread_id from list_emails). If the thread is with a character, they'll write back shortly.",
	}, handler)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func snippet(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= n {
		return s
	}
	return strings.TrimSpace(s[:n]) + "…"
}

func replySubject(s string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(s)), "re:") {
		return s
	}
	if strings.TrimSpace(s) == "" {
		return "Re:"
	}
	return "Re: " + s
}
