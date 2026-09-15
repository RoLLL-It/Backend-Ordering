package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"net/http"

	"github.com/google/uuid"

	"github.com/RoLLL-It/Backend-Ordering/internal/http/errs"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/response"
	"github.com/RoLLL-It/Backend-Ordering/internal/platform/storage"
)

var allowedImageTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

type UploadHandler struct {
	s3       *storage.S3Client
	endpoint string
}

func NewUploadHandler(s3 *storage.S3Client, endpoint string) *UploadHandler {
	return &UploadHandler{s3: s3, endpoint: endpoint}
}

// PresignImageUpload issues a short-lived presigned PUT URL for a menu item image.
// The client uploads the file bytes directly to the returned URL, then saves
// the returned public_url against the menu item via the existing update endpoint.
func (h *UploadHandler) PresignImageUpload(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ContentType string `json:"content_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}

	ext, ok := allowedImageTypes[strings.ToLower(body.ContentType)]
	if !ok {
		errs.WriteError(w, r, errs.Validation(map[string]string{
			"content_type": "must be one of: image/jpeg, image/png, image/webp",
		}))
		return
	}

	key := fmt.Sprintf("menu-items/%s-%d.%s", uuid.New().String(), time.Now().Unix(), ext)

	uploadURL, err := h.s3.PresignUpload(r.Context(), key, body.ContentType)
	if err != nil {
		errs.WriteError(w, r, errs.Internal("could not create upload URL"))
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"upload_url": uploadURL,
		"public_url": h.s3.PublicURL(h.endpoint, key),
		"expires_in": 900,
	})
}
