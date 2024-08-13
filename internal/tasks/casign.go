package tasks

import (
	"context"

	"github.com/flightctl/flightctl/internal/store"
	"github.com/flightctl/flightctl/internal/store/model"
    "github.com/flightctl/flightctl/internal/crypto"
	"github.com/sirupsen/logrus"
    "gorm.io/gorm"
)

type ProcessCSR interface {
    RunBatch(batch_size int) error
    InitialMigration() error
}

type ProcessCSRStore struct {
	db  *gorm.DB
	log logrus.FieldLogger
    ProcessCSR
}

const DefaultBatchSize = 100

// Postgres JSONB query for cert reqs which are yet to be signed
// WHERE Status->'certificate' is not null

func NewProcessCSR(store store.Store, log logrus.FieldLogger) ProcessCSR {
	return &ProcessCSRStore{db: store.CertificateSigningRequest().db, log: log}
}

func (s *ProcessCSRStore) InitialMigration() error {
	res := s.db.AutoMigrate(&model.CertificateSigningRequest{})
    if res != nil {
        return res
    }
    return s.db.AutoMigrate(&model.EnrollmentRequest{})
}

func (s *ProcessCSRStore) RunBatch(batch_size int) error {
    var csrs []  model.CertificateSigningRequest
    s.db.Where("Status->'certificate' is not null").Find(&csrs).Limit(batch_size)
    ca := crypto.GetDefaultCA()
    if ca == nil {
        // CA has not been instantiated yet, we return and let the processing
        // take place on the next attempt
        return nil
    }
    for _, csrData := range csrs {

        csr, err := crypto.ParseCSR(csrData.Spec.Data.Request)
        if err == nil {
            cert, err := ca.IssueRequestedClientCertificate(csr, int(*csrData.Spec.Data.ExpirationSeconds) / (24 * 3600))
            if err == nil {
                csrData.Status.Data.Certificate = &cert
                s.db.Save(csrData)
            }
        }
	}
    return nil
}

func asyncSign(ctx context.Context, resourceRef *ResourceReference, store store.Store, callbackManager CallbackManager, log logrus.FieldLogger) error {

    switch resourceRef.Op {
    case AsyncSignOpSignAll:
        var ret =  asyncSignEnrollment(ctx, resourceRef, store, callbackManager, log)
        if ret != nil {
            return ret
        }
        return asyncSignCSR(ctx, resourceRef, store, callbackManager, log)
    case AsyncSignOpSignCSR:
        return asyncSignCSR(ctx, resourceRef, store, callbackManager, log)
    case AsyncSignOpSignEnrollment:
        return asyncSignEnrollment(ctx, resourceRef, store, callbackManager, log)
    default:
		log.Errorf("asyncSign called with unexpected op %s", resourceRef.Op)
	}
	return nil
}

func asyncSignEnrollment(ctx context.Context, resourceRef *ResourceReference, store store.Store, callbackManager CallbackManager, log logrus.FieldLogger) error {

    // query database, find all unsigned csrs, submit them for signing
    return nil
}

func asyncSignCSR(ctx context.Context, resourceRef *ResourceReference, store store.Store, callbackManager CallbackManager, log logrus.FieldLogger) error {
    processCSR := NewProcessCSR(store, log)
    processCSR.RunBatch(DefaultBatchSize)
    return nil
}

