package main

import (
	"slices"
	"strconv"
	"time"

	amodels "github.com/ghotso/libredesk/internal/auth/models"
	authzModels "github.com/ghotso/libredesk/internal/authz/models"
	"github.com/ghotso/libredesk/internal/automation/models"
	cmodels "github.com/ghotso/libredesk/internal/conversation/models"
	smodels "github.com/ghotso/libredesk/internal/conversation/status/models"
	"github.com/ghotso/libredesk/internal/envelope"
	medModels "github.com/ghotso/libredesk/internal/media/models"
	"github.com/ghotso/libredesk/internal/stringutil"
	umodels "github.com/ghotso/libredesk/internal/user/models"
	vmodels "github.com/ghotso/libredesk/internal/view/models"
	wmodels "github.com/ghotso/libredesk/internal/webhook/models"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

type assigneeChangeReq struct {
	AssigneeID int `json:"assignee_id"`
}

type teamAssigneeChangeReq struct {
	AssigneeID int `json:"assignee_id"`
}

type priorityUpdateReq struct {
	Priority string `json:"priority"`
}

type statusUpdateReq struct {
	Status       string `json:"status"`
	StatusID     int    `json:"status_id"`
	SnoozedUntil string `json:"snoozed_until,omitempty"`
}

type tagsUpdateReq struct {
	Tags []string `json:"tags"`
}

type createConversationRequest struct {
	InboxID                int    `json:"inbox_id"`
	AssignedAgentID        int    `json:"agent_id"`
	AssignedTeamID         int    `json:"team_id"`
	Email                  string `json:"contact_email"`
	FirstName              string `json:"first_name"`
	LastName               string `json:"last_name"`
	Subject                string `json:"subject"`
	Content                string `json:"content"`
	Attachments            []int  `json:"attachments"`
	Initiator              string `json:"initiator"`               // "contact" | "agent"
	ShareWithOrganization  *bool  `json:"share_with_organization"` // optional; when true/false overrides contact's default
}

// handleGetAllConversations retrieves all conversations.
func handleGetAllConversations(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		user    = r.RequestCtx.UserValue("user").(amodels.User)
		order   = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters = string(r.RequestCtx.QueryArgs().Peek("filters"))
		total   = 0
	)
	page, pageSize := getPagination(r)

	conversations, err := app.conversation.GetAllConversationsList(user.ID, order, orderBy, filters, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetAssignedConversations retrieves conversations assigned to the current user.
func handleGetAssignedConversations(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		user    = r.RequestCtx.UserValue("user").(amodels.User)
		order   = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters = string(r.RequestCtx.QueryArgs().Peek("filters"))
		total   = 0
	)
	page, pageSize := getPagination(r)
	conversations, err := app.conversation.GetAssignedConversationsList(user.ID, user.ID, order, orderBy, filters, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetUnassignedConversations retrieves unassigned conversations.
func handleGetUnassignedConversations(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		user    = r.RequestCtx.UserValue("user").(amodels.User)
		order   = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters = string(r.RequestCtx.QueryArgs().Peek("filters"))
		total   = 0
	)
	page, pageSize := getPagination(r)

	conversations, err := app.conversation.GetUnassignedConversationsList(user.ID, order, orderBy, filters, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetMentionedConversations retrieves conversations where the current user is mentioned.
func handleGetMentionedConversations(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		user    = r.RequestCtx.UserValue("user").(amodels.User)
		order   = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters = string(r.RequestCtx.QueryArgs().Peek("filters"))
		total   = 0
	)
	page, pageSize := getPagination(r)

	conversations, err := app.conversation.GetMentionedConversationsList(user.ID, order, orderBy, filters, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetViewConversations retrieves conversations for a view.
func handleGetViewConversations(r *fastglue.Request) error {
	var (
		app       = r.Context.(*App)
		auser     = r.RequestCtx.UserValue("user").(amodels.User)
		viewID, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		order     = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy   = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		total     = 0
	)
	page, pageSize := getPagination(r)
	if viewID < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`view_id`"), nil, envelope.InputError)
	}

	// Check if user has access to the view.
	view, err := app.view.Get(viewID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	hasAccess := false
	switch view.Visibility {
	case vmodels.VisibilityUser:
		hasAccess = view.UserID != nil && *view.UserID == auser.ID
	case vmodels.VisibilityAll:
		hasAccess = true
	case vmodels.VisibilityTeam:
		if view.TeamID != nil {
			hasAccess = slices.Contains(user.Teams.IDs(), *view.TeamID)
		}
	}

	if !hasAccess {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("conversation.viewPermissionDenied"), nil, envelope.PermissionError)
	}

	// Prepare lists user has access to based on user permissions, internally this prepares the SQL query.
	lists := []string{}
	hasTeamAll := slices.Contains(user.Permissions, authzModels.PermConversationsReadTeamAll)
	for _, perm := range user.Permissions {
		if perm == authzModels.PermConversationsReadAll {
			// No further lists required as user has access to all conversations.
			lists = []string{cmodels.AllConversations}
			break
		}
		if perm == authzModels.PermConversationsReadUnassigned {
			lists = append(lists, cmodels.UnassignedConversations)
		}
		if perm == authzModels.PermConversationsReadAssigned {
			lists = append(lists, cmodels.AssignedConversations)
		}
		// Skip TeamUnassignedConversations if user has TeamAllConversations (superset).
		if perm == authzModels.PermConversationsReadTeamInbox && !hasTeamAll {
			lists = append(lists, cmodels.TeamUnassignedConversations)
		}
		if perm == authzModels.PermConversationsReadTeamAll {
			lists = append(lists, cmodels.TeamAllConversations)
		}
	}

	// No lists found, user doesn't have access to any conversations.
	if len(lists) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.Ts("globals.messages.denied", "name", "{globals.terms.permission}"), nil, envelope.PermissionError)
	}

	conversations, err := app.conversation.GetViewConversationsList(user.ID, user.ID, user.Teams.IDs(), lists, order, orderBy, string(view.Filters), page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetTeamUnassignedConversations returns conversations assigned to a team but not to any user.
func handleGetTeamUnassignedConversations(r *fastglue.Request) error {
	var (
		app       = r.Context.(*App)
		auser     = r.RequestCtx.UserValue("user").(amodels.User)
		teamIDStr = r.RequestCtx.UserValue("id").(string)
		order     = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy   = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters   = string(r.RequestCtx.QueryArgs().Peek("filters"))
		total     = 0
	)
	page, pageSize := getPagination(r)
	teamID, _ := strconv.Atoi(teamIDStr)
	if teamID < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`team_id`"), nil, envelope.InputError)
	}

	// Check if user belongs to the team.
	exists, err := app.team.UserBelongsToTeam(teamID, auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if !exists {
		return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, app.i18n.T("conversation.notMemberOfTeam"), nil))
	}

	conversations, err := app.conversation.GetTeamUnassignedConversationsList(auser.ID, teamID, order, orderBy, filters, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetConversation retrieves a single conversation by it's UUID.
func handleGetConversation(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	conv, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	prev, _ := app.conversation.GetContactPreviousConversations(conv.ContactID, 10)
	conv.PreviousConversations = filterCurrentPreviousConv(prev, conv.UUID)
	return r.SendEnvelope(conv)
}

// handleUpdateConversationAssigneeLastSeen updates the current user's last seen timestamp for a conversation.
func handleUpdateConversationAssigneeLastSeen(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err = app.conversation.UpdateUserLastSeen(uuid, auser.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleMarkConversationAsUnread marks a conversation as unread for the current user.
func handleMarkConversationAsUnread(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err = app.conversation.MarkAsUnread(uuid, auser.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleGetConversationParticipants retrieves participants of a conversation.
func handleGetConversationParticipants(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	p, err := app.conversation.GetConversationParticipants(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(p)
}

// handleUpdateUserAssignee updates the user assigned to a conversation.
func handleUpdateUserAssignee(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		req   = assigneeChangeReq{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error decoding assignee change request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	conversation, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Already assigned?
	if conversation.AssignedUserID.Int == req.AssigneeID {
		return r.SendEnvelope(true)
	}

	if err := app.conversation.UpdateConversationUserAssignee(uuid, req.AssigneeID, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleUpdateTeamAssignee updates the team assigned to a conversation.
func handleUpdateTeamAssignee(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		req   = teamAssigneeChangeReq{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error decoding team assignee change request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	assigneeID := req.AssigneeID

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	_, err = app.team.Get(assigneeID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	conversation, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Already assigned?
	if conversation.AssignedTeamID.Int == assigneeID {
		return r.SendEnvelope(true)
	}
	if err := app.conversation.UpdateConversationTeamAssignee(uuid, assigneeID, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleUpdateConversationPriority updates the priority of a conversation.
func handleUpdateConversationPriority(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		req   = priorityUpdateReq{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error decoding priority update request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	priority := req.Priority
	if priority == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`priority`"), nil, envelope.InputError)
	}

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.conversation.UpdateConversationPriority(uuid, 0 /**priority_id**/, priority, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleUpdateConversationStatus updates the status of a conversation.
func handleUpdateConversationStatus(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		req   = statusUpdateReq{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error decoding status update request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	statusName := req.Status
	snoozedUntil := req.SnoozedUntil
	statusID := req.StatusID

	// Resolve status by ID or by name (so renamed default statuses work).
	if statusID > 0 {
		s, err := app.status.Get(statusID)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`status_id`"), nil, envelope.InputError)
		}
		statusName = s.Name
	} else if statusName != "" {
		allStatuses, err := app.status.GetAll()
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		for _, s := range allStatuses {
			if s.Name == statusName {
				statusID = s.ID
				break
			}
		}
		if statusID == 0 {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`status`"), nil, envelope.InputError)
		}
	} else {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`status`"), nil, envelope.InputError)
	}

	// Validate snoozed_until when setting Snoozed status (ID 2).
	if statusID == smodels.DefaultStatusIDSnoozed {
		if snoozedUntil == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`snoozed_until`"), nil, envelope.InputError)
		}
		if _, err := time.ParseDuration(snoozedUntil); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`snoozed_until`"), nil, envelope.InputError)
		}
	}

	// Enforce conversation access.
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	conversation, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Make sure a user is assigned before resolving conversation (resolved = default status ID 3).
	if statusID == smodels.DefaultStatusIDResolved && conversation.AssignedUserID.Int == 0 {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("conversation.resolveWithoutAssignee"), nil))
	}

	// Update conversation status.
	if err := app.conversation.UpdateConversationStatus(uuid, statusID, "", snoozedUntil, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// If status is Resolved (ID 3), send CSAT survey if enabled on inbox.
	if statusID == smodels.DefaultStatusIDResolved {
		// Check if CSAT is enabled on the inbox and send CSAT survey message.
		inbox, err := app.inbox.GetDBRecord(conversation.InboxID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if inbox.CSATEnabled {
			if err := app.conversation.SendCSATReply(user.ID, *conversation); err != nil {
				return sendErrorEnvelope(r, err)
			}
		}
	}
	return r.SendEnvelope(true)
}

type shareWithOrganizationReq struct {
	SharedWithOrganization bool `json:"shared_with_organization"`
}

// handleUpdateConversationShareWithOrganization sets or clears the conversation's organization_id (share with contact's org).
func handleUpdateConversationShareWithOrganization(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		req   = shareWithOrganizationReq{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	auser := r.RequestCtx.UserValue("user").(amodels.User)
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	conv, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	mem, ok, err := app.organization.GetMembershipForContact(int64(conv.ContactID))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	orgID := 0
	if req.SharedWithOrganization && ok {
		orgID = mem.OrganizationID
	}

	if err := app.conversation.UpdateConversationOrganizationID(uuid, orgID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleUpdateConversationtags updates conversation tags.
func handleUpdateConversationtags(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		req   = tagsUpdateReq{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error decoding tags update request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	tagNames := req.Tags

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.conversation.SetConversationTags(uuid, models.ActionSetTags, tagNames, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleUpdateConversationCustomAttributes updates custom attributes of a conversation.
func handleUpdateConversationCustomAttributes(r *fastglue.Request) error {
	var (
		app        = r.Context.(*App)
		attributes = map[string]any{}
		auser      = r.RequestCtx.UserValue("user").(amodels.User)
		uuid       = r.RequestCtx.UserValue("uuid").(string)
	)
	if err := r.Decode(&attributes, ""); err != nil {
		app.lo.Error("error unmarshalling custom attributes JSON", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	// Enforce conversation access.
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Update custom attributes.
	if err := app.conversation.UpdateConversationCustomAttributes(uuid, attributes); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleUpdateContactCustomAttributes updates custom attributes of a contact.
func handleUpdateContactCustomAttributes(r *fastglue.Request) error {
	var (
		app        = r.Context.(*App)
		attributes = map[string]any{}
		auser      = r.RequestCtx.UserValue("user").(amodels.User)
		uuid       = r.RequestCtx.UserValue("uuid").(string)
	)
	if err := r.Decode(&attributes, ""); err != nil {
		app.lo.Error("error unmarshalling custom attributes JSON", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	// Enforce conversation access.
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	conversation, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.user.UpdateCustomAttributes(conversation.ContactID, attributes); err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Broadcast update.
	app.conversation.BroadcastConversationUpdate(conversation.UUID, "contact.custom_attributes", attributes)
	return r.SendEnvelope(true)
}

// enforceConversationAccess fetches the conversation and checks if the user has access to it.
func enforceConversationAccess(app *App, uuid string, user umodels.User) (*cmodels.Conversation, error) {
	conversation, err := app.conversation.GetConversation(0, uuid, "")
	if err != nil {
		return nil, err
	}
	allowed, err := app.authz.EnforceConversationAccess(user, conversation)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, envelope.NewError(envelope.PermissionError, "Permission denied", nil)
	}
	return &conversation, nil
}

// handleRemoveUserAssignee removes the user assigned to a conversation.
func handleRemoveUserAssignee(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err = app.conversation.RemoveConversationAssignee(uuid, "user", user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleRemoveTeamAssignee removes the team assigned to a conversation.
func handleRemoveTeamAssignee(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err = app.conversation.RemoveConversationAssignee(uuid, "team", user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// filterCurrentPreviousConv removes the current conversation from the list of previous conversations.
func filterCurrentPreviousConv(convs []cmodels.PreviousConversation, uuid string) []cmodels.PreviousConversation {
	for i, c := range convs {
		if c.UUID == uuid {
			return append(convs[:i], convs[i+1:]...)
		}
	}
	return []cmodels.PreviousConversation{}
}

// handleCreateConversation creates a new conversation and sends a message to it.
func handleCreateConversation(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		req   = createConversationRequest{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error decoding create conversation request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	// Validate the request
	if err := validateCreateConversationRequest(req, app); err != nil {
		return sendErrorEnvelope(r, err)
	}

	to := []string{req.Email}
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Find or create contact.
	contact := umodels.User{
		Email:           null.StringFrom(req.Email),
		SourceChannelID: null.StringFrom(req.Email),
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		InboxID:         req.InboxID,
	}
	if err := app.user.CreateContact(&contact); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.contact}"), nil))
	}
	// Auto-add contact to organizations whose domain matches contact email.
	addContactToOrganizationsByEmailDomain(app, int64(contact.ID), req.Email)

	// Resolve organization_id for ticket sharing when contact has org membership.
	setOrgID := 0
	mem, hasOrg, _ := app.organization.GetMembershipForContact(int64(contact.ID))
	if hasOrg {
		shareWithOrg := mem.ShareTicketsByDefault
		if req.ShareWithOrganization != nil {
			shareWithOrg = *req.ShareWithOrganization
		}
		if shareWithOrg {
			setOrgID = mem.OrganizationID
		}
	}

	// Create conversation first.
	conversationID, conversationUUID, err := app.conversation.CreateConversation(
		contact.ID,
		contact.ContactChannelID,
		req.InboxID,
		"",
		time.Now(),
		req.Subject,
		true,
		setOrgID,
	)
	if err != nil {
		app.lo.Error("error creating conversation", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.conversation}"), nil))
	}

	// Get media for the attachment ids.
	var media = make([]medModels.Media, 0, len(req.Attachments))
	for _, id := range req.Attachments {
		m, err := app.media.Get(id, "")
		if err != nil {
			app.lo.Error("error fetching media", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.media}"), nil, envelope.GeneralError)
		}
		media = append(media, m)
	}

	// Send initial message based on the initiator of conversation.
	switch req.Initiator {
	case umodels.UserTypeAgent:
		// Queue reply.
		if _, err := app.conversation.QueueReply(media, req.InboxID, auser.ID /**sender_id**/, conversationUUID, req.Content, to, nil /**cc**/, nil /**bcc**/, map[string]any{} /**meta**/); err != nil {
			// Delete the conversation if msg queue fails.
			if err := app.conversation.DeleteConversation(conversationUUID); err != nil {
				app.lo.Error("error deleting conversation", "error", err)
			}
			return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorSending", "name", "{globals.terms.message}"), nil))
		}
	case umodels.UserTypeContact:
		// Create contact message.
		if _, err := app.conversation.CreateContactMessage(media, contact.ID, conversationUUID, req.Content, cmodels.ContentTypeHTML); err != nil {
			// Delete the conversation if message creation fails.
			if err := app.conversation.DeleteConversation(conversationUUID); err != nil {
				app.lo.Error("error deleting conversation", "error", err)
			}
			return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorSending", "name", "{globals.terms.message}"), nil))
		}
	default:
		// Guard anyway.
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`initiator`"), nil, envelope.InputError)
	}

	// Assign the conversation to team/agent if provided, always assign team first as it clears assigned agent.
	if req.AssignedTeamID > 0 {
		app.conversation.UpdateConversationTeamAssignee(conversationUUID, req.AssignedTeamID, user)
	}
	if req.AssignedAgentID > 0 {
		app.conversation.UpdateConversationUserAssignee(conversationUUID, req.AssignedAgentID, user)
	}

	// Trigger webhook event for conversation created.
	conversation, err := app.conversation.GetConversation(conversationID, "", "")
	if err == nil {
		app.webhook.TriggerEvent(wmodels.EventConversationCreated, conversation)
	}

	return r.SendEnvelope(conversation)
}

// validateCreateConversationRequest validates the create conversation request fields.
func validateCreateConversationRequest(req createConversationRequest, app *App) error {
	if req.InboxID <= 0 {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.required", "name", "`inbox_id`"), nil)
	}
	if req.Content == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.required", "name", "`content`"), nil)
	}
	if req.Email == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.required", "name", "`contact_email`"), nil)
	}
	if req.FirstName == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.required", "name", "`first_name`"), nil)
	}
	if !stringutil.ValidEmail(req.Email) {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "`contact_email`"), nil)
	}
	if req.Initiator != umodels.UserTypeContact && req.Initiator != umodels.UserTypeAgent {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "`initiator`"), nil)
	}

	// Check if inbox exists and is enabled.
	inbox, err := app.inbox.GetDBRecord(req.InboxID)
	if err != nil {
		return err
	}
	if !inbox.Enabled {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.disabled", "name", "inbox"), nil)
	}

	return nil
}
