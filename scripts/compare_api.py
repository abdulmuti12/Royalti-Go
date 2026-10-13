#!/usr/bin/env python3
"""Read-only staging comparison. Tokens are read from environment, never printed."""
import argparse
import datetime as dt
import json
import os
import urllib.error
import urllib.request

PUBLIC = ["check-route", "part-color", "part-category", "part-type-product", "part-brand", "home-banner", "customer-promotion", "customers/get-province", "admins/active-logo"]
ADMIN = ["me", "dashboard", "admin", "role", "customer", "point", "gift", "member_level", "category", "brand", "banner", "promotion", "logo", "transaction", "contact-us", "redeem-point", "gift-claims", "get-orders", "get-customers"]
CUSTOMER = ["me", "get-orders", "member-level", "gifts", "gifts/claims", "get-addresses", "get-address-texts", "transactions"]

def normalize(value, base):
    if isinstance(value, dict):
        return {k: normalize(v, base) for k, v in value.items()}
    if isinstance(value, list):
        return [normalize(v, base) for v in value]
    if isinstance(value, str):
        if value.startswith(base + "/"):
            return "$API" + value[len(base):]
        if len(value) >= 19 and value[4:5] == "-" and value[7:8] == "-":
            try:
                stamp = dt.datetime.fromisoformat(value.replace("Z", "+00:00"))
                if stamp.tzinfo is None:
                    stamp = stamp.replace(tzinfo=dt.timezone.utc)
                return stamp.astimezone(dt.timezone.utc).isoformat()
            except ValueError:
                pass
    return value

def request(base, path, token):
    headers = {"Accept": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    req = urllib.request.Request(base + path, headers=headers, method="GET")
    try:
        response = urllib.request.urlopen(req, timeout=15)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        body = response.read()
        try:
            payload = json.loads(body)
        except (UnicodeDecodeError, json.JSONDecodeError):
            payload = body.decode("utf-8", errors="replace")
        return response.code, normalize(payload, base)

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--laravel", required=True, help="base URL without /api")
    parser.add_argument("--go", required=True, help="base URL without /api")
    args = parser.parse_args()
    old, new = args.laravel.rstrip("/"), args.go.rstrip("/")
    failures, checked = 0, 0
    groups = [("", PUBLIC, "", ""), ("admins/", ADMIN, "LARAVEL_ADMIN_TOKEN", "GO_ADMIN_TOKEN"), ("customers/", CUSTOMER, "LARAVEL_CUSTOMER_TOKEN", "GO_CUSTOMER_TOKEN")]
    for prefix, endpoints, old_env, new_env in groups:
        old_token, new_token = os.getenv(old_env, ""), os.getenv(new_env, "")
        if prefix and not (old_token and new_token):
            print("SKIP " + prefix + " (both staging tokens are required)")
            continue
        for endpoint in endpoints:
            path = "/api/" + prefix + endpoint
            try:
                left, right = request(old, path, old_token), request(new, path, new_token)
                match = left == right
            except (OSError, urllib.error.URLError) as error:
                print("ERROR " + path + ": " + str(type(error).__name__))
                match = False
            checked += 1
            if not match:
                failures += 1
            print(("PASS " if match else "DIFF ") + path)
    print(f"{checked} checked, {failures} differences/errors")
    return 1 if failures else 0

if __name__ == "__main__":
    raise SystemExit(main())
