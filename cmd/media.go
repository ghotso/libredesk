package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"slices"

	"github.com/ghotso/libredesk/internal/attachment"
	amodels "github.com/ghotso/libredesk/internal/auth/models"
	"github.com/ghotso/libredesk/internal/envelope"
	"github.com/ghotso/libredesk/internal/image"
	mmodels "github.com/ghotso/libredesk/internal/media/models"
	"github.com/ghotso/libredesk/internal/stringutil"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

// handleMediaUpload handles media uploads.
func handleMediaUpload(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		cleanUp = false
	)

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		app.lo.Error("error parsing form data.", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.GeneralError)
	}

	files, ok := form.File["files"]
	if !ok || len(files) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.file}"), nil, envelope.InputError)
	}

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		app.lo.Error("error reading uploaded file", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorReading", "name", "{globals.terms.file}"), nil, envelope.GeneralError)
	}
	defer file.Close()

	// Inline?
	var disposition = null.StringFrom(attachment.DispositionAttachment)
	inline, ok := form.Value["inline"]
	if ok && len(inline) > 0 && inline[0] == "true" {
		disposition = null.StringFrom(attachment.DispositionInline)
	}

	// Linked model?
	var linkedModel string
	model, ok := form.Value["linked_model"]
	if ok && len(model) > 0 {
		linkedModel = model[0]
	}

	// Sanitize filename.
	srcFileName := stringutil.SanitizeFilename(fileHeader.Filename)
	srcContentType := fileHeader.Header.Get("Content-Type")
	srcFileSize := fileHeader.Size
	srcExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(srcFileName)), ".")

	// Check if file is empty
	if srcFileSize == 0 {
		app.lo.Error("error: uploaded file is empty (0 bytes)")
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("media.fileEmpty"), nil, envelope.InputError)
	}

	// Check file size
	consts := app.consts.Load().(*constants)
	if bytesToMegabytes(srcFileSize) > float64(consts.MaxFileUploadSizeMB) {
		app.lo.Error("error: uploaded file size is larger than max allowed", "size", bytesToMegabytes(srcFileSize), "max_allowed", consts.MaxFileUploadSizeMB)
		return r.SendErrorEnvelope(
			fasthttp.StatusRequestEntityTooLarge,
			app.i18n.Ts("media.fileSizeTooLarge", "size", fmt.Sprintf("%dMB", consts.MaxFileUploadSizeMB)),
			nil,
			envelope.GeneralError,
		)
	}

	if !slices.Contains(consts.AllowedUploadFileExtensions, "*") && !slices.Contains(consts.AllowedUploadFileExtensions, srcExt) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("media.fileTypeNotAllowed"), nil, envelope.InputError)
	}

	// Delete files on any error.
	var uuid = uuid.New()
	thumbName := image.ThumbPrefix + uuid.String()
	defer func() {
		if cleanUp {
			app.media.Delete(uuid.String())
			app.media.Delete(thumbName)
		}
	}()

	// Generate and upload thumbnail and store image dimensions in the media meta.
	var meta = []byte("{}")
	if slices.Contains(image.Exts, srcExt) {
		file.Seek(0, 0)
		thumbFile, err := image.CreateThumb(image.DefThumbSize, file)
		if err != nil {
			app.lo.Error("error creating thumb image", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.thumbnail}"), nil, envelope.GeneralError)
		}
		thumbName, _, err = app.media.Upload(thumbName, srcContentType, thumbFile)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}

		// Store image dimensions in media meta, storing dimensions for image previews in future.
		file.Seek(0, 0)
		width, height, err := image.GetDimensions(file)
		if err != nil {
			cleanUp = true
			app.lo.Error("error getting image dimensions", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorUploading", "name", "{globals.terms.media}"), nil, envelope.GeneralError)
		}
		meta, _ = json.Marshal(map[string]interface{}{
			"width":  width,
			"height": height,
		})
	}

	// Reset ptr.
	file.Seek(0, 0)

	// Override content type after upload (in case it was detected incorrectly).
	_, srcContentType, err = app.media.Upload(uuid.String(), srcContentType, file)
	if err != nil {
		cleanUp = true
		app.lo.Error("error uploading file", "error", err)
		return sendErrorEnvelope(r, err)
	}

	// Insert in DB.
	media, err := app.media.Insert(disposition, srcFileName, srcContentType, "" /**content_id**/, null.NewString(linkedModel, linkedModel != ""), uuid.String(), null.Int{} /**model_id**/, int(srcFileSize), meta)
	if err != nil {
		cleanUp = true
		app.lo.Error("error inserting metadata into database", "error", err)
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(media)
}

// handleServeMedia serves uploaded media.
// Supports both authenticated access (with permission checks) and signed URL access (no permission checks).
// Portal contacts can access media for messages in conversations they can access (owner or org-shared).
func handleServeMedia(r *fastglue.Request) error {
	var (
		app        = r.Context.(*App)
		uuid       = r.RequestCtx.UserValue("uuid").(string)
		authMethod = r.RequestCtx.UserValue("auth_method")
		userType   = r.RequestCtx.UserValue("user_type")
	)

	// If accessed via signed URL, skip permission checks and serve file directly.
	if authMethod == "signed_url" {
		return serveMediaFile(r, app, uuid, nil)
	}

	auser := r.RequestCtx.UserValue("user").(amodels.User)

	// Portal contact: allow only message attachments for conversations the contact can access.
	if userType == "contact" {
		media, err := getMediaByUUID(app, uuid)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if media.Model.String != "messages" {
			return r.SendErrorEnvelope(http.StatusForbidden, app.i18n.Ts("globals.messages.denied", "name", "{globals.terms.permission}"), nil, envelope.PermissionError)
		}
		conv, err := app.conversation.GetConversationByMessageID(media.ModelID.Int)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		orgID, err := portalContactOrgID(app, auser.ID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if !contactCanAccessConversation(conv, auser.ID, orgID) {
			return r.SendErrorEnvelope(http.StatusForbidden, app.i18n.Ts("globals.messages.denied", "name", "{globals.terms.permission}"), nil, envelope.PermissionError)
		}
		return serveMediaFile(r, app, uuid, &media)
	}

	// Agent: full permission check.
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	media, err := getMediaByUUID(app, uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	allowed, err := app.authz.EnforceMediaAccess(user, media.Model.String)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if media.Model.String == "messages" {
		conversation, err := app.conversation.GetConversationByMessageID(media.ModelID.Int)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		allowed, err = app.authz.EnforceConversationAccess(user, conversation)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
	}

	if !allowed {
		return r.SendErrorEnvelope(http.StatusUnauthorized, app.i18n.Ts("globals.messages.denied", "name", "{globals.terms.permission}"), nil, envelope.UnauthorizedError)
	}

	return serveMediaFile(r, app, uuid, &media)
}

// serveMediaFile serves the actual file content based on the storage provider.
// If media is nil, it will be fetched from DB.
func serveMediaFile(r *fastglue.Request, app *App, uuid string, media *mmodels.Media) error {
	// Fetch media metadata from DB if not provided.
	if media == nil {
		m, err := getMediaByUUID(app, uuid)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		media = &m
	}

	consts := app.consts.Load().(*constants)
	switch consts.UploadProvider {
	case "fs":
		disposition := "attachment"

		// Keep certain content types inline.
		if strings.HasPrefix(media.ContentType, "image/") ||
			strings.HasPrefix(media.ContentType, "video/") ||
			media.ContentType == "application/pdf" {
			disposition = "inline"
		}

		r.RequestCtx.Response.Header.Set("Content-Type", media.ContentType)
		r.RequestCtx.Response.Header.Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, media.Filename))

		fasthttp.ServeFile(r.RequestCtx, filepath.Join(ko.String("upload.fs.upload_path"), uuid))
	case "s3":
		r.RequestCtx.Redirect(app.media.GetURL(uuid, media.ContentType, media.Filename), http.StatusFound)
	}
	return nil
}

// bytesToMegabytes converts bytes to megabytes.
func bytesToMegabytes(bytes int64) float64 {
	return float64(bytes) / 1024 / 1024
}

// getMediaByUUID fetches media metadata from DB, handling thumbnail prefix.
func getMediaByUUID(app *App, uuid string) (mmodels.Media, error) {
	return app.media.Get(0, strings.TrimPrefix(uuid, image.ThumbPrefix))
}
