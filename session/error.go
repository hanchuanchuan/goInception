// Copyright 2013 The ql Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSES/QL-LICENSE file.

// Copyright 2015 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package session

import (
	// "errors"
	"fmt"
)

// SQLError records an error information, from executing SQL.
type SQLError struct {
	Code    int
	Message string
}

// Error prints errors, with a formatted string.
func (e *SQLError) Error() string {
	return e.Message
}

// NewErr generates a SQL error, with an error code and default format specifier defined in MySQLErrName.
func NewErr(errCode int, args ...interface{}) *SQLError {
	e := &SQLError{Code: errCode}
	e.Message = fmt.Sprintf(GetErrorMessage(errCode), args...)
	return e
}

// NewErrf creates a SQL error, with an error code and a format specifier.
func NewErrf(format string, args ...interface{}) *SQLError {
	e := &SQLError{Code: 0}
	e.Message = fmt.Sprintf(format, args...)
	return e
}
