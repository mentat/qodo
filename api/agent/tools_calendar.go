package agent

import (
	"context"
	"time"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/mentat/qodo/api/services"
)

// ─── list_events ─────────────────────────────────────────────────────────────

type ListEventsInput struct {
	From string `json:"from,omitempty" jsonschema:"start of range, ISO 8601 (default: 7 days ago)"`
	To   string `json:"to,omitempty" jsonschema:"end of range, ISO 8601 (default: 30 days out)"`
}

type EventOut struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Start    string `json:"start"`
	End      string `json:"end"`
	AllDay   bool   `json:"all_day,omitempty"`
	Location string `json:"location,omitempty"`
}

type ListEventsOutput struct {
	Events []EventOut `json:"events"`
	Count  int        `json:"count"`
	Notice string     `json:"notice,omitempty"`
}

func NewListEventsTool(svc *services.EventService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in ListEventsInput) (ListEventsOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return ListEventsOutput{Notice: "internal: missing user context"}, nil
		}
		from, to := eventRange(in.From, in.To, time.Now().UTC())
		items, err := svc.ListRange(context.Background(), uid, from, to)
		if err != nil {
			return ListEventsOutput{Notice: err.Error()}, nil
		}
		out := ListEventsOutput{Events: make([]EventOut, 0, len(items)), Count: len(items)}
		for _, e := range items {
			out.Events = append(out.Events, toEventOut(e))
		}
		return out, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "list_events",
		Description: "List the user's calendar events in a date range (defaults to the past week through 30 days out). Returns id, title, start, end.",
	}, handler)
}

// ─── create_event ────────────────────────────────────────────────────────────

type CreateEventInput struct {
	Title    string `json:"title" jsonschema:"event title, required"`
	Start    string `json:"start" jsonschema:"ISO 8601 start datetime, required (resolve relative dates first)"`
	End      string `json:"end,omitempty" jsonschema:"ISO 8601 end datetime (default: start + 1 hour)"`
	Location string `json:"location,omitempty"`
	AllDay   bool   `json:"all_day,omitempty"`
}

type CreateEventOutput struct {
	Event *EventOut `json:"event,omitempty"`
	Error string    `json:"error,omitempty"`
}

func NewCreateEventTool(svc *services.EventService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in CreateEventInput) (CreateEventOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return CreateEventOutput{Error: "internal: missing user context"}, nil
		}
		start, err := parseDueDate(in.Start)
		if err != nil || start == nil {
			return CreateEventOutput{Error: "start is required as an ISO 8601 datetime"}, nil
		}
		var end time.Time
		if e, err := parseDueDate(in.End); err == nil && e != nil {
			end = *e
		}
		ev, err := svc.Create(context.Background(), uid, services.EventInput{
			Title: in.Title, Location: in.Location, Start: *start, End: end, AllDay: in.AllDay, Color: "#9B5DE5",
		})
		if err != nil {
			return CreateEventOutput{Error: errString(err)}, nil
		}
		out := toEventOut(ev)
		return CreateEventOutput{Event: &out}, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "create_event",
		Description: "Create a calendar event. Title and start are required; resolve relative dates ('tomorrow 3pm') to ISO 8601 first. End defaults to one hour after start.",
	}, handler)
}

// ─── move_event ──────────────────────────────────────────────────────────────

type MoveEventInput struct {
	ID       string `json:"id" jsonschema:"the event id from list_events"`
	NewStart string `json:"new_start" jsonschema:"ISO 8601 new start datetime"`
	NewEnd   string `json:"new_end,omitempty" jsonschema:"ISO 8601 new end (default: preserve duration)"`
}

type MoveEventOutput struct {
	Event *EventOut `json:"event,omitempty"`
	Error string    `json:"error,omitempty"`
}

func NewMoveEventTool(svc *services.EventService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in MoveEventInput) (MoveEventOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return MoveEventOutput{Error: "internal: missing user context"}, nil
		}
		start, err := parseDueDate(in.NewStart)
		if err != nil || start == nil {
			return MoveEventOutput{Error: "new_start is required as an ISO 8601 datetime"}, nil
		}
		var end time.Time
		if e, err := parseDueDate(in.NewEnd); err == nil && e != nil {
			end = *e
		}
		ev, err := svc.Move(context.Background(), uid, in.ID, *start, end)
		if err != nil {
			return MoveEventOutput{Error: errString(err)}, nil
		}
		out := toEventOut(ev)
		return MoveEventOutput{Event: &out}, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "move_event",
		Description: "Reschedule an event to a new start (and optionally end) time.",
	}, handler)
}

// ─── delete_event ────────────────────────────────────────────────────────────

type DeleteEventInput struct {
	ID string `json:"id" jsonschema:"the event id to delete"`
}

type DeleteEventOutput struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func NewDeleteEventTool(svc *services.EventService) (tool.Tool, error) {
	handler := func(ctx tool.Context, in DeleteEventInput) (DeleteEventOutput, error) {
		uid := UserIDFromContext(ctx)
		if uid == "" {
			return DeleteEventOutput{Error: "internal: missing user context"}, nil
		}
		if err := svc.Delete(context.Background(), uid, in.ID); err != nil {
			return DeleteEventOutput{Error: errString(err)}, nil
		}
		return DeleteEventOutput{OK: true}, nil
	}
	return functiontool.New(functiontool.Config{
		Name:        "delete_event",
		Description: "Delete a calendar event by id.",
	}, handler)
}

// eventRange resolves the from/to args for list_events into an absolute,
// day-inclusive [from, to) window. Defaults to the past week through 30 days
// out. A date-only 'to' (midnight) is extended to end-of-day so afternoon
// events aren't excluded, and an empty/inverted window is widened to a full
// day — so a single-day query (from == to) still matches that day's events.
func eventRange(fromStr, toStr string, now time.Time) (time.Time, time.Time) {
	from := now.AddDate(0, 0, -7)
	to := now.AddDate(0, 0, 30)
	if t, err := parseDueDate(fromStr); err == nil && t != nil {
		from = *t
	}
	if t, err := parseDueDate(toStr); err == nil && t != nil {
		to = *t
		if to.Equal(to.Truncate(24 * time.Hour)) {
			to = to.Add(24 * time.Hour)
		}
	}
	if !to.After(from) {
		to = from.Add(24 * time.Hour)
	}
	return from, to
}

func toEventOut(e services.Event) EventOut {
	return EventOut{
		ID: e.ID, Title: e.Title, Start: e.Start.UTC().Format(time.RFC3339),
		End: e.End.UTC().Format(time.RFC3339), AllDay: e.AllDay, Location: e.Location,
	}
}
