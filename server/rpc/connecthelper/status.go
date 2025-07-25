/*
 * Copyright 2022 The Yorkie Authors. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package connecthelper provides helper functions for connectRPC.
package connecthelper

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/yorkie-team/yorkie/api/converter"
	"github.com/yorkie-team/yorkie/api/types"
	"github.com/yorkie-team/yorkie/internal/metaerrors"
	"github.com/yorkie-team/yorkie/internal/validation"
	"github.com/yorkie-team/yorkie/pkg/document"
	"github.com/yorkie-team/yorkie/pkg/document/key"
	"github.com/yorkie-team/yorkie/pkg/document/time"
	"github.com/yorkie-team/yorkie/pkg/document/yson"
	"github.com/yorkie-team/yorkie/pkg/webhook"
	"github.com/yorkie-team/yorkie/server/backend/database"
	"github.com/yorkie-team/yorkie/server/backend/pubsub"
	"github.com/yorkie-team/yorkie/server/clients"
	"github.com/yorkie-team/yorkie/server/documents"
	"github.com/yorkie-team/yorkie/server/packs"
	"github.com/yorkie-team/yorkie/server/rpc/auth"
)

// errorToConnectCode maps an error to connectRPC status code.
var errorToConnectCode = map[error]connect.Code{
	// InvalidArgument means the request is malformed.
	converter.ErrPackRequired:          connect.CodeInvalidArgument,
	converter.ErrCheckpointRequired:    connect.CodeInvalidArgument,
	converter.ErrUnsupportedDateRange:  connect.CodeInvalidArgument,
	converter.ErrInvalidSchemaKey:      connect.CodeInvalidArgument,
	time.ErrInvalidHexString:           connect.CodeInvalidArgument,
	time.ErrInvalidActorID:             connect.CodeInvalidArgument,
	types.ErrInvalidID:                 connect.CodeInvalidArgument,
	clients.ErrInvalidClientID:         connect.CodeInvalidArgument,
	clients.ErrInvalidClientKey:        connect.CodeInvalidArgument,
	key.ErrInvalidKey:                  connect.CodeInvalidArgument,
	types.ErrEmptyProjectFields:        connect.CodeInvalidArgument,
	yson.ErrInvalidYSON:                connect.CodeInvalidArgument,
	yson.ErrUnsupported:                connect.CodeInvalidArgument,
	document.ErrSchemaValidationFailed: connect.CodeInvalidArgument,

	// NotFound means the requested resource does not exist.
	database.ErrProjectNotFound:  connect.CodeNotFound,
	database.ErrClientNotFound:   connect.CodeNotFound,
	database.ErrDocumentNotFound: connect.CodeNotFound,
	database.ErrUserNotFound:     connect.CodeNotFound,
	database.ErrSchemaNotFound:   connect.CodeNotFound,

	// AlreadyExists means the requested resource already exists.
	database.ErrProjectAlreadyExists:     connect.CodeAlreadyExists,
	database.ErrProjectNameAlreadyExists: connect.CodeAlreadyExists,
	database.ErrUserAlreadyExists:        connect.CodeAlreadyExists,
	pubsub.ErrAlreadyConnected:           connect.CodeAlreadyExists,
	database.ErrSchemaAlreadyExists:      connect.CodeAlreadyExists,

	// FailedPrecondition means the request is rejected because the state of the
	// system is not the desired state.
	database.ErrClientNotActivated:      connect.CodeFailedPrecondition,
	database.ErrDocumentNotAttached:     connect.CodeFailedPrecondition,
	database.ErrDocumentAlreadyAttached: connect.CodeFailedPrecondition,
	database.ErrDocumentAlreadyDetached: connect.CodeFailedPrecondition,
	documents.ErrDocumentAttached:       connect.CodeFailedPrecondition,
	packs.ErrInvalidServerSeq:           connect.CodeFailedPrecondition,
	database.ErrConflictOnUpdate:        connect.CodeFailedPrecondition,
	packs.ErrDocumentNotRemoved:         connect.CodeFailedPrecondition,

	// Unimplemented means the server does not implement the functionality.
	converter.ErrUnsupportedOperation:   connect.CodeUnimplemented,
	converter.ErrUnsupportedElement:     connect.CodeUnimplemented,
	converter.ErrUnsupportedEventType:   connect.CodeUnimplemented,
	converter.ErrUnsupportedValueType:   connect.CodeUnimplemented,
	converter.ErrUnsupportedCounterType: connect.CodeUnimplemented,

	// Unauthenticated means the request does not have valid authentication
	auth.ErrUnauthenticated:        connect.CodeUnauthenticated,
	database.ErrMismatchedPassword: connect.CodeUnauthenticated,

	// Internal means an internal error occurred.
	webhook.ErrUnexpectedStatusCode: connect.CodeInternal,
	webhook.ErrUnexpectedResponse:   connect.CodeInternal,
	webhook.ErrWebhookTimeout:       connect.CodeInternal,
	pubsub.ErrTooManySubscribers:    connect.CodeInternal,

	// PermissionDenied means the request does not have permission for the operation.
	auth.ErrPermissionDenied: connect.CodePermissionDenied,

	// Canceled means the operation was canceled (typically by the caller).
	context.Canceled: connect.CodeCanceled,
}

// errorToCode maps an error to a string representation of the error.
// TODO(hackerwins): We need to add codes by hand for each error. It would be
// better to generate this map automatically.
var errorToCode = map[error]string{
	converter.ErrPackRequired:       "ErrPackRequired",
	converter.ErrCheckpointRequired: "ErrCheckpointRequired",
	converter.ErrInvalidSchemaKey:   "ErrInvalidSchemaKey",
	time.ErrInvalidHexString:        "ErrInvalidHexString",
	time.ErrInvalidActorID:          "ErrInvalidActorID",
	types.ErrInvalidID:              "ErrInvalidID",
	clients.ErrInvalidClientID:      "ErrInvalidClientID",
	clients.ErrInvalidClientKey:     "ErrInvalidClientKey",
	key.ErrInvalidKey:               "ErrInvalidKey",
	types.ErrEmptyProjectFields:     "ErrEmptyProjectFields",

	document.ErrSchemaValidationFailed: "ErrSchemaValidationFailed",

	database.ErrProjectNotFound:  "ErrProjectNotFound",
	database.ErrClientNotFound:   "ErrClientNotFound",
	database.ErrDocumentNotFound: "ErrDocumentNotFound",
	database.ErrUserNotFound:     "ErrUserNotFound",
	database.ErrSchemaNotFound:   "ErrSchemaNotFound",

	database.ErrProjectAlreadyExists:     "ErrProjectAlreadyExists",
	database.ErrProjectNameAlreadyExists: "ErrProjectNameAlreadyExists",
	database.ErrUserAlreadyExists:        "ErrUserAlreadyExists",
	database.ErrSchemaAlreadyExists:      "ErrSchemaAlreadyExists",

	database.ErrClientNotActivated:      "ErrClientNotActivated",
	database.ErrDocumentNotAttached:     "ErrDocumentNotAttached",
	database.ErrDocumentAlreadyAttached: "ErrDocumentAlreadyAttached",
	database.ErrDocumentAlreadyDetached: "ErrDocumentAlreadyDetached",
	documents.ErrDocumentAttached:       "ErrDocumentAttached",
	packs.ErrInvalidServerSeq:           "ErrInvalidServerSeq",
	database.ErrConflictOnUpdate:        "ErrConflictOnUpdate",

	converter.ErrUnsupportedOperation:   "ErrUnsupportedOperation",
	converter.ErrUnsupportedElement:     "ErrUnsupportedElement",
	converter.ErrUnsupportedEventType:   "ErrUnsupportedEventType",
	converter.ErrUnsupportedValueType:   "ErrUnsupportedValueType",
	converter.ErrUnsupportedCounterType: "ErrUnsupportedCounterType",

	auth.ErrPermissionDenied:        "ErrPermissionDenied",
	auth.ErrUnauthenticated:         "ErrUnauthenticated",
	webhook.ErrUnexpectedResponse:   "ErrUnexpectedResponse",
	webhook.ErrUnexpectedStatusCode: "ErrUnexpectedStatusCode",
	webhook.ErrWebhookTimeout:       "ErrWebhookTimeout",
	database.ErrMismatchedPassword:  "ErrMismatchedPassword",

	pubsub.ErrTooManySubscribers: "ErrTooManySubscribers",
	pubsub.ErrAlreadyConnected:   "ErrAlreadyConnected",
}

// CodeOf returns the string representation of the given error.
func CodeOf(err error) string {
	cause := err
	for errors.Unwrap(cause) != nil {
		cause = errors.Unwrap(cause)
	}

	if code, ok := errorToCode[cause]; ok {
		return code
	}

	return ""
}

// errorToConnectError returns connect.Error from the given error.
func errorToConnectError(err error) (*connect.Error, bool) {
	cause := err
	for errors.Unwrap(cause) != nil {
		cause = errors.Unwrap(cause)
	}

	// NOTE(hackerwins): This prevents panic when the cause is an unhashable
	// error.
	var connectCode connect.Code
	var ok bool
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()

	connectCode, ok = errorToConnectCode[cause]
	if !ok {
		return nil, false
	}

	connectErr := connect.NewError(connectCode, err)
	if code, ok := errorToCode[cause]; ok {
		errorInfo := &errdetails.ErrorInfo{
			Metadata: map[string]string{"code": code},
		}
		if detail, detailErr := connect.NewErrorDetail(errorInfo); detailErr == nil {
			connectErr.AddDetail(detail)
		}
	}

	return connectErr, true
}

// metaErrorToConnectError returns connect.Error from the given rich error.
func metaErrorToConnectError(err error) (*connect.Error, bool) {
	var metaErr *metaerrors.MetaError
	if !errors.As(err, &metaErr) {
		return nil, false
	}

	// NOTE(hackerwins): This prevents panic when the cause is an unhashable
	// error.
	var connectCode connect.Code
	var ok bool
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()

	connectCode, ok = errorToConnectCode[metaErr.Err]
	if !ok {
		return nil, false
	}

	connectErr := connect.NewError(connectCode, err)
	if code, ok := errorToCode[metaErr.Err]; ok {
		errorInfo := &errdetails.ErrorInfo{
			Metadata: map[string]string{"code": code},
		}
		for key, value := range metaErr.Metadata {
			errorInfo.Metadata[key] = value
		}
		if detail, detailErr := connect.NewErrorDetail(errorInfo); detailErr == nil {
			connectErr.AddDetail(detail)
		}
	}

	return connectErr, true
}

// formErrorToConnectError returns connect.Error from the given form error.
func formErrorToConnectError(err error) (*connect.Error, bool) {
	var invalidFieldsError *validation.FormError
	if !errors.As(err, &invalidFieldsError) {
		return nil, false
	}

	connectErr := connect.NewError(connect.CodeInvalidArgument, err)
	badRequest, ok := badRequestFromError(err)
	if !ok {
		return connectErr, true
	}
	if detail, err := connect.NewErrorDetail(badRequest); err == nil {
		connectErr.AddDetail(detail)
	}

	return connectErr, true
}

func badRequestFromError(err error) (*errdetails.BadRequest, bool) {
	var invalidFieldsError *validation.FormError
	if !errors.As(err, &invalidFieldsError) {
		return nil, false
	}

	violations := invalidFieldsError.Violations
	br := &errdetails.BadRequest{}
	for _, violation := range violations {
		v := &errdetails.BadRequest_FieldViolation{
			Field:       violation.Field,
			Description: violation.Description,
		}
		br.FieldViolations = append(br.FieldViolations, v)
	}

	return br, true
}

// ToStatusError returns connect.Error from the given logic error. If an error
// occurs while executing logic in API handler, connectRPC connect.error should be
// returned so that the client can know more about the status of the request.
func ToStatusError(err error) error {
	if err == nil {
		return nil
	}

	if err, ok := metaErrorToConnectError(err); ok {
		return err
	}

	if err, ok := errorToConnectError(err); ok {
		return err
	}

	if err, ok := formErrorToConnectError(err); ok {
		return err
	}

	return connect.NewError(connect.CodeInternal, err)
}

// ToRPCCodeString returns a string representation of the given error.
func ToRPCCodeString(err error) string {
	if err == nil {
		return "ok"
	}

	cause := err
	for errors.Unwrap(cause) != nil {
		cause = errors.Unwrap(cause)
	}
	if code, ok := errorToConnectCode[cause]; ok {
		return code.String()
	}

	return connect.CodeInternal.String()
}
