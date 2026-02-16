// Package crm — shared constants for the CRM domain.
// Task 3.8: Extracted to satisfy goconst lint gate.
package crm

// Timeline entity type constants.
const (
	timelineEntityCase    = "case_ticket"
	timelineEntityLead    = "lead"
	timelineEntityDeal    = "deal"
	timelineEntityContact = "contact"
	timelineEntityAccount = "account"
)

// Timeline action constants.
const (
	timelineActionCreated = "created"
	timelineActionUpdated = "updated"
	timelineActionDeleted = "deleted"
)
