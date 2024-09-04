package tasks

import (
	"context"
	"sync"

	"github.com/flightctl/flightctl/api/v1alpha1"
	"github.com/flightctl/flightctl/internal/store"
	"github.com/flightctl/flightctl/internal/crypto"
	"github.com/sirupsen/logrus"
)

const DefaultBatchSize = 100
const DefaultCSRExpiry = 7 * 24 *60
const DefaultEnrollExpiry = 365 * 24 * 60
var caMutex sync.Mutex

func asyncSign(ctx context.Context, resourceRef *ResourceReference, store store.Store, callbackManager CallbackManager, log logrus.FieldLogger) error {

	// Only one instance is run at any given time

	if !caMutex.TryLock() {
		return nil
	}
	defer caMutex.Unlock()

	// Run repeatedly until there are no records to process.
	// Process DefaultBatchSize at a time

	var count = 1

	for count > 0 {
		switch resourceRef.Op {
		case AsyncSignOpSignAll:
			 count, err := asyncSignEnrollment(ctx, resourceRef, store, callbackManager, log)
			 if err != nil {
			     return err
			 }
			 extra, _ := asyncSignCSR(ctx, resourceRef, store, callbackManager, log)
			 count += extra
		case AsyncSignOpSignCSR:
			 count, _ = asyncSignCSR(ctx, resourceRef, store, callbackManager, log)
		case AsyncSignOpSignEnrollment:
			 count, _ = asyncSignEnrollment(ctx, resourceRef, store, callbackManager, log)
		default:
			log.Errorf("asyncSign called with unexpected op %s", resourceRef.Op)
		}
	}
	return nil
}



func asyncSignEnrollment(ctx context.Context, resourceRef *ResourceReference, dbStore store.Store, callbackManager CallbackManager, log logrus.FieldLogger) (int, error) {

	listParams := store.ListParams{Limit:DefaultBatchSize}

	filterMap, err := store.ConvertFieldFilterParamsToMap([]string{"status.certificate=null", "status.conditions != null"}) // this should never return an err

	if err != nil  {
		return 0, err
	}
	
	listParams.Filter = filterMap
	orgId := store.NullOrgId
	ereqs, err := dbStore.EnrollmentRequest().List(ctx, orgId, listParams)
	if err != nil {
		return 0, err
	}

	ca := crypto.GetDefaultCA()

	count := 0

	for _, ereq := range ereqs.Items {
		if v1alpha1.IsStatusConditionTrue(ereq.Status.Conditions, v1alpha1.EnrollmentRequestApproved) {
			csr, err := crypto.ParseCSR([]byte(ereq.Spec.Csr))
			if err == nil {
				cert, err := ca.IssueRequestedClientCertificate(csr, DefaultEnrollExpiry)
				if err == nil {
					signed := string(cert)
					ereq.Status.Certificate = &signed
					dbStore.EnrollmentRequest().UpdateStatus(ctx, orgId, &ereq)
					count++
				}
			}
		}
	}
	return count, nil

}

func asyncSignCSR(ctx context.Context, resourceRef *ResourceReference, dbStore store.Store, callbackManager CallbackManager, log logrus.FieldLogger) (int, error) {

	listParams := store.ListParams{Limit:DefaultBatchSize}

	filterMap, err := store.ConvertFieldFilterParamsToMap([]string{"status.certificate=null", "status.conditions != null"}) // this should never return an err

	if err != nil  {
		return 0, err
	}
	
	listParams.Filter = filterMap
	orgId := store.NullOrgId
	ereqs, err := dbStore.CertificateSigningRequest().List(ctx, orgId, listParams)
	if err != nil {
		return 0, err
	}

	ca := crypto.GetDefaultCA()
	count := 0

	for _, ereq := range ereqs.Items {
		if v1alpha1.IsStatusConditionTrue(ereq.Status.Conditions, v1alpha1.CertificateSigningRequestApproved) {
			csr, err := crypto.ParseCSR([]byte(ereq.Spec.Request))
			if err == nil {
				expiry := DefaultCSRExpiry
				if ereq.Spec.ExpirationSeconds != nil {
					expiry = int(*ereq.Spec.ExpirationSeconds)
				} 
				cert, err := ca.IssueRequestedClientCertificate(csr, expiry)
				if err == nil {
					ereq.Status.Certificate = &cert
					dbStore.CertificateSigningRequest().Update(ctx, orgId, &ereq)
					count++
				}

			}
		}
	}

	return count, nil

}

