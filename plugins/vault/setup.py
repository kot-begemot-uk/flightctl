#!/usr/bin/python3


'''Setup scripts for using Hashi Vault as a flightctl CA plugin'''

import sys
import json
import hvac
import urllib3  
# or if this does not work with the previous import:
# from requests.packages import urllib3  

# Suppress only the single warning from urllib3.


APPROLE_DEFAULT = {
    "bind_secret_id":            True,
    "secret_id_bound_cidrs":     [],
    "secret_id_num_uses":        10000,
    "secret_id_ttl":             "87600h",
    "token_bound_cidrs":         [],
    "token_explicit_max_ttl":    "0s",
    "token_max_ttl":             "0s",
    "token_no_default_policy":   False,
    "token_num_uses":            0,
    "token_period":              "0s",
    "token_policies":            ["flightctl"],
    "token_ttl":                 "0s",
    "token_type":                "batch",
}

PKIROLE_DEFAULT = {
    'allow_any_name': True,
    'allow_bare_domains': False,
    'allow_glob_domains': False,
    'allow_ip_sans': True,
    'allow_localhost': True,
    'allow_subdomains': False,
    'allow_token_displayname': False,
    'allow_wildcard_certificates': True,
    'allowed_domains': [],
    'allowed_domains_template': False,
    'allowed_other_sans': [],
    'allowed_serial_numbers': [],
    'allowed_uri_sans': [],
    'allowed_uri_sans_template': False,
    'allowed_user_ids': [],
    'basic_constraints_valid_for_non_ca': False,
    'client_flag': True,
    'cn_validations': ['email', 'hostname'],
    'code_signing_flag': False,
    'country': [],
    'email_protection_flag': False,
    'enforce_hostnames': False,
    'ext_key_usage': [],
    'ext_key_usage_oids': [],
    'generate_lease': False,
    'issuer_ref': 'default',
    'key_bits': 2048,
    'key_type': 'rsa',
    'key_usage': ['DigitalSignature', 'KeyAgreement', 'KeyEncipherment'],
    'locality': [],
    'max_ttl': 0,
    'no_store': False,
    'not_after': '',
    'not_before_duration': 30,
    'organization': [],
    'ou': [],
    'policy_identifiers': [],
    'postal_code': [],
    'province': [],
    'require_cn': True,
    'server_flag': True,
    'signature_bits': 256,
    'street_address': [],
    'ttl': 315360000,
    'use_csr_common_name': True,
    'use_csr_sans': True,
    'use_pss': False
}



def load_config():
    '''Load config'''
    config = json.load(sys.stdin)
    try:
        if config["debug"]:
            print(f"Config {config}")
    except KeyError:
        pass
    return config

class FlightctlSetup():
    '''Flightctl Setup'''

    def __init__(self, config):
        self.config = config
        auth = config["auth"]

        aargs = dict()
        aargs["url"] = auth["url"]
        aargs["token"] = auth["root_token"]
        if auth["tls_verify"]:
                aargs["verify"] = auth["server_cert_path"]
                if auth.get("tls_auth") is not None:
                    aargs["cert"] = (auth["client_cert_path"], auth["client_key_path"])
        else:
            aargs["verify"] = False
                    
        self.client = hvac.Client(**aargs)
        if not self.client.is_authenticated:
            raise PermissionError("Failed to authenticate")

    def approle(self):
        '''Setup an approle as needed by flightctl'''

        args = dict()
        for (key, value) in APPROLE_DEFAULT.items():
            if self.config["approle"].get(key) is None:
                args[key] = value
            else:
                args[key] = self.config["approle"][key]

        args["role_name"] = self.config["approle"]["role_name"]

        try:
            if self.config["debug"]:
                print(f"Approle {args}")
        except KeyError:
            pass
        self.client.auth.approle.create_or_update_approle(**args)

    def policy(self):
        '''Setup a policy as needed by flighctl'''
        self.client.sys.list_policies()
        self.client.sys.create_or_update_policy(
            name="flightctl", policy='path "secret/pki/*" { capabilities = ["create", "read","update", "delete", "list"]}',
        )

    def pki_role(self):
        '''Setup a PKI role as needed by flightctl'''
        args = dict()
        for (key, value) in PKIROLE_DEFAULT.items():
            if self.config["pkirole"].get(key) is None:
                args[key] = value
            else:
                args[key] = self.config["pkirole"][key]


        try:
            if self.config["debug"]:
                print(f"Approle {args}")
        except KeyError:
            pass
        self.client.secrets.pki.create_or_update_role(self.config["pkirole"]["role_name"], args)


    def role_id(self):
        '''Generate and return a secret id'''
        resp = self.client.auth.approle.read_role_id(role_name=self.config["approle"]["role_name"])
        return resp["data"]["role_id"]

    def secret_id(self):
        '''Generate and return a secret id'''
        resp = self.client.auth.approle.generate_secret_id(role_name=self.config["approle"]["role_name"])
        return resp["data"]["secret_id"]


setup = FlightctlSetup(load_config())
setup.policy()
setup.approle()
setup.pki_role()
print("=====")
print("Role ID: {}".format(setup.role_id()))
print("Secret ID: {}".format(setup.secret_id()))
print("=====")

