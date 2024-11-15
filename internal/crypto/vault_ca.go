package crypto

import (
	"crypto/x509"
	"encoding/pem"
	"time"
	"context"
	"errors"
	"sync"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
	"github.com/flightctl/flightctl/internal/config"
)

type vaultCA struct {
	Mutex sync.Mutex
	Id string
	cfg *config.HashiCAConfig
	vault *vault.Client
	Config *TLSCertificateConfig
}

func (ca *vaultCA) GetId() string {
	return ca.Id
}

func (ca *vaultCA) GetConfig() *TLSCertificateConfig {
	return ca.Config
}

func (ca *vaultCA) EnsureServerCertificate(certFile, keyFile string, hostnames []string, expireDays int) (*TLSCertificateConfig, bool, error) {
        certConfig, err := GetServerCert(certFile, keyFile, hostnames)
        if err != nil {
		// We do not support autogeneration with Hashi - certs must exist
            return nil, false, err
        }
        return certConfig, false, nil
}

func (ca *vaultCA) MakeAndWriteServerCert(certFile, keyFile string, hostnames []string, expireDays int) (*TLSCertificateConfig, error) {
	return nil, errors.New("Not supported")
}

func (ca *vaultCA) MakeServerCert(hostnames []string, expiryDays int, fns ...CertificateExtensionFunc) (*TLSCertificateConfig, error) {
	return nil, errors.New("Not supported")
}

func (ca *vaultCA) reAuthenticate() error {
	resp, err := ca.vault.Auth.AppRoleLogin(context.Background(), schema.AppRoleLoginRequest{RoleId:ca.cfg.AppRole, SecretId:ca.cfg.SecretID}, vault.WithMountPath(ca.cfg.AppRolePath))
	if err != nil {
		return err
	}
	err = ca.vault.SetToken(resp.Auth.ClientToken)
	return err
}


func (ca *vaultCA) getClient() (*vault.Client, error) {
	ca.Mutex.Lock()
	defer ca.Mutex.Unlock()
	var err error
	if ca.vault == nil {
		ca.vault, err = vault.New(vault.WithAddress(ca.cfg.VaultURL), vault.WithRequestTimeout(30*time.Second),)
		if err != nil {
			return nil, err
		}
		err := ca.reAuthenticate()
		if err != nil {
			return nil, err
		}
	}
	return ca.vault, nil
}


func (ca *vaultCA) EnsureClientCertificate(certFile, keyFile string, subjectName string, expireDays int) (*TLSCertificateConfig, bool, error) {
	certConfig, err := GetClientCertificate(certFile, keyFile, subjectName)
	if err != nil {
		// Autogeneration not supported for this CA type
		return nil, false, err
	}
	return certConfig, false, nil
}

func (ca *vaultCA) MakeClientCertificate(certFile, keyFile string, subject string, expiryDays int) (*TLSCertificateConfig, error) {
	// Autogeneration not supported for this CA type
	return nil, errors.New("Not Supported")
}

func (ca *vaultCA) IssueRequestedClientCertificate(csr *x509.CertificateRequest, expirySeconds int) ([]byte, error) {
	_, err := ca.getClient()
	if err != nil {
		return nil, err
	}

	req := schema.PkiIssuerSignWithRoleRequest{
		Csr:string(pem.EncodeToMemory(&pem.Block{Type:"CERTIFICATE REQUEST", Bytes:csr.Raw,})),
	}
	resp, err := ca.vault.Secrets.PkiIssuerSignWithRole(context.Background(), ca.cfg.Signer, ca.cfg.Role, req, vault.WithMountPath(ca.cfg.CAPath))
	if err != nil {
        // We may need to refresh the approle token
        // Try once, if that does not work, return an error
        err = ca.reAuthenticate()
        if err != nil {
            return nil, err
        }
        resp, err = ca.vault.Secrets.PkiIssuerSignWithRole(context.Background(), ca.cfg.Signer, ca.cfg.Role, req, vault.WithMountPath(ca.cfg.CAPath))
        if err != nil {
            return nil, err
        }
	}
	
	result, _ := pem.Decode([]byte(resp.Data.Certificate))

	if result == nil {
		return nil, errors.New("Failed to decode PEM block")
	}

	return result.Bytes, nil
}

