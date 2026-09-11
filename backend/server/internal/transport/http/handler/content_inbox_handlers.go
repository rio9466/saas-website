package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/domain/media"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
	"github.com/rio9466/easy-admin/server/internal/transport/http/response"
)

// --- public contact form ---------------------------------------------------

// Contact accepts a public "contact us" submission and returns 201 with the
// receipt. The response is never cached.
func (h *ContentPublicHandlers) Contact(c *gin.Context) {
	var req publicContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationFailure(c)
		return
	}
	receipt, err := h.svc.SubmitContact(c.Request.Context(), req.toInput(), c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	response.JSON(c, http.StatusCreated, toContactReceiptData(receipt))
}

// MediaFile serves one stored media object by filename.
func (h *ContentPublicHandlers) MediaFile(c *gin.Context) {
	filename := c.Param("filename")
	file, err := h.svc.OpenMedia(filename)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	defer file.Close()
	c.Header("Cache-Control", publicContentCacheControl)
	http.ServeContent(c.Writer, c.Request, filename, file.ModTime(), file)
}

// --- administrator contact inbox -------------------------------------------

// ListContactSubmissions returns a paginated inbox page, newest first.
func (h *ContentAdminHandlers) ListContactSubmissions(c *gin.Context) {
	page, pageSize := pageParams(c)
	result, err := h.svc.ListContactSubmissions(c.Request.Context(), actorFromContext(c), c.Query("status"), page, pageSize)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, pageData{Items: toContactSubmissionList(result.Items), Total: result.Total, Page: result.Page, PageSize: result.PageSize})
}

// GetContactSubmission returns one inbox submission.
func (h *ContentAdminHandlers) GetContactSubmission(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	submission, err := h.svc.GetContactSubmission(c.Request.Context(), actorFromContext(c), id)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toContactSubmissionData(submission))
}

// UpdateContactSubmissionStatus changes only the status of one submission.
func (h *ContentAdminHandlers) UpdateContactSubmissionStatus(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	var req adminContactStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil || normalizeStatus(req.Status) == "" {
		validationFailure(c)
		return
	}
	submission, err := h.svc.UpdateContactSubmissionStatus(c.Request.Context(), actorFromContext(c), id, req.Status)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toContactSubmissionData(submission))
}

// --- administrator media library -------------------------------------------

// UploadMedia accepts one multipart image and returns its metadata.
func (h *ContentAdminHandlers) UploadMedia(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, media.MaxUploadBytes+(1<<20))
	header, err := c.FormFile("file")
	if err != nil {
		validationFailure(c)
		return
	}
	file, err := header.Open()
	if err != nil {
		validationFailure(c)
		return
	}
	defer file.Close()

	asset, err := h.svc.UploadMedia(c.Request.Context(), actorFromContext(c), header.Filename, header.Size, file)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, toMediaAssetData(asset))
}

// ListMedia returns a paginated media page, newest first.
func (h *ContentAdminHandlers) ListMedia(c *gin.Context) {
	page, pageSize := pageParams(c)
	result, err := h.svc.ListMedia(c.Request.Context(), actorFromContext(c), page, pageSize)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, pageData{Items: toMediaAssetList(result.Items), Total: result.Total, Page: result.Page, PageSize: result.PageSize})
}

// DeleteMedia removes one media object and its stored bytes.
func (h *ContentAdminHandlers) DeleteMedia(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil || id <= 0 {
		validationFailure(c)
		return
	}
	if err := h.svc.DeleteMedia(c.Request.Context(), actorFromContext(c), id); err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, map[string]any{})
}
