package tasks

import (
	"context"

	"github.com/flightctl/flightctl/internal/store"
	"github.com/sirupsen/logrus"
)

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

    // query database, find all unsigned enrollment reqs, submit them for signing
    return nil
}

