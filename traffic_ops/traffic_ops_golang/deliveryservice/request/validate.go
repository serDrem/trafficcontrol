package request

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/go-ozzo/ozzo-validation"
	"github.com/jmoiron/sqlx"
)

// Validate ensures all required fields are present and in correct form.  Also checks request JSON is complete and valid
func (req *TODeliveryServiceRequest) Validate(db *sqlx.DB) []error {
	tx, err := db.DB.Begin() // must be last, MUST not return an error if this suceeds, without closing the tx
	if err != nil {
		return []error{errors.New("beginning transaction: " + err.Error())}
	}
	commitTx := false
	defer dbhelpers.FinishTx(tx, &commitTx)

	fromStatus := tc.RequestStatusDraft
	if req.ID != nil && *req.ID > 0 {
		err := tx.QueryRow(`SELECT status FROM deliveryservice_request WHERE id=` + strconv.Itoa(*req.ID)).Scan(&fromStatus)
		if err != nil {
			return []error{err}
		}
	}

	validTransition := func(s interface{}) error {
		if s == nil {
			return errors.New("cannot transition to nil status")
		}
		toStatus, ok := s.(*tc.RequestStatus)
		if !ok {
			return fmt.Errorf("Expected *tc.RequestStatus type,  got %T", s)
		}
		return fromStatus.ValidTransition(*toStatus)
	}

	errMap := validation.Errors{
		"changeType":      validation.Validate(req.ChangeType, validation.Required),
		"deliveryservice": validation.Validate(req.DeliveryService, validation.Required),
		"status":          validation.Validate(req.Status, validation.Required, validation.By(validTransition)),
	}
	errs := tovalidate.ToErrors(errMap)
	// ensure the deliveryservice requested is valid
	if err := req.DeliveryService.Validate(tx); err != nil {
		errs = append(errs, err)
	}
	commitTx = true
	return errs
}
