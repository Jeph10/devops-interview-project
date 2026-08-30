#!/usr/bin/env python3
"""Encrypt and set the KUBECONFIG_STAGING GitHub Actions secret."""
import base64
import json
import os
import urllib.request

import nacl.public

repo = os.environ["GH_REPO"]
token = os.environ["GH_TOKEN"]

# 1. Read local kubeconfig
kubeconfig = open(os.path.expanduser("~/.kube/config")).read()
secret_value = base64.b64encode(kubeconfig.encode()).decode()

# 2. Get public key
req = urllib.request.Request(
    f"https://api.github.com/repos/{repo}/actions/secrets/public-key",
    headers={"Authorization": f"Bearer {token}", "Accept": "application/vnd.github+json"},
)
with urllib.request.urlopen(req) as resp:
    pk = json.loads(resp.read())
key_id = pk["key_id"]
public_key = base64.b64decode(pk["key"])

# 3. Encrypt with libsodium sealed box
sealed = nacl.public.SealedBox(nacl.public.PublicKey(public_key)).encrypt(secret_value.encode())
encrypted = base64.b64encode(sealed).decode()

# 4. PUT the secret
payload = json.dumps({"encrypted_value": encrypted, "key_id": key_id}).encode()
req = urllib.request.Request(
    f"https://api.github.com/repos/{repo}/actions/secrets/KUBECONFIG_STAGING",
    data=payload,
    method="PUT",
    headers={"Authorization": f"Bearer {token}", "Accept": "application/vnd.github+json", "Content-Type": "application/json"},
)
with urllib.request.urlopen(req) as resp:
    print("PUT status:", resp.status)
print("KUBECONFIG_STAGING secret set successfully")